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
	if cfg.Paper.MaxPositionPerSymbol != 0.5 {
		t.Fatalf("expected max position 0.5, got %.2f", cfg.Paper.MaxPositionPerSymbol)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
}
