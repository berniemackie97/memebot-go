package strategy

import (
	"testing"
	"time"

	"memebot-go/internal/signal"
)

func TestTrendFollowerLongSignal(t *testing.T) {
	strat := NewTrendFollower(0.02, 120, 100)
	now := time.Now()
	ticks := []signal.Tick{
		{Symbol: "WIFSOL", Price: 0.01, Size: 5000, Side: 1, Ts: now.Add(-90 * time.Second)},
		{Symbol: "WIFSOL", Price: 0.0105, Size: 4000, Side: 1, Ts: now.Add(-60 * time.Second)},
		{Symbol: "WIFSOL", Price: 0.011, Size: 3000, Side: 1, Ts: now},
	}

	var sig *signal.Signal
	for _, tk := range ticks {
		sig = strat.OnTick(tk)
	}
	if sig == nil {
		t.Fatalf("expected long signal")
	}
	if sig.Score <= 0 {
		t.Fatalf("expected positive score, got %.4f", sig.Score)
	}
}

func TestTrendFollowerShortSignal(t *testing.T) {
	strat := NewTrendFollower(0.02, 120, 100)
	now := time.Now()
	ticks := []signal.Tick{
		{Symbol: "BODENSOL", Price: 0.02, Size: 4000, Side: -1, Ts: now.Add(-90 * time.Second)},
		{Symbol: "BODENSOL", Price: 0.0195, Size: 4000, Side: -1, Ts: now.Add(-60 * time.Second)},
		{Symbol: "BODENSOL", Price: 0.018, Size: 4000, Side: -1, Ts: now},
	}

	var sig *signal.Signal
	for _, tk := range ticks {
		sig = strat.OnTick(tk)
	}
	if sig == nil {
		t.Fatalf("expected short signal")
	}
	if sig.Score >= 0 {
		t.Fatalf("expected negative score, got %.4f", sig.Score)
	}
}

func TestTrendFollowerRespectsVolume(t *testing.T) {
	strat := NewTrendFollower(0.02, 120, 1000)
	now := time.Now()
	ticks := []signal.Tick{
		{Symbol: "LOWVOL", Price: 1, Size: 1, Side: 1, Ts: now.Add(-30 * time.Second)},
		{Symbol: "LOWVOL", Price: 1.03, Size: 1, Side: 1, Ts: now},
	}

	var sig *signal.Signal
	for _, tk := range ticks {
		sig = strat.OnTick(tk)
	}
	if sig != nil {
		t.Fatalf("expected nil signal due to insufficient volume")
	}
}
