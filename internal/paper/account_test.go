package paper

import (
	"math"
	"testing"

	"memebot-go/internal/execution"
)

func TestMarketFillBuySellPnL(t *testing.T) {
	account := NewAccount(1000, 1)

	if err := account.MarketFill("BTCUSDT", execution.Buy, 0.5, 1000); err != nil {
		t.Fatalf("unexpected buy error: %v", err)
	}
	if err := account.MarketFill("BTCUSDT", execution.Buy, 0.25, 1100); err != nil {
		t.Fatalf("unexpected second buy error: %v", err)
	}

	snap := account.Snapshot(map[string]float64{"BTCUSDT": 1150})
	pos := snap.Positions["BTCUSDT"]
	if pos.Qty < 0.74 || pos.Qty > 0.76 {
		t.Fatalf("expected qty ~0.75, got %.4f", pos.Qty)
	}
	if pos.AvgCost <= 0 {
		t.Fatalf("avg cost not tracked")
	}
	if snap.Equity <= 0 {
		t.Fatalf("equity should be positive")
	}

	if err := account.MarketFill("BTCUSDT", execution.Sell, 0.25, 1200); err != nil {
		t.Fatalf("unexpected sell error: %v", err)
	}
	realized := account.RealizedPnL()
	if realized <= 0 {
		t.Fatalf("expected positive realized pnl got %.2f", realized)
	}

	snap = account.Snapshot(map[string]float64{"BTCUSDT": 1180})
	if math.Abs(snap.Cash+snap.Positions["BTCUSDT"].MarketValue-snap.Equity) > 1e-6 {
		t.Fatalf("equity did not balance")
	}
}

func TestMarketFillInsufficientCash(t *testing.T) {
	account := NewAccount(10, 1)
	if err := account.MarketFill("BTCUSDT", execution.Buy, 0.1, 200); err == nil {
		t.Fatalf("expected cash error")
	}
}

func TestMarketFillPositionLimit(t *testing.T) {
	account := NewAccount(1000, 0.1)
	if err := account.MarketFill("BTCUSDT", execution.Buy, 0.2, 1000); err == nil {
		t.Fatalf("expected position limit error")
	}
}

func TestMarketFillInsufficientPosition(t *testing.T) {
	account := NewAccount(1000, 1)
	if err := account.MarketFill("BTCUSDT", execution.Sell, 0.01, 1000); err == nil {
		t.Fatalf("expected insufficient position error")
	}
}
