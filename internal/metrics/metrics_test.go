package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestServeRegistersMetrics(t *testing.T) {
	srv := Serve(":0")
	defer srv.Close()

	TicksTotal.WithLabelValues("BTCUSDT").Inc()

	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	found := false
	for _, mf := range mfs {
		if mf.GetName() == "ticks_total" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("ticks_total metric not found")
	}
}
