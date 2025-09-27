// Binary paper spins up a simulated trading loop with live market data and virtual risk checks.
package main

import (
	"context"
	"math"
	"os"
	ossignal "os/signal"
	"syscall"

	"memebot-go/internal/config"
	"memebot-go/internal/exchange"
	"memebot-go/internal/execution"
	"memebot-go/internal/metrics"
	"memebot-go/internal/paper"
	"memebot-go/internal/risk"
	sig "memebot-go/internal/signal"
	"memebot-go/internal/strategy"
	"memebot-go/internal/util"
)

func main() {
	// Logger initialization happens first to give us instrumentation for downstream failures.
	log := util.NewLogger("info")

	// Load strongly-typed configuration from YAML so we can wire all subsystems.
	cfg, err := config.Load("internal/config/config.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	// Launch Prometheus metrics early to watch the bot before it starts trading.
	_ = metrics.Serve(cfg.App.MetricsAddr)
	log.Info().Str("addr", cfg.App.MetricsAddr).Msg("metrics up")

	// Use signal-aware context so Ctrl+C and termination signals shut everything down cleanly.
	ctx, cancel := ossignal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Wire the market data feed and channel fanout the strategy consumes.
	feed := exchange.NewFeed(cfg.Exchange.Name, cfg.Exchange.Symbols, log)
	ticks := make(chan sig.Tick, 1024)

	// Kick off feed streaming in the background; cancel the app if the feed errors.
	go func() {
		if err := feed.Run(ctx, ticks); err != nil {
			log.Error().Err(err).Msg("feed stopped")
			cancel()
		}
	}()

	// Instantiate strategy, risk checks, executor, mark storage, and paper account state.
	strat := strategy.NewOBIMomentum(cfg.Strategy.Params.OBIThreshold, cfg.Strategy.Params.VolWindowSecs)
	limits := risk.Limits{MaxNotionalPerTrade: cfg.Risk.MaxNotionalPerTrade}
	exec := execution.NewExecutor(log)
	account := paper.NewAccount(cfg.Paper.StartingCash, cfg.Paper.MaxPositionPerSymbol)
	marks := make(map[string]float64, len(cfg.Exchange.Symbols))

	log.Info().Msg("paper engine started")
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("shutting down")
			return
		case tk := <-ticks:
			if tk.Price <= 0 {
				continue
			}
			marks[tk.Symbol] = tk.Price

			// Strategy -> Signal
			sig := strat.OnTick(tk)
			if sig == nil {
				continue
			}

			side := execution.Buy
			if sig.Score < 0 {
				side = execution.Sell
			}

			var qty float64
			switch side {
			case execution.Buy:
				cashBudget := account.AvailableCash()
				if cashBudget <= 0 {
					log.Warn().Msg("paper account out of cash; waiting for positions to unwind")
					continue
				}
				notional := cfg.Risk.MaxNotionalPerTrade
				if notional <= 0 {
					notional = cashBudget
				} else {
					notional = math.Min(notional, cashBudget)
				}
				if notional <= 0 {
					continue
				}
				qty = notional / tk.Price
			case execution.Sell:
				qty = account.Position(tk.Symbol)
			}

			if qty <= 0 {
				continue
			}

			order := execution.Order{
				Symbol: tk.Symbol,
				Side:   side,
				Qty:    qty,
				Price:  tk.Price,
			}

			notional := order.Qty * order.Price
			if !limits.Allow(notional) {
				log.Warn().Str("symbol", order.Symbol).Msg("risk rejected order over notional limit")
				continue
			}

			if err := account.MarketFill(order.Symbol, order.Side, order.Qty, order.Price); err != nil {
				log.Warn().Err(err).Str("symbol", order.Symbol).Msg("paper fill rejected")
				continue
			}

			if err := exec.Submit(order); err != nil {
				log.Error().Err(err).Str("symbol", order.Symbol).Msg("executor submit failed")
				continue
			}

			snap := account.Snapshot(marks)
			metrics.PaperEquity.Set(snap.Equity)
			for sym, pos := range snap.Positions {
				metrics.PaperPositions.WithLabelValues(sym).Set(pos.Qty)
			}
			if _, ok := snap.Positions[order.Symbol]; !ok {
				metrics.PaperPositions.WithLabelValues(order.Symbol).Set(0)
			}

			if pos, ok := snap.Positions[order.Symbol]; ok {
				log.Info().Str("symbol", order.Symbol).
					Str("side", string(order.Side)).
					Float64("qty", order.Qty).
					Float64("cash", snap.Cash).
					Float64("position", pos.Qty).
					Float64("avg_cost", pos.AvgCost).
					Float64("unrealized", pos.Unrealized).
					Float64("equity", snap.Equity).
					Float64("realized", snap.RealizedPnL).
					Msg("paper fill processed")
			} else {
				log.Info().Str("symbol", order.Symbol).
					Str("side", string(order.Side)).
					Float64("qty", order.Qty).
					Float64("cash", snap.Cash).
					Float64("equity", snap.Equity).
					Float64("realized", snap.RealizedPnL).
					Msg("paper position closed")
			}
		}
	}
}
