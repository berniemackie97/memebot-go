package exchange

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"memebot-go/internal/signal"
)

func TestFeedRunEmitsTicks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	feed := NewFeed(ProviderStub, []string{"BTCUSDT"}, zerolog.Nop())
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

func TestParseBinanceSymbol(t *testing.T) {
	cases := map[string]string{
		"btcusdt@trade":    "BTCUSDT",
		"ethusdt@aggTrade": "ETHUSDT",
		"dogeusdt":         "DOGEUSDT",
		"":                 "",
	}
	for stream, expected := range cases {
		if got := parseBinanceSymbol(stream); got != expected {
			t.Fatalf("expected %s got %s", expected, got)
		}
	}
}
