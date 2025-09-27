package risk

type Limits struct {
	MaxNotionalPerTrade float64
}

func (l Limits) Allow(notional float64) bool {
	return notional <= l.MaxNotionalPerTrade
}
