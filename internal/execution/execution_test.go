package execution

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestSubmitLogsOrder(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	exec := NewExecutor(logger)
	err := exec.Submit(Order{Symbol: "BTCUSDT", Side: Buy, Qty: 1, Price: 0})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "BTCUSDT") {
		t.Fatalf("log does not contain symbol: %s", out)
	}
}
