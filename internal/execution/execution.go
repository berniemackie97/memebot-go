// Package execution handles order lifecycle and interaction with venues.
package execution

import (
	"math/rand"
	"time"

	"github.com/rs/zerolog"

	"memebot-go/internal/metrics"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Side enumerates order directions used by the executor.
type Side string

const (
	// Buy indicates a long order.
	Buy Side = "BUY"
	// Sell indicates a short order.
	Sell Side = "SELL"
)

// Fill models a simulated execution result.
type Fill struct {
	Symbol   string        `json:"symbol"`
	Side     Side          `json:"side"`
	Qty      float64       `json:"qty"`
	Price    float64       `json:"price"`
	Slippage float64       `json:"slippage"`
	Latency  time.Duration `json:"latency"`
	Ts       time.Time     `json:"ts"`
}

// Config toggles paper execution behaviour.
type Config struct {
	MaxLatencyMs           int
	SlippageBps            float64
	PartialFillProbability float64
	MaxPartialFills        int
}

// Order represents a placement request the executor can process.
type Order struct {
	Symbol string
	Side   Side
	Qty    float64
	Price  float64 // 0 for market (avoid in real life)
}

// Executor implements a logger-backed submitter for orders.
type Executor struct {
	log    zerolog.Logger
	config Config
}

// NewExecutor wraps a zerolog logger for future order submissions.
func NewExecutor(log zerolog.Logger) *Executor {
	return &Executor{log: log, config: Config{MaxLatencyMs: 150, SlippageBps: 5, PartialFillProbability: 0.0, MaxPartialFills: 1}}
}

// SetConfig updates paper execution behaviour.
func (executor *Executor) SetConfig(cfg Config) { executor.config = cfg }

// Submit logs the order request and returns simulated fills; wire real exchange APIs later.
func (executor *Executor) Submit(order Order) ([]Fill, error) {
	metrics.OrdersTotal.WithLabelValues(order.Symbol, string(order.Side)).Inc()

	fills := executor.generateFills(order)
	for _, fill := range fills {
		executor.log.Info().
			Str("sym", order.Symbol).
			Str("side", string(order.Side)).
			Float64("qty", fill.Qty).
			Float64("px", order.Price).
			Float64("fill_px", fill.Price).
			Float64("slippage", fill.Slippage).
			Dur("latency", fill.Latency).
			Msg("submit order (stub)")
	}
	return fills, nil
}

func (executor *Executor) generateFills(order Order) []Fill {
	parts := executor.sampleParts()
	weights := make([]float64, parts)
	total := 0.0
	for i := range weights {
		w := rand.Float64()
		if w <= 0 {
			w = 1e-6
		}
		weights[i] = w
		total += w
	}

	fills := make([]Fill, parts)
	allocated := 0.0
	for i := 0; i < parts; i++ {
		qty := order.Qty * (weights[i] / total)
		if i == parts-1 {
			qty = order.Qty - allocated
		}
		allocated += qty

		latency := executor.sampleLatency()
		price := executor.applySlippage(order.Price, order.Side)
		fills[i] = Fill{
			Symbol:   order.Symbol,
			Side:     order.Side,
			Qty:      qty,
			Price:    price,
			Slippage: price - order.Price,
			Latency:  latency,
			Ts:       time.Now().Add(latency),
		}
	}
	return fills
}

func (executor *Executor) sampleParts() int {
	max := executor.config.MaxPartialFills
	if max < 2 || executor.config.PartialFillProbability <= 0 {
		return 1
	}
	if rand.Float64() >= executor.config.PartialFillProbability {
		return 1
	}
	return 1 + rand.Intn(max)
}

func (executor *Executor) sampleLatency() time.Duration {
	max := executor.config.MaxLatencyMs
	if max <= 0 {
		return 0
	}
	return time.Duration(rand.Intn(max+1)) * time.Millisecond
}

func (executor *Executor) applySlippage(px float64, side Side) float64 {
	if px <= 0 {
		return px
	}
	bps := executor.config.SlippageBps
	if bps <= 0 {
		return px
	}
	direction := 1.0
	if side == Sell {
		direction = -1.0
	}
	magnitude := (rand.Float64()*2 - 1) * bps / 10000 // uniform in [-bps,bps]
	return px * (1 + magnitude*direction)
}
