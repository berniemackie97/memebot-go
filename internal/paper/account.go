package paper

import (
	"errors"
	"sync"

	"memebot-go/internal/execution"
)

// FillRecorder captures paper fills for later inspection.
type FillRecorder interface {
	Record(execution.Fill)
}

const epsilon = 1e-9

type positionState struct {
	Qty     float64
	AvgCost float64
}

// Account tracks virtual cash, realized PnL, and per-symbol positions while trading in paper mode.
type Account struct {
	mu                   sync.Mutex
	startingCash         float64
	cash                 float64
	realizedPnL          float64
	maxPositionPerSymbol float64
	positions            map[string]positionState
}

// PositionSnapshot exposes a read-only view of a single symbol position.
type PositionSnapshot struct {
	Qty         float64
	AvgCost     float64
	MarketValue float64
	Unrealized  float64
}

// Snapshot represents a thread-safe view of the account state, optionally marked to market using provided prices.
type Snapshot struct {
	Cash        float64
	RealizedPnL float64
	Equity      float64
	Positions   map[string]PositionSnapshot
}

// NewAccount constructs an account populated with starting cash and optional position cap.
func NewAccount(startingCash, maxPositionPerSymbol float64) *Account {
	return &Account{
		startingCash:         startingCash,
		cash:                 startingCash,
		maxPositionPerSymbol: maxPositionPerSymbol,
		positions:            make(map[string]positionState),
	}
}

// StartingCash returns the initial bankroll used to compute drawdown.
func (a *Account) StartingCash() float64 { return a.startingCash }

// MarketFill attempts to execute a market order at the provided price, mutating balances if successful.
func (a *Account) MarketFill(symbol string, side execution.Side, qty, price float64) error {
	if qty <= 0 {
		return errors.New("quantity must be positive")
	}
	if price <= 0 {
		return errors.New("price must be positive")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	state := a.positions[symbol]
	notional := qty * price

	switch side {
	case execution.Buy:
		if notional > a.cash+epsilon {
			return errors.New("insufficient cash for buy")
		}
		newQty := state.Qty + qty
		if a.maxPositionPerSymbol > 0 && newQty > a.maxPositionPerSymbol+epsilon {
			return errors.New("position limit exceeded")
		}
		newAvg := price
		if newQty > 0 {
			newAvg = ((state.AvgCost * state.Qty) + notional) / newQty
		}
		a.cash -= notional
		a.positions[symbol] = positionState{Qty: newQty, AvgCost: newAvg}

	case execution.Sell:
		if state.Qty <= 0 || state.Qty+epsilon < qty {
			return errors.New("insufficient position to sell")
		}
		realized := (price - state.AvgCost) * qty
		a.realizedPnL += realized
		a.cash += notional
		newQty := state.Qty - qty
		if newQty <= epsilon {
			delete(a.positions, symbol)
		} else {
			a.positions[symbol] = positionState{Qty: newQty, AvgCost: state.AvgCost}
		}

	default:
		return errors.New("unknown order side")
	}
	return nil
}

// Snapshot returns a copy of balances, optionally marked using the supplied prices map.
func (a *Account) Snapshot(prices map[string]float64) Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()

	positions := make(map[string]PositionSnapshot, len(a.positions))
	equity := a.cash
	for sym, pos := range a.positions {
		mark := prices[sym]
		marketValue := pos.Qty * mark
		unrealized := (mark - pos.AvgCost) * pos.Qty
		if mark == 0 {
			marketValue = 0
			unrealized = 0
		}
		positions[sym] = PositionSnapshot{
			Qty:         pos.Qty,
			AvgCost:     pos.AvgCost,
			MarketValue: marketValue,
			Unrealized:  unrealized,
		}
		equity += marketValue
	}

	return Snapshot{
		Cash:        a.cash,
		RealizedPnL: a.realizedPnL,
		Equity:      equity,
		Positions:   positions,
	}
}

// AvailableCash reports free cash that can be deployed into new longs.
func (a *Account) AvailableCash() float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cash
}

// Position returns the current position size for the supplied symbol.
func (a *Account) Position(symbol string) float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.positions[symbol].Qty
}

// RealizedPnL returns total closed-trade profit and loss.
func (a *Account) RealizedPnL() float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.realizedPnL
}
