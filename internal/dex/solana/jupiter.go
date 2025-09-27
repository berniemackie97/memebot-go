// Package solana provides Solana-specific exchange connectivity, including Jupiter aggregator access.
package solana

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	bin "github.com/gagliardetto/binary"
	solana "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// JupiterClient orchestrates quote retrieval and swap submission through Jupiter.
type JupiterClient struct {
	Base   string
	RPC    *rpc.Client
	Owner  solana.PrivateKey
	Commit rpc.CommitmentType
	Http   *http.Client
}

// Quote captures the subset of the Jupiter quote response relied on by the executor.
type Quote struct {
	InputMint      string  `json:"inputMint"`
	OutputMint     string  `json:"outputMint"`
	InAmount       string  `json:"inAmount"`
	OutAmount      string  `json:"outAmount"`
	OtherAmount    string  `json:"otherAmountThreshold"`
	SlippageBps    int     `json:"slippageBps"`
	RoutePlan      any     `json:"routePlan"`
	PriceImpactPct float64 `json:"priceImpactPct"`
}

// NewJupiterClient hydrates a JupiterClient by wiring an RPC client, signer, and HTTP settings.
func NewJupiterClient(rpcURL, base string, owner solana.PrivateKey, commit string) *JupiterClient {
	commitment := rpc.CommitmentConfirmed
	switch commit {
	case "processed":
		commitment = rpc.CommitmentProcessed
	case "finalized":
		commitment = rpc.CommitmentFinalized
	}
	return &JupiterClient{
		Base:   base,
		RPC:    rpc.New(rpcURL),
		Owner:  owner,
		Commit: commitment,
		Http:   &http.Client{Timeout: 8 * time.Second},
	}
}

// GetQuote asks the Jupiter REST API for the best route given amount and slippage in basis points.
func (jupiterClient *JupiterClient) GetQuote(ctx context.Context, inputMint, outputMint string, amount uint64, slippageBps int) (*Quote, error) {
	urlValues := url.Values{}
	urlValues.Set("inputMint", inputMint)
	urlValues.Set("outputMint", outputMint)
	urlValues.Set("amount", fmt.Sprintf("%d", amount))
	urlValues.Set("slippageBps", fmt.Sprintf("%d", slippageBps))
	urlValues.Set("onlyDirectRoutes", "false")
	URL := jupiterClient.Base + "/v6/quote?" + urlValues.Encode()

	req, _ := http.NewRequestWithContext(ctx, "GET", URL, nil)
	resp, err := jupiterClient.Http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("jupiter quote status %d", resp.StatusCode)
	}

	var quote Quote
	if err := json.NewDecoder(resp.Body).Decode(&quote); err != nil {
		return nil, err
	}
	return &quote, nil
}

// BuildAndSendSwap asks Jupiter for a ready-to-sign transaction, signs it locally, then submits via RPC.
func (jupiterClient *JupiterClient) BuildAndSendSwap(ctx context.Context, quote *Quote) (sig solana.Signature, err error) {
	payload := map[string]any{
		"userPublicKey":             jupiterClient.Owner.PublicKey().String(),
		"wrapAndUnwrapSol":          true,
		"asLegacyTransaction":       false,
		"useTokenLedger":            false,
		"prioritizationFeeLamports": 0, // tune later
		"quoteResponse":             quote,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", jupiterClient.Base+"/v6/swap", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := jupiterClient.Http.Do(req)
	if err != nil {
		return sig, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return sig, fmt.Errorf("jupiter swap status %d", resp.StatusCode)
	}

	var sr struct {
		SwapTransaction string `json:"swapTransaction"` // base64-encoded tx (unsigned)
	}
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return sig, err
	}

	raw, err := base64.StdEncoding.DecodeString(sr.SwapTransaction)
	if err != nil {
		return sig, fmt.Errorf("decode tx: %w", err)
	}

	// Decode the transaction using the binary decoder.
	transaction, err := solana.TransactionFromDecoder(bin.NewBinDecoder(raw))
	if err != nil {
		return sig, fmt.Errorf("unmarshal tx: %w", err)
	}

	// Sign with our wallet (tx.Sign returns (signatures, error) - ignore the first value).
	_, err = transaction.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(jupiterClient.Owner.PublicKey()) {
			return &jupiterClient.Owner
		}
		return nil
	})
	if err != nil {
		return sig, fmt.Errorf("sign: %w", err)
	}

	// Send the signed transaction.
	sig, err = jupiterClient.RPC.SendTransactionWithOpts(ctx, transaction, rpc.TransactionOpts{
		SkipPreflight:       false,
		PreflightCommitment: jupiterClient.Commit,
	})
	return sig, err
}
