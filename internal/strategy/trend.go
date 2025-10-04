package strategy

import (
	"fmt"
	"math"
	"sync"
	"time"

	"memebot-go/internal/signal"
)

// TrendFollower emits signals when price momentum over a lookback window exceeds a threshold alongside minimum volume.
type TrendFollower struct {
	threshold    float64
	window       time.Duration
	minVolume    float64
	mu           sync.Mutex
	observations map[string]*trendSeries
}

type trendSeries struct {
	ticks []signal.Tick
}

// NewTrendFollower builds a trend-following strategy using percent change and volume filters.
func NewTrendFollower(threshold float64, windowSecs int, minVolumeUSD float64) *TrendFollower {
	if threshold <= 0 {
		threshold = 0.05
	}
	if windowSecs <= 0 {
		windowSecs = 180
	}
	return &TrendFollower{
		threshold:    threshold,
		window:       time.Duration(windowSecs) * time.Second,
		minVolume:    math.Max(0, minVolumeUSD),
		observations: make(map[string]*trendSeries),
	}
}

// Name returns the configured identifier for logging.
func (t *TrendFollower) Name() string { return "TrendFollower" }

// OnTick evaluates momentum and volume to decide whether to emit a signal.
func (t *TrendFollower) OnTick(tk signal.Tick) *signal.Signal {
	if tk.Symbol == "" || tk.Price <= 0 {
		return nil
	}

	t.mu.Lock()
	series := t.observations[tk.Symbol]
	if series == nil {
		series = &trendSeries{}
		t.observations[tk.Symbol] = series
	}
	series.append(tk, t.window)
	oldest, latest := series.bounds()
	totalNotional := series.notional()
	t.mu.Unlock()

	if oldest.Price <= 0 {
		return nil
	}
	change := (latest.Price - oldest.Price) / oldest.Price
	if math.Abs(change) < t.threshold {
		return nil
	}
	if t.minVolume > 0 && totalNotional < t.minVolume {
		return nil
	}
	reason := fmt.Sprintf("Δ=%.2f%% volume=%.0f", change*100, totalNotional)
	return &signal.Signal{Symbol: tk.Symbol, Score: change, Reason: reason, Ts: tk.Ts}
}

func (s *trendSeries) append(tk signal.Tick, window time.Duration) {
	s.ticks = append(s.ticks, tk)
	cutoff := tk.Ts.Add(-window)
	idx := 0
	for i, existing := range s.ticks {
		if existing.Ts.After(cutoff) {
			idx = i
			break
		}
		idx = i + 1
	}
	if idx > 0 && idx <= len(s.ticks) {
		s.ticks = s.ticks[idx:]
	}
}

func (s *trendSeries) bounds() (signal.Tick, signal.Tick) {
	if len(s.ticks) == 0 {
		return signal.Tick{}, signal.Tick{}
	}
	return s.ticks[0], s.ticks[len(s.ticks)-1]
}

func (s *trendSeries) notional() float64 {
	var total float64
	for _, tk := range s.ticks {
		total += math.Abs(tk.Price * tk.Size)
	}
	return total
}
