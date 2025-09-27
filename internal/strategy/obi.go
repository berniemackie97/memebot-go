package strategy

import (
	"time"

	"memebot-go/internal/signal"
)

type OBIMomentum struct {
	threshold float64
	window    time.Duration
}

func NewOBIMomentum(threshold float64, windowSec int) *OBIMomentum {
	return &OBIMomentum{ threshold: threshold, window: time.Duration(windowSec)*time.Second }
}

// Placeholder: in real code, consume L2 book & trades; here we transform ticks -> neutral signal
func (s *OBIMomentum) OnTick(t signal.Tick) *signal.Signal {
	// TODO: feed order book imbalance; returning 0 score for scaffold
	return &signal.Signal{ Symbol: t.Symbol, Score: 0, Reason: "stub", Ts: t.Ts }
}
