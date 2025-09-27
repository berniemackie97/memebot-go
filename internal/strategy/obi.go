// Package strategy contains trading signal generation logic wires into ticks.
package strategy

import (
	"time"

	"memebot-go/internal/signal"
)

// OBIMomentum models a bare-bones order book imbalance plus momentum heuristic.
type OBIMomentum struct {
	threshold float64
	window    time.Duration
}

// NewOBIMomentum builds an OBIMomentum instance using threshold and look-back window seconds.
func NewOBIMomentum(threshold float64, windowSec int) *OBIMomentum {
	return &OBIMomentum{threshold: threshold, window: time.Duration(windowSec) * time.Second}
}

// OnTick transforms incoming ticks into directional signals (currently a stub baseline).
func (s *OBIMomentum) OnTick(t signal.Tick) *signal.Signal {
	// TODO: feed order book imbalance; returning 0 score for scaffold
	return &signal.Signal{Symbol: t.Symbol, Score: 0, Reason: "stub", Ts: t.Ts}
}
