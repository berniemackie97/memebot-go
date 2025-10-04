// Package risk centralizes guard-rails that keep trading within safe limits.
package risk

import "math"

// Limits represents simple scalar checks evaluated before order submission.
type Limits struct {
	MaxNotionalPerTrade  float64
	MaxDrawdownPct       float64
	IntraTradeDrawdown   float64
	MaxDailyLoss         float64
	MaxPortfolioNotional float64
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

// IntraTradeBreached compares current equity to the local peak to detect intra-session drawdowns.
func (l Limits) IntraTradeBreached(peakEquity, currentEquity float64) bool {
	if l.IntraTradeDrawdown <= 0 || peakEquity <= 0 {
		return false
	}
	drawdown := (peakEquity - currentEquity) / peakEquity
	return drawdown >= l.IntraTradeDrawdown
}

// DailyLossBreached returns true if realized losses exceed the daily cap.
func (l Limits) DailyLossBreached(realizedPnL float64) bool {
	if l.MaxDailyLoss <= 0 {
		return false
	}
	return -realizedPnL >= l.MaxDailyLoss
}

// PortfolioBreached reports whether adding additional notional would exceed the global cap.
func (l Limits) PortfolioBreached(currentGross, projectedGross float64) bool {
	if l.MaxPortfolioNotional <= 0 {
		return false
	}
	return projectedGross > l.MaxPortfolioNotional && projectedGross > currentGross
}

// Exposure returns gross and net exposure metrics based on positions and marks.
func Exposure(positions map[string]float64, marks map[string]float64) (gross float64, net float64) {
	for sym, qty := range positions {
		px := marks[sym]
		notion := qty * px
		gross += math.Abs(notion)
		net += notion
	}
	return
}

// UnrealizedPnL computes total unrealized PnL across positions.
func UnrealizedPnL(positions map[string]float64, avgCosts map[string]float64, marks map[string]float64) float64 {
	total := 0.0
	for sym, qty := range positions {
		mark := marks[sym]
		avg := avgCosts[sym]
		total += (mark - avg) * qty
	}
	return total
}
