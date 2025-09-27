// Package metrics exposes Prometheus collectors and HTTP serving helpers.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// TicksTotal tracks the number of market data ticks ingested per symbol.
	TicksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "ticks_total", Help: "Count of market ticks ingested"},
		[]string{"symbol"},
	)
	// OrdersTotal counts the number of orders submitted, keyed by symbol and side.
	OrdersTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "orders_total", Help: "Orders submitted"},
		[]string{"symbol", "side"},
	)
	// PaperEquity gauges current paper trading account equity (cash + mark-to-market positions).
	PaperEquity = prometheus.NewGauge(
		prometheus.GaugeOpts{Name: "paper_equity", Help: "Paper trading account equity mark-to-market"},
	)
	// PaperPositions reports current paper position sizes per symbol.
	PaperPositions = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "paper_position", Help: "Paper trading position size"},
		[]string{"symbol"},
	)
)

func init() {
	prometheus.MustRegister(TicksTotal, OrdersTotal, PaperEquity, PaperPositions)
}

// Serve mounts the Prometheus handler on /metrics and launches the HTTP server in a goroutine.
func Serve(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() { _ = srv.ListenAndServe() }()
	return srv
}
