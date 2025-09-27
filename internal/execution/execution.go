package execution

import "github.com/rs/zerolog"

type Side string

const (
	Buy  Side = "BUY"
	Sell Side = "SELL"
)

type Order struct {
	Symbol string
	Side   Side
	Qty    float64
	Price  float64 // 0 for market (avoid in real life)
}

type Executor struct{ log zerolog.Logger }

func NewExecutor(log zerolog.Logger) *Executor { return &Executor{log: log} }

func (executor *Executor) Submit(order Order) error {
	executor.log.Info().Str("sym", order.Symbol).Str("side", string(order.Side)).Float64("qty", order.Qty).Float64("px", order.Price).Msg("submit order (stub)")
	// TODO: wire real REST/WS placement via exchange connector
	return nil
}
