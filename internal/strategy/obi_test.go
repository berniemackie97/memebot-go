package strategy

import (
	"testing"
	"time"

	"memebot-go/internal/signal"
)

func TestOnTickReturnsSignalLong(t *testing.T) {
	strat := NewOBIMomentum(0.1, 30)
	now := time.Now()
	ticks := []signal.Tick{
		{Symbol: "BTCUSDT", Price: 100, Size: 1, Side: 1, Ts: now.Add(-2 * time.Second)},
		{Symbol: "BTCUSDT", Price: 101, Size: 1, Side: 1, Ts: now.Add(-1 * time.Second)},
		{Symbol: "BTCUSDT", Price: 102, Size: 1, Side: 1, Ts: now},
	}

	var sig *signal.Signal
	for _, tk := range ticks {
		sig = strat.OnTick(tk)
	}
	if sig == nil {
		t.Fatalf("expected long signal")
	}
	if sig.Score <= 0 {
		t.Fatalf("expected positive score, got %.2f", sig.Score)
	}
}

func TestOnTickReturnsSignalShort(t *testing.T) {
	strat := NewOBIMomentum(0.1, 30)
	now := time.Now()
	ticks := []signal.Tick{
		{Symbol: "ETHUSDT", Price: 200, Size: 1, Side: -1, Ts: now.Add(-2 * time.Second)},
		{Symbol: "ETHUSDT", Price: 199, Size: 1, Side: -1, Ts: now.Add(-1 * time.Second)},
		{Symbol: "ETHUSDT", Price: 198, Size: 1, Side: -1, Ts: now},
	}

	var sig *signal.Signal
	for _, tk := range ticks {
		sig = strat.OnTick(tk)
	}
	if sig == nil {
		t.Fatalf("expected short signal")
	}
	if sig.Score >= 0 {
		t.Fatalf("expected negative score, got %.2f", sig.Score)
	}
}

func TestOnTickBelowThreshold(t *testing.T) {
	strat := NewOBIMomentum(0.9, 30)
	now := time.Now()
	tk := signal.Tick{Symbol: "SOLUSDT", Price: 50, Size: 1, Side: 1, Ts: now}
	if sig := strat.OnTick(tk); sig != nil {
		t.Fatalf("expected nil signal when below threshold")
	}
}
