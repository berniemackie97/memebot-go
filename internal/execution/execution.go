package execution

import "github.com/rs/zerolog"

type Side string
const (
	Buy Side = "BUY"
	Sell Side = "SELL"
)

type Order struct {
	Symbol string
	Side   Side
	Qty    float64
	Price  float64 // 0 for market (avoid in real life)
}

type Executor struct { log zerolog.Logger }

func NewExecutor(log zerolog.Logger) *Executor { return &Executor{log: log} }

func (e *Executor) Submit(o Order) error {
	e.log.Info().Str("sym", o.Symbol).Str("side", string(o.Side)).Float64("qty", o.Qty).Float64("px", o.Price).Msg("submit order (stub)")
	// TODO: wire real REST/WS placement via exchange connector
	return nil
}
