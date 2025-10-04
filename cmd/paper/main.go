// Binary paper spins up a simulated trading loop with live market data and virtual risk checks.
package main

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"os"
	ossignal "os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"

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
	srv := metrics.Serve(cfg.App.MetricsAddr)
	log.Info().Str("addr", cfg.App.MetricsAddr).Msg("metrics up")

	// Use signal-aware context so Ctrl+C and termination signals shut everything down cleanly.
	ctx, cancel := ossignal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Wire the market data feed and channel fanout the strategy consumes.
	feedOpts := []exchange.Option{}
	if cfg.Exchange.DexScreener.PollInterval > 0 {
		feedOpts = append(feedOpts, exchange.WithPollInterval(time.Duration(cfg.Exchange.DexScreener.PollInterval)*time.Millisecond))
	}
	if strings.EqualFold(cfg.Exchange.Name, exchange.ProviderDexScreener) {
		feedOpts = append(feedOpts, exchange.WithDexScreenerConfig(cfg.Exchange.DexScreener.BaseURL, cfg.Exchange.DexScreener.DefaultChain))
	}
	feed := exchange.NewFeed(cfg.Exchange.Name, cfg.Exchange.Symbols, log, feedOpts...)
	ticks := make(chan sig.Tick, 1024)

	if strings.EqualFold(cfg.Exchange.Name, exchange.ProviderDexScreener) {
		if discovery := exchange.NewDexScreenerDiscovery(log, feed, cfg.Exchange.Symbols, cfg.Exchange.DexScreener, cfg.Exchange.Discovery); discovery != nil {
			discovery.Start(ctx)
		}
	}

	// Kick off feed streaming in the background; cancel the app if the feed errors.
	go func() {
		if err := feed.Run(ctx, ticks); err != nil {
			log.Error().Err(err).Msg("feed stopped")
			cancel()
		}
	}()

	// Instantiate strategy, risk checks, executor, mark storage, and paper account state.
	strategyParams := strategy.Params{
		OBILevels:         cfg.Strategy.Params.OBILevels,
		OBIThreshold:      cfg.Strategy.Params.OBIThreshold,
		VolWindowSecs:     cfg.Strategy.Params.VolWindowSecs,
		TrendThreshold:    cfg.Strategy.Params.TrendThreshold,
		TrendWindowSecs:   cfg.Strategy.Params.TrendWindowSecs,
		TrendMinVolumeUSD: cfg.Strategy.Params.TrendMinVolumeUSD,
	}
	strat := strategy.Build(cfg.Strategy.Mode, strategyParams)
	log.Info().Str("strategy", strat.Name()).Msg("strategy initialized")
	limits := risk.Limits{
		MaxNotionalPerTrade: cfg.Risk.MaxNotionalPerTrade,
		MaxDrawdownPct:      cfg.Risk.KillSwitchDrawdown,
		IntraTradeDrawdown:  cfg.Risk.KillSwitchDrawdown / 2,
		MaxDailyLoss:        cfg.Risk.MaxDailyLoss,
	}

	exec := execution.NewExecutor(log)
	exec.SetConfig(execution.Config{
		MaxLatencyMs:           cfg.Paper.MaxLatencyMs,
		SlippageBps:            cfg.Paper.SlippageBps,
		PartialFillProbability: cfg.Paper.PartialFillProbability,
		MaxPartialFills:        cfg.Paper.MaxPartialFills,
	})

	account := paper.NewAccount(cfg.Paper.StartingCash, cfg.Paper.MaxPositionPerSymbol, cfg.Paper.MaxPositionNotionalUSD)
	marks := make(map[string]float64, len(cfg.Exchange.Symbols))
	ledger := paper.NewLedger(2048)

	var recorder paper.FillRecorder
	if path := cfg.Paper.FillsPath; path != "" {
		rec, err := paper.NewJSONLRecorder(path)
		if err != nil {
			log.Warn().Err(err).Msg("paper recorder disabled")
		} else {
			recorder = rec
			defer rec.Close()
		}
	}

	// Expose ledger snapshots at /paper/fills for testers.
	mux := http.NewServeMux()
	mux.HandleFunc("/paper/fills", func(w http.ResponseWriter, r *http.Request) {
		fills := ledger.Snapshot()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fills)
	})
	mux.HandleFunc("/paper/account", func(w http.ResponseWriter, r *http.Request) {
		snap := account.Snapshot(marks)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snap)
	})
	go func() {
		log.Info().Str("addr", ":8081").Msg("paper HTTP API up")
		_ = http.ListenAndServe(":8081", mux)
	}()

	peakEquity := cfg.Paper.StartingCash
	halted := false
	terminate := func(reason string) {
		if halted {
			return
		}
		halted = true
		log.Warn().Str("reason", reason).Msg("risk limit triggered; flattening positions and pausing trading")
		flattenPositions(exec, account, marks, ledger, recorder, log)
		snap := account.Snapshot(marks)
		metrics.PaperEquity.Set(snap.Equity)
		for sym := range marks {
			metrics.PaperPositions.WithLabelValues(sym).Set(0)
		}
		for sym, pos := range snap.Positions {
			metrics.PaperPositions.WithLabelValues(sym).Set(pos.Qty)
		}
		peakEquity = snap.Equity
	}

	log.Info().Msg("paper engine started")
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("shutting down")
			srv.Shutdown(context.Background())
			return
		case tk := <-ticks:
			if tk.Price <= 0 {
				continue
			}
			marks[tk.Symbol] = tk.Price
			if halted {
				continue
			}

			// Check drawdowns before trading.
			currentSnap := account.Snapshot(marks)
			if currentSnap.Equity > peakEquity {
				peakEquity = currentSnap.Equity
			}
			if limits.DailyLossBreached(account.RealizedPnL()) {
				terminate("daily loss limit reached")
				return
			}
			if limits.Breached(account.StartingCash(), currentSnap.Equity) {
				terminate("drawdown limit reached")
				return
			}
			if limits.IntraTradeBreached(peakEquity, currentSnap.Equity) {
				terminate("intratrade drawdown reached")
				return
			}

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
				capacity := account.MaxAdditionalLong(tk.Symbol, tk.Price)
				if capacity <= 0 {
					log.Debug().Str("symbol", tk.Symbol).Msg("position cap reached; skipping buy")
					continue
				}
				qty = math.Min(qty, capacity)
			case execution.Sell:
				qty = account.Position(tk.Symbol)
			}

			if qty <= 0 {
				continue
			}

			if side == execution.Buy && limits.MaxPortfolioNotional > 0 {
				grossBefore, _ := risk.Exposure(extractQtys(currentSnap.Positions), marks)
				projected := grossBefore + qty*tk.Price
				if limits.PortfolioBreached(grossBefore, projected) {
					log.Debug().Float64("projected", projected).Float64("limit", limits.MaxPortfolioNotional).Str("symbol", tk.Symbol).Msg("portfolio notional limit reached; skipping buy")
					continue
				}
			}

			order := execution.Order{
				Symbol: tk.Symbol,
				Side:   side,
				Qty:    qty,
				Price:  tk.Price,
			}

			notional := order.Qty * order.Price
			if side == execution.Buy && !limits.Allow(notional) {
				log.Warn().Str("symbol", order.Symbol).Msg("risk rejected order over notional limit")
				continue
			}

			fills, err := exec.Submit(order)
			if err != nil {
				log.Error().Err(err).Str("symbol", order.Symbol).Msg("executor submit failed")
				continue
			}

			var totalFilled float64
			for _, fill := range fills {
				price := fill.Price
				if price <= 0 {
					price = order.Price
				}
				if err := account.MarketFill(order.Symbol, order.Side, fill.Qty, price); err != nil {
					log.Warn().Err(err).Str("symbol", order.Symbol).Msg("paper fill rejected")
					continue
				}
				totalFilled += fill.Qty
				ledger.Record(fill)
				if recorder != nil {
					recorder.Record(fill)
				}
			}
			if totalFilled <= 0 {
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

			gross, net := risk.Exposure(extractQtys(snap.Positions), marks)
			unrealized := aggregateUnrealized(snap.Positions)
			logEvent := log.Info().Str("symbol", order.Symbol).
				Str("side", string(order.Side)).
				Float64("qty", totalFilled).
				Float64("signal_score", sig.Score).
				Float64("cash", snap.Cash).
				Float64("equity", snap.Equity).
				Float64("realized", snap.RealizedPnL).
				Float64("gross_exposure", gross).
				Float64("net_exposure", net).
				Float64("unrealized", unrealized)
			if pos, ok := snap.Positions[order.Symbol]; ok {
				logEvent = logEvent.Float64("position", pos.Qty).Float64("avg_cost", pos.AvgCost)
			} else {
				logEvent = logEvent.Float64("position", 0).Float64("avg_cost", 0)
			}
			logEvent.Msg("paper fills processed")

			if snap.Equity > peakEquity {
				peakEquity = snap.Equity
			}
			if limits.Breached(account.StartingCash(), snap.Equity) {
				terminate("drawdown limit reached after fill")
				return
			}
			if limits.DailyLossBreached(account.RealizedPnL()) {
				terminate("daily loss limit reached after fill")
				return
			}
			currentSnap = snap
		}
	}
}

