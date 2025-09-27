package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	TicksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "ticks_total", Help: "Count of market ticks ingested"},
		[]string{"symbol"},
	)
	OrdersTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "orders_total", Help: "Orders submitted"},
		[]string{"symbol","side"},
	)
)

func init() {
	prometheus.MustRegister(TicksTotal, OrdersTotal)
}

func Serve(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{ Addr: addr, Handler: mux }
	go func() { _ = srv.ListenAndServe() }()
	return srv
}
