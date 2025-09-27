package integration

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"memebot-go/internal/exchange"
	"memebot-go/internal/execution"
	"memebot-go/internal/risk"
	sig "memebot-go/internal/signal"
	"memebot-go/internal/strategy"
)

func TestPaperFlowProducesOrder(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	feed := exchange.NewFeed([]string{"BTCUSDT"})
	ticks := make(chan sig.Tick, 1)
	go func() {
		_ = feed.Run(ctx, ticks)
	}()

	strat := strategy.NewOBIMomentum(1.0, 10)
	limits := risk.Limits{MaxNotionalPerTrade: 20}

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	exec := execution.NewExecutor(logger)

	select {
	case tk := <-ticks:
		sig := strat.OnTick(tk)
		if sig == nil {
			t.Fatalf("expected signal output")
		}
		if !limits.Allow(10) {
			t.Fatalf("expected notional under limit to pass")
		}
		if err := exec.Submit(execution.Order{Symbol: tk.Symbol, Side: execution.Buy, Qty: 0.001}); err != nil {
			t.Fatalf("Submit returned error: %v", err)
		}
		if !strings.Contains(buf.String(), "submit order") {
			t.Fatalf("expected log output to include submit order, got %s", buf.String())
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for integration flow")
	}
}
