package signal

import "time"

type Tick struct {
	Symbol string
	Price  float64
	Size   float64
	Side   int    // +1 buy, -1 sell (aggressor)
	Ts     time.Time
}

type Signal struct {
	Symbol string
	Score  float64 // positive long bias, negative short bias
	Reason string
	Ts     time.Time
}
