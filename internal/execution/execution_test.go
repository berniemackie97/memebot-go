package execution

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestSubmitLogsOrder(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	exec := NewExecutor(logger)
	exec.SetConfig(Config{MaxLatencyMs: 1, SlippageBps: 0, PartialFillProbability: 0, MaxPartialFills: 1})
	fills, err := exec.Submit(Order{Symbol: "BTCUSDT", Side: Buy, Qty: 1, Price: 1000})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}
	if len(fills) != 1 {
		t.Fatalf("expected single fill, got %d", len(fills))
	}
	if fills[0].Price != 1000 {
		t.Fatalf("expected no slippage, got %.2f", fills[0].Price)
	}
	out := buf.String()
	if !strings.Contains(out, "submit order") {
		t.Fatalf("log does not contain symbol: %s", out)
	}
}

func TestSubmitSlippageLatencyAndPartials(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	exec := NewExecutor(logger)
	exec.SetConfig(Config{MaxLatencyMs: 20, SlippageBps: 10, PartialFillProbability: 1.0, MaxPartialFills: 3})

	fills, err := exec.Submit(Order{Symbol: "ETHUSDT", Side: Sell, Qty: 2, Price: 2000})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}
	if len(fills) < 1 || len(fills) > 3 {
		t.Fatalf("expected between 1 and 3 fills, got %d", len(fills))
	}
	totalQty := 0.0
	for _, fill := range fills {
		totalQty += fill.Qty
		if fill.Latency < 0 || fill.Latency > 20*time.Millisecond {
			t.Fatalf("latency not within configured bounds: %v", fill.Latency)
		}
	}
	if totalQty <= 0 {
		t.Fatalf("total quantity should be positive")
	}
}
