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
)

func init() {
	prometheus.MustRegister(TicksTotal, OrdersTotal)
}

// Serve mounts the Prometheus handler on /metrics and launches the HTTP server in a goroutine.
func Serve(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() { _ = srv.ListenAndServe() }()
	return srv
}
