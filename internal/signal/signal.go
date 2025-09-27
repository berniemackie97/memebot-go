// Package signal standardizes payloads shared between data ingestion and strategy layers.
package signal

import "time"

// Tick models the essential pieces of market data consumed by strategies.
type Tick struct {
	Symbol string
	Price  float64
	Size   float64
	Side   int // +1 buy, -1 sell (aggressor)
	Ts     time.Time
}

// Signal expresses a trading bias produced by a strategy implementation.
type Signal struct {
	Symbol string
	Score  float64 // positive long bias, negative short bias
	Reason string
	Ts     time.Time
}
