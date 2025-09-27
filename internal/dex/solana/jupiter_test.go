package solana

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	solana "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func TestNewJupiterClientCommit(t *testing.T) {
	wallet := solana.NewWallet()
	client := NewJupiterClient("https://rpc", "https://jup", wallet.PrivateKey, "finalized")
	if client.Commit != rpc.CommitmentFinalized {
		t.Fatalf("expected finalized commitment, got %v", client.Commit)
	}
}

func TestGetQuote(t *testing.T) {
	wallet := solana.NewWallet()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v6/quote" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("inputMint") != "AAA" {
			t.Fatalf("missing inputMint query")
		}
		resp := Quote{InputMint: "AAA", OutputMint: "BBB", InAmount: "10", OutAmount: "20", SlippageBps: 50}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewJupiterClient("https://rpc", server.URL, wallet.PrivateKey, "processed")
	client.Http = server.Client()
	client.Base = server.URL

	quote, err := client.GetQuote(context.Background(), "AAA", "BBB", 10, 50)
	if err != nil {
		t.Fatalf("GetQuote returned error: %v", err)
	}
	if quote.OutAmount != "20" {
		t.Fatalf("expected OutAmount 20, got %s", quote.OutAmount)
	}
}
