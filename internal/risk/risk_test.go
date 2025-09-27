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
