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

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	exec := execution.NewExecutor(logger)
	account := paper.NewAccount(1000, 1)
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
			if err := account.MarketFill(order.Symbol, order.Side, order.Qty, order.Price); err != nil {
				t.Fatalf("MarketFill returned error: %v", err)
			}
			if err := exec.Submit(order); err != nil {
				t.Fatalf("Submit returned error: %v", err)
			}
			snap := account.Snapshot(marks)
			if snap.Equity <= 0 {
				t.Fatalf("expected positive equity")
			}
			if !strings.Contains(buf.String(), "submit order") {
				t.Fatalf("expected log output to include submit order, got %s", buf.String())
			}
			return
		case <-ctx.Done():
			t.Fatalf("timed out waiting for integration flow")
		}
	}
}
