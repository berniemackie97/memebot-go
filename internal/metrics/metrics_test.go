package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestServeRegistersMetrics(t *testing.T) {
	srv := Serve(":0")
	defer srv.Close()

	TicksTotal.WithLabelValues("BTCUSDT").Inc()
	OrdersTotal.WithLabelValues("BTCUSDT", "BUY").Inc()
	PaperEquity.Set(123.45)
	PaperPositions.WithLabelValues("BTCUSDT").Set(0.5)

	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	found := map[string]bool{}
	for _, mf := range mfs {
		found[mf.GetName()] = true
	}

	required := []string{"ticks_total", "orders_total", "paper_equity", "paper_position"}
	for _, name := range required {
		if !found[name] {
			t.Fatalf("expected metric %s", name)
		}
	}
}
