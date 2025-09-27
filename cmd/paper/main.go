package main

import (
	"context"
	"fmt"
	"os"
	ossignal "os/signal"
	"syscall"

	"memebot-go/internal/config"
	"memebot-go/internal/exchange"
	"memebot-go/internal/execution"
	"memebot-go/internal/metrics"
	"memebot-go/internal/risk"
	sig "memebot-go/internal/signal"
	"memebot-go/internal/strategy"
	"memebot-go/internal/util"
)

func main() {
	log := util.NewLogger("info")

	cfg, err := config.Load("internal/config/config.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	_ = metrics.Serve(cfg.App.MetricsAddr)
	log.Info().Str("addr", cfg.App.MetricsAddr).Msg("metrics up")

	ctx, cancel := ossignal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer cancel()

	feed := exchange.NewFeed(cfg.Exchange.Symbols)
	ticks := make(chan sig.Tick, 1024)

	go func() {
		if err := feed.Run(ctx, ticks); err != nil {
			log.Error().Err(err).Msg("feed stopped")
			cancel()
		}
	}()

	strat := strategy.NewOBIMomentum(cfg.Strategy.Params.OBIThreshold, cfg.Strategy.Params.VolWindowSecs)
	limits := risk.Limits{MaxNotionalPerTrade: cfg.Risk.MaxNotionalPerTrade}
	exec := execution.NewExecutor(log)

	log.Info().Msg("paper engine started")
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("shutting down")
			return
		case tk := <-ticks:
			// Strategy → Signal
			sig := strat.OnTick(tk)
			if sig == nil {
				continue
			}

			notional := 10.0
			if !limits.Allow(notional) {
				continue
			}

			// PAPER: just log the hypothetical order
			side := execution.Buy
			if sig.Score < 0 {
				side = execution.Sell
			}
			_ = exec.Submit(execution.Order{
				Symbol: tk.Symbol,
				Side:   side,
				Qty:    0.001,
				Price:  0,
			})
			fmt.Print("") // keep import sanity
		}
	}
}
