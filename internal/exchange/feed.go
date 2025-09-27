// Package exchange hosts connectors for centralized venues and tick sources.
package exchange

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"memebot-go/internal/metrics"
	"memebot-go/internal/signal"
)

const (
	// ProviderStub emits deterministic synthetic ticks (useful for tests/offline work).
	ProviderStub = "stub"
	// ProviderBinance streams live trades from Binance public websockets.
	ProviderBinance = "binance"
)

// Feed represents a pluggable market data stream implementation.
type Feed struct {
	provider string
	Symbols  []string
	log      zerolog.Logger
}

// NewFeed constructs a feed backed by the requested provider.
func NewFeed(provider string, symbols []string, log zerolog.Logger) *Feed {
	if provider == "" {
		provider = ProviderStub
	}
	return &Feed{provider: strings.ToLower(provider), Symbols: symbols, log: log}
}

// Run pushes ticks onto the provided channel until the context is canceled.
func (f *Feed) Run(ctx context.Context, out chan<- signal.Tick) error {
	switch f.provider {
	case ProviderBinance:
		return f.runBinance(ctx, out)
	default:
		return f.runStub(ctx, out)
	}
}

func (f *Feed) runStub(ctx context.Context, out chan<- signal.Tick) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var px float64 = 100.0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ts := <-ticker.C:
			px += 0.1
			for _, s := range f.Symbols {
				tick := signal.Tick{Symbol: s, Price: px, Size: 1, Side: 1, Ts: ts}
				select {
				case out <- tick:
					metrics.TicksTotal.WithLabelValues(s).Inc()
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}
