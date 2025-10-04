package risk

import "testing"

func TestAllow(t *testing.T) {
	limits := Limits{MaxNotionalPerTrade: 50}
	if !limits.Allow(49.9) {
		t.Fatalf("expected notional under limit to pass")
	}
	if limits.Allow(50.1) {
		t.Fatalf("expected notional above limit to fail")
	}
}

func TestAllowNoLimit(t *testing.T) {
	limits := Limits{MaxNotionalPerTrade: 0}
	if !limits.Allow(1e9) {
		t.Fatalf("expected unlimited notional when limit set to zero")
	}
}

func TestBreached(t *testing.T) {
	limits := Limits{MaxDrawdownPct: 0.2}
	if limits.Breached(1000, 900) {
		t.Fatalf("10%% drawdown should not breach 20%% limit")
	}
	if !limits.Breached(1000, 750) {
		t.Fatalf("25%% drawdown should breach 20%% limit")
	}
	if limits.Breached(0, 0) {
		t.Fatalf("zero starting cash should never breach")
	}
}

func TestIntraTradeBreached(t *testing.T) {
	limits := Limits{IntraTradeDrawdown: 0.1}
	if limits.IntraTradeBreached(1000, 950) {
		t.Fatalf("5%% drop should not breach 10%% limit")
	}
	if !limits.IntraTradeBreached(1000, 850) {
		t.Fatalf("15%% drop should breach 10%% limit")
	}
}

func TestDailyLossBreached(t *testing.T) {
	limits := Limits{MaxDailyLoss: 200}
	if limits.DailyLossBreached(-150) {
		t.Fatalf("loss below limit should not breach")
	}
	if !limits.DailyLossBreached(-250) {
		t.Fatalf("loss above limit should breach")
	}
	if limits.DailyLossBreached(100) {
		t.Fatalf("profit should never breach daily loss")
	}
}

func TestPortfolioBreached(t *testing.T) {
	limits := Limits{MaxPortfolioNotional: 500}
	if limits.PortfolioBreached(100, 400) {
		t.Fatalf("projected within limit should not breach")
	}
	if !limits.PortfolioBreached(400, 600) {
		t.Fatalf("projected above limit should breach")
	}
	if limits.PortfolioBreached(0, 100) {
		t.Fatalf("no limit breach expected when current below and projected below cap")
	}
}

func TestExposure(t *testing.T) {
	gross, net := Exposure(map[string]float64{"BTC": 1, "ETH": -0.5}, map[string]float64{"BTC": 10000, "ETH": 2000})
	if gross <= 0 {
		t.Fatalf("expected gross exposure > 0")
	}
	if net == 0 {
		t.Fatalf("expected net exposure != 0")
	}
}

func TestUnrealizedPnL(t *testing.T) {
	pnl := UnrealizedPnL(
		map[string]float64{"BTC": 1},
		map[string]float64{"BTC": 9000},
		map[string]float64{"BTC": 10000},
	)
	if pnl <= 0 {
		t.Fatalf("expected positive unrealized pnl")
	}
}
