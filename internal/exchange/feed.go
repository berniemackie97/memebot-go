package exchange

import (
	"context"
	"time"

	"memebot-go/internal/signal"
)

// Feed is a stub tick stream; replace with real WS client per exchange
type Feed struct {
	Symbols []string
}

func NewFeed(symbols []string) *Feed { return &Feed{Symbols: symbols} }

func (f *Feed) Run(ctx context.Context, out chan<- signal.Tick) error {
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
				out <- signal.Tick{Symbol: s, Price: px, Size: 1, Side: 1, Ts: ts}
			}
		}
	}
}
