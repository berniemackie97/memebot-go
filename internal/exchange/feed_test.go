package exchange

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestParseDexScreenerSymbols(t *testing.T) {
	targets, err := parseDexScreenerSymbols([]string{"WIFSOL@solana/PAIR", "BODEN@/another"}, "solana")
	if err != nil {
		t.Fatalf("parseDexScreenerSymbols returned error: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}
	if targets[0].Alias != "WIFSOL_PAIR" || targets[0].Chain != "solana" || targets[0].Address != "PAIR" {
		t.Fatalf("unexpected first target: %+v", targets[0])
	}
	if targets[1].Chain != "solana" {
		t.Fatalf("expected default chain applied")
	}
}

func TestRunDexScreenerEmitsTick(t *testing.T) {
	const body = `{"pairs":[{"priceUsd":"0.01","priceNative":"0.0001","txns":{"m5":{"buys":3,"sells":1},"h1":{"buys":5,"sells":4},"h6":{"buys":10,"sells":8},"h24":{"buys":20,"sells":20}},"volume":{"m5":120,"h1":500,"h6":1000,"h24":5000},"liquidity":{"usd":20000,"base":1000000,"quote":5000}}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	feed := NewFeed(
		ProviderDexScreener,
		[]string{"WIFSOL@solana/PAIR"},
		zerolog.Nop(),
		WithDexScreenerConfig(server.URL, "solana"),
		WithPollInterval(50*time.Millisecond),
	)

	ticks := make(chan signal.Tick, 1)
	errCh := make(chan error, 1)
	go func() {
		if err := feed.Run(ctx, ticks); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case tk := <-ticks:
		if tk.Symbol != "WIFSOL_PAIR" {
			t.Fatalf("unexpected symbol %s", tk.Symbol)
		}
		if tk.Price <= 0 {
			t.Fatalf("expected positive price")
		}
		if tk.Size <= 0 {
			t.Fatalf("expected positive size")
		}
		cancel()
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatalf("timed out waiting for tick")
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("feed returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("feed did not stop after cancel")
	}
}
