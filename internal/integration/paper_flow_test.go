package integration

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"memebot-go/internal/exchange"
	"memebot-go/internal/execution"
	"memebot-go/internal/paper"
	"memebot-go/internal/risk"
	sig "memebot-go/internal/signal"
	"memebot-go/internal/strategy"
)

func TestPaperFlowProducesOrder(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	feed := exchange.NewFeed(exchange.ProviderStub, []string{"BTCUSDT"}, zerolog.Nop())
	ticks := make(chan sig.Tick, 8)
	go func() {
		_ = feed.Run(ctx, ticks)
	}()

	strat := strategy.NewOBIMomentum(0.05, 5)
	limits := risk.Limits{MaxNotionalPerTrade: 20}

	exec := NewTestExecutor(zerolog.New(io.Discard))
	account := paper.NewAccount(1000, 1, 100)
	marks := map[string]float64{}

	for {
		select {
		case tk := <-ticks:
			marks[tk.Symbol] = tk.Price
			sig := strat.OnTick(tk)
			if sig == nil {
				continue
			}
			order := execution.Order{Symbol: tk.Symbol, Side: execution.Buy, Qty: 0.001, Price: tk.Price}
			if !limits.Allow(order.Qty * order.Price) {
				t.Fatalf("expected notional under limit to pass")
			}
			fills, err := exec.Submit(order)
			if err != nil {
				t.Fatalf("Submit returned error: %v", err)
			}
			if len(fills) == 0 {
				t.Fatalf("expected fills to be generated")
			}
			for _, fill := range fills {
				if err := account.MarketFill(order.Symbol, order.Side, fill.Qty, fill.Price); err != nil {
					t.Fatalf("MarketFill returned error: %v", err)
				}
			}
			snap := account.Snapshot(marks)
			if snap.Equity <= 0 {
				t.Fatalf("expected positive equity")
			}
			return
		case <-ctx.Done():
			t.Fatalf("timed out waiting for integration flow")
		}
	}
}

// NewTestExecutor creates an executor with deterministic partials for tests.
func NewTestExecutor(log zerolog.Logger) *execution.Executor {
	exec := execution.NewExecutor(log)
	exec.SetConfig(execution.Config{MaxLatencyMs: 1, SlippageBps: 5, PartialFillProbability: 1.0, MaxPartialFills: 2})
	return exec
}
