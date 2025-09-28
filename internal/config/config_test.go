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
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
}
