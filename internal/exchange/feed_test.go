package exchange

import (
	"context"
	"testing"
	"time"

	"memebot-go/internal/signal"
)

func TestFeedRunEmitsTicks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	feed := NewFeed([]string{"BTCUSDT"})
	ticks := make(chan signal.Tick, 1)

	go func() {
		_ = feed.Run(ctx, ticks)
	}()

	select {
	case tk := <-ticks:
		if tk.Symbol != "BTCUSDT" {
			t.Fatalf("unexpected symbol %s", tk.Symbol)
		}
		cancel()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for tick")
	}
}
