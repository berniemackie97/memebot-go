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

type JupiterClient struct {
	Base   string
	RPC    *rpc.Client
	Owner  solana.PrivateKey
	Commit rpc.CommitmentType
	Http   *http.Client
}

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

func NewJupiterClient(rpcURL, base string, owner solana.PrivateKey, commit string) *JupiterClient {
	c := rpc.CommitmentConfirmed
	switch commit {
	case "processed":
		c = rpc.CommitmentProcessed
	case "finalized":
		c = rpc.CommitmentFinalized
	}
	return &JupiterClient{
		Base:   base,
		RPC:    rpc.New(rpcURL),
		Owner:  owner,
		Commit: c,
		Http:   &http.Client{Timeout: 8 * time.Second},
	}
}

// amount is in smallest units (lamports for SOL; token decimals apply).
func (j *JupiterClient) GetQuote(ctx context.Context, inputMint, outputMint string, amount uint64, slippageBps int) (*Quote, error) {
	q := url.Values{}
	q.Set("inputMint", inputMint)
	q.Set("outputMint", outputMint)
	q.Set("amount", fmt.Sprintf("%d", amount))
	q.Set("slippageBps", fmt.Sprintf("%d", slippageBps))
	q.Set("onlyDirectRoutes", "false")
	u := j.Base + "/v6/quote?" + q.Encode()

	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	resp, err := j.Http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("jupiter quote status %d", resp.StatusCode)
	}
	var out Quote
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// BuildAndSendSwap asks Jupiter for a ready-to-sign transaction, signs it locally, then submits via RPC.
func (j *JupiterClient) BuildAndSendSwap(ctx context.Context, quote *Quote) (sig solana.Signature, err error) {
	payload := map[string]any{
		"userPublicKey":             j.Owner.PublicKey().String(),
		"wrapAndUnwrapSol":          true,
		"asLegacyTransaction":       false,
		"useTokenLedger":            false,
		"prioritizationFeeLamports": 0, // tune later
		"quoteResponse":             quote,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", j.Base+"/v6/swap", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := j.Http.Do(req)
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

	// Decode the transaction using the binary decoder
	tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(raw))
	if err != nil {
		return sig, fmt.Errorf("unmarshal tx: %w", err)
	}

	// Sign with our wallet (tx.Sign returns (signatures, error) — ignore the first value)
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(j.Owner.PublicKey()) {
			return &j.Owner
		}
		return nil
	})
	if err != nil {
		return sig, fmt.Errorf("sign: %w", err)
	}

	// Send the signed transaction
	sig, err = j.RPC.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{
		SkipPreflight:       false,
		PreflightCommitment: j.Commit,
	})
	return sig, err
}
