// Package execution handles order lifecycle and interaction with venues.
package execution

import (
	"memebot-go/internal/metrics"

	"github.com/rs/zerolog"
)

// Side enumerates order directions used by the executor.
type Side string

const (
	// Buy indicates a long order.
	Buy Side = "BUY"
	// Sell indicates a short order.
	Sell Side = "SELL"
)

// Order represents a placement request the executor can process.
type Order struct {
	Symbol string
	Side   Side
	Qty    float64
	Price  float64 // 0 for market (avoid in real life)
}

// Executor implements a logger-backed submitter for orders.
type Executor struct{ log zerolog.Logger }

// NewExecutor wraps a zerolog logger for future order submissions.
func NewExecutor(log zerolog.Logger) *Executor { return &Executor{log: log} }

// Submit currently logs out the order request; wire real exchange APIs later.
func (executor *Executor) Submit(order Order) error {
	metrics.OrdersTotal.WithLabelValues(order.Symbol, string(order.Side)).Inc()
	executor.log.Info().Str("sym", order.Symbol).Str("side", string(order.Side)).Float64("qty", order.Qty).Float64("px", order.Price).Msg("submit order (stub)")
	// TODO: wire real REST/WS placement via exchange connector
	return nil
}
