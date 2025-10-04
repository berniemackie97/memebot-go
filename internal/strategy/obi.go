// Package strategy contains trading signal generation logic wires into ticks.
package strategy

import (
	"fmt"
	"math"
	"sync"
	"time"

	"memebot-go/internal/signal"
)

// OBIMomentum models a simple trade imbalance plus price momentum heuristic over a sliding window.
type OBIMomentum struct {
	threshold float64
	window    time.Duration
	mu        sync.Mutex
	series    map[string]*tickSeries
}

// Name returns the identifier for the strategy implementation.
func (s *OBIMomentum) Name() string { return "OBIMomentum" }

type tickSeries struct {
	ticks []signal.Tick
}

// NewOBIMomentum builds an OBIMomentum instance using threshold and look-back window seconds.
func NewOBIMomentum(threshold float64, windowSec int) *OBIMomentum {
	if threshold <= 0 {
		threshold = 0.25
	}
	if windowSec <= 0 {
		windowSec = 60
	}
	return &OBIMomentum{
		threshold: threshold,
		window:    time.Duration(windowSec) * time.Second,
		series:    make(map[string]*tickSeries),
	}
}

// OnTick transforms incoming ticks into directional signals by combining order flow imbalance and price momentum.
func (s *OBIMomentum) OnTick(t signal.Tick) *signal.Signal {
	if t.Symbol == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ts := s.series[t.Symbol]
	if ts == nil {
		ts = &tickSeries{}
		s.series[t.Symbol] = ts
	}
	ts.append(t, s.window)

	obi, momentum := ts.computeFeatures(t)
	score := 0.6*obi + 0.4*momentum
	if math.Abs(score) < s.threshold {
		return nil
	}

	reason := fmt.Sprintf("obi=%.2f momentum=%.2f", obi, momentum)
	return &signal.Signal{Symbol: t.Symbol, Score: score, Reason: reason, Ts: t.Ts}
}

func (ts *tickSeries) append(t signal.Tick, window time.Duration) {
	ts.ticks = append(ts.ticks, t)
	cutoff := t.Ts.Add(-window)
	idx := 0
	for i, tk := range ts.ticks {
		if tk.Ts.After(cutoff) {
			idx = i
			break
		}
		idx = i + 1
	}
	if idx > 0 && idx <= len(ts.ticks) {
		ts.ticks = ts.ticks[idx:]
	}
}

func (ts *tickSeries) computeFeatures(latest signal.Tick) (float64, float64) {
	if len(ts.ticks) == 0 {
		return 0, 0
	}

	var buyVol, sellVol float64
	for _, tk := range ts.ticks {
		vol := math.Abs(tk.Size)
		if tk.Side >= 0 {
			buyVol += vol
		} else {
			sellVol += vol
		}
	}

	total := buyVol + sellVol
	var obi float64
	if total > 0 {
		obi = (buyVol - sellVol) / total
		obi = clamp(obi, -1, 1)
	}

	anchor := ts.ticks[0].Price
	momentum := 0.0
	if anchor > 0 {
		raw := (latest.Price - anchor) / anchor
		momentum = clamp(math.Tanh(raw*3), -1, 1)
	}

	return obi, momentum
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
