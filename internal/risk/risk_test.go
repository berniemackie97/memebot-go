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
