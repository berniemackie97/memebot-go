// Package risk centralizes guard-rails that keep trading within safe limits.
package risk

// Limits represents simple scalar checks evaluated before order submission.
type Limits struct {
	MaxNotionalPerTrade float64
	MaxDrawdownPct      float64
}

// Allow reports whether the proposed notional value fits within policy.
func (l Limits) Allow(notional float64) bool {
	return notional <= l.MaxNotionalPerTrade || l.MaxNotionalPerTrade <= 0
}

// Breached returns true if the equity relative to the starting capital exceeds the configured drawdown.
func (l Limits) Breached(startingCash, currentEquity float64) bool {
	if l.MaxDrawdownPct <= 0 || startingCash <= 0 {
		return false
	}
	drawdown := (startingCash - currentEquity) / startingCash
	return drawdown >= l.MaxDrawdownPct
}
