package config

import (
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	path := filepath.Join("testdata", "config.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.App.Name != "memebot-test" {
		t.Fatalf("unexpected App.Name: %s", cfg.App.Name)
	}
	if len(cfg.Exchange.Symbols) != 1 || cfg.Exchange.Symbols[0] != "BTCUSDT" {
		t.Fatalf("expected BTCUSDT symbol, got %+v", cfg.Exchange.Symbols)
	}
	if cfg.Exchange.DexScreener.BaseURL != "https://api.dexscreener.com" {
		t.Fatalf("unexpected DexScreener.BaseURL: %s", cfg.Exchange.DexScreener.BaseURL)
	}
	if cfg.Exchange.DexScreener.DefaultChain != "solana" {
		t.Fatalf("unexpected DexScreener.DefaultChain: %s", cfg.Exchange.DexScreener.DefaultChain)
	}
	if cfg.Exchange.DexScreener.PollInterval != 750 {
		t.Fatalf("unexpected DexScreener.PollInterval: %d", cfg.Exchange.DexScreener.PollInterval)
	}
	if !cfg.Exchange.Discovery.Enabled {
		t.Fatalf("expected discovery enabled")
	}
	if len(cfg.Exchange.Discovery.Keywords) != 1 || cfg.Exchange.Discovery.Keywords[0] != "pepe" {
		t.Fatalf("unexpected discovery keywords: %+v", cfg.Exchange.Discovery.Keywords)
	}
	if cfg.Exchange.Discovery.MaxPairs != 5 {
		t.Fatalf("unexpected discovery max pairs: %d", cfg.Exchange.Discovery.MaxPairs)
	}
	if cfg.Exchange.Discovery.RefreshInterval != 20000 {
		t.Fatalf("unexpected discovery refresh interval: %d", cfg.Exchange.Discovery.RefreshInterval)
	}
	if cfg.Exchange.Discovery.MinLiquidityUSD != 1000 {
		t.Fatalf("unexpected discovery min liquidity: %.2f", cfg.Exchange.Discovery.MinLiquidityUSD)
	}
	if cfg.Exchange.Discovery.MinVolumeUSD != 500 {
		t.Fatalf("unexpected discovery min volume: %.2f", cfg.Exchange.Discovery.MinVolumeUSD)
	}
	if cfg.Exchange.Discovery.MaxPairsPerKeyword != 3 {
		t.Fatalf("unexpected discovery max pairs per keyword: %d", cfg.Exchange.Discovery.MaxPairsPerKeyword)
	}
	if cfg.Strategy.Params.TrendThreshold != 0.05 {
		t.Fatalf("unexpected trend threshold: %.2f", cfg.Strategy.Params.TrendThreshold)
	}
	if cfg.Strategy.Params.TrendWindowSecs != 90 {
		t.Fatalf("unexpected trend window: %d", cfg.Strategy.Params.TrendWindowSecs)
	}
	if cfg.Strategy.Params.TrendMinVolumeUSD != 1000 {
		t.Fatalf("unexpected trend min volume: %.2f", cfg.Strategy.Params.TrendMinVolumeUSD)
	}
	if cfg.Risk.MaxPortfolioNotional != 100 {
		t.Fatalf("unexpected max portfolio notional: %.2f", cfg.Risk.MaxPortfolioNotional)
	}
	if cfg.Dex.Commitment != "processed" {
		t.Fatalf("expected processed commitment, got %s", cfg.Dex.Commitment)
	}
	if cfg.Paper.StartingCash != 5000 {
		t.Fatalf("expected starting cash 5000, got %.2f", cfg.Paper.StartingCash)
	}
	if cfg.Paper.MaxLatencyMs != 50 {
		t.Fatalf("expected max latency 50, got %d", cfg.Paper.MaxLatencyMs)
	}
	if cfg.Paper.SlippageBps != 3 {
		t.Fatalf("expected slippage 3 bps, got %.2f", cfg.Paper.SlippageBps)
	}
	if cfg.Paper.PartialFillProbability != 0.5 {
		t.Fatalf("expected partial fill probability 0.5, got %.2f", cfg.Paper.PartialFillProbability)
	}
	if cfg.Paper.MaxPartialFills != 2 {
		t.Fatalf("expected max partial fills 2, got %d", cfg.Paper.MaxPartialFills)
	}
	if cfg.Paper.MaxPositionNotionalUSD != 200 {
		t.Fatalf("expected max position notional 200, got %.2f", cfg.Paper.MaxPositionNotionalUSD)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
}
