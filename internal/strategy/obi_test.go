package strategy

import (
	"testing"
	"time"

	"memebot-go/internal/signal"
)

func TestOnTickReturnsSignal(t *testing.T) {
	strat := NewOBIMomentum(1.0, 30)
	tick := signal.Tick{Symbol: "BTCUSDT", Ts: time.Now()}
	sig := strat.OnTick(tick)
	if sig == nil {
		t.Fatalf("expected signal output")
	}
	if sig.Symbol != tick.Symbol {
		t.Fatalf("expected symbol to match")
	}
}
