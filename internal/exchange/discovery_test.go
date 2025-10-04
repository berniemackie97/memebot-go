package exchange

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"memebot-go/internal/config"
)

func TestDexScreenerDiscoveryRefreshMergesSymbols(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"pairs": [{
				"chainId": "solana",
				"pairAddress": "ADDR1",
				"baseToken": {"address": "BASE", "name": "Dog Wif Hat", "symbol": "WIF"},
				"quoteToken": {"address": "QUOTE", "name": "Wrapped SOL", "symbol": "SOL"},
				"priceUsd": "0.1",
				"txns": {"m5": {"buys": 10, "sells": 2}},
				"volume": {"h24": 10000},
				"liquidity": {"usd": 15000},
				"priceChange": {"h24": 2.5}
			}]
		}`))
	}))
	defer server.Close()

	feed := NewFeed(ProviderDexScreener, []string{"MANUAL@solana/MANUAL"}, zerolog.Nop())

	discCfg := config.Discovery{
		Enabled:         true,
		Keywords:        []string{"wif"},
		Chains:          []string{"solana"},
		MaxPairs:        5,
		RefreshInterval: 1000,
		MinLiquidityUSD: 5000,
	}
	deConfig := config.DexScreener{BaseURL: server.URL, DefaultChain: "solana"}
	disc := NewDexScreenerDiscovery(zerolog.Nop(), feed, []string{"MANUAL@solana/MANUAL"}, deConfig, discCfg)
	if disc == nil {
		t.Fatalf("expected discovery to be constructed")
	}
	disc.client = server.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := disc.Refresh(ctx); err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}

	symbols := feed.snapshotSymbols()
	if len(symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d: %+v", len(symbols), symbols)
	}
	var hasManual, hasDiscovered bool
	for _, sym := range symbols {
		switch sym {
		case "MANUAL@solana/MANUAL":
			hasManual = true
		case "WIFSOL_ADDR1@solana/ADDR1":
			hasDiscovered = true
		}
	}
	if !hasManual {
		t.Fatalf("manual symbol missing: %+v", symbols)
	}
	if !hasDiscovered {
		t.Fatalf("discovered symbol missing: %+v", symbols)
	}
}

func TestDexScreenerDiscoveryVolumeFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"pairs": [{"chainId": "solana","pairAddress": "ADDRLOW","baseToken": {"address": "BASE", "name": "Low Vol", "symbol": "LOW"},"quoteToken": {"address": "QUOTE", "name": "Wrapped SOL", "symbol": "SOL"},"priceUsd": "0.01","txns": {"m5": {"buys": 1, "sells": 1}},"volume": {"h24": 100},"liquidity": {"usd": 12000},"priceChange": {"h24": 0.5}}]}`))
	}))
	defer server.Close()

	feed := NewFeed(ProviderDexScreener, []string{"MANUAL@solana/MANUAL"}, zerolog.Nop())
	discCfg := config.Discovery{
		Enabled:         true,
		Keywords:        []string{"low"},
		Chains:          []string{"solana"},
		MaxPairs:        5,
		RefreshInterval: 1000,
		MinLiquidityUSD: 500,
		MinVolumeUSD:    5000,
	}
	deConfig := config.DexScreener{BaseURL: server.URL, DefaultChain: "solana"}
	disc := NewDexScreenerDiscovery(zerolog.Nop(), feed, []string{"MANUAL@solana/MANUAL"}, deConfig, discCfg)
	if disc == nil {
		t.Fatalf("expected discovery to be constructed")
	}
	disc.client = server.Client()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := disc.Refresh(ctx); err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}

	symbols := feed.snapshotSymbols()
	if len(symbols) != 1 || symbols[0] != "MANUAL@solana/MANUAL" {
		t.Fatalf("expected only manual symbol, got %+v", symbols)
	}
}