func extractQtys(pos map[string]paper.PositionSnapshot) map[string]float64 {
	out := make(map[string]float64, len(pos))
	for sym, snapshot := range pos {
		out[sym] = snapshot.Qty
	}
	return out
}

func aggregateUnrealized(pos map[string]paper.PositionSnapshot) float64 {
	total := 0.0
	for _, snapshot := range pos {
		total += snapshot.Unrealized
	}
	return total
}

func flattenPositions(exec *execution.Executor, account *paper.Account, marks map[string]float64, ledger *paper.Ledger, recorder paper.FillRecorder, log zerolog.Logger) {
	snap := account.Snapshot(marks)
	for sym, pos := range snap.Positions {
		qty := pos.Qty
		if math.Abs(qty) <= 1e-9 {
			continue
		}
		side := execution.Sell
		if qty < 0 {
			side = execution.Buy
			qty = -qty
		}
		price := marks[sym]
		if price <= 0 {
			price = pos.AvgCost
			if price <= 0 {
				price = 1
			}
		}
		order := execution.Order{Symbol: sym, Side: side, Qty: qty, Price: price}
		fills, err := exec.Submit(order)
		if err != nil {
			log.Warn().Err(err).Str("symbol", sym).Msg("flatten submit failed")
			continue
		}
		for _, fill := range fills {
			px := fill.Price
			if px <= 0 {
				px = order.Price
			}
			if err := account.MarketFill(sym, side, fill.Qty, px); err != nil {
				log.Warn().Err(err).Str("symbol", sym).Msg("flatten fill rejected")
				continue
			}
			fill.Price = px
			fill.Side = side
			fill.Symbol = sym
			ledger.Record(fill)
			if recorder != nil {
				recorder.Record(fill)
			}
		}
		marks[sym] = price
	}
}
