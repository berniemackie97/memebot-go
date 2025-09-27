// Package risk centralizes guard-rails that keep trading within safe limits.
package risk

// Limits represents simple scalar checks evaluated before order submission.
type Limits struct {
	MaxNotionalPerTrade float64
}

// Allow reports whether the proposed notional value fits within policy.
func (l Limits) Allow(notional float64) bool {
	return notional <= l.MaxNotionalPerTrade
}
