package paper

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"

	"memebot-go/internal/execution"
)

func TestJSONLRecorder(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/fills.jsonl"

	recorder, err := NewJSONLRecorder(path)
	if err != nil {
		t.Fatalf("NewJSONLRecorder error: %v", err)
	}
	fill := execution.Fill{Symbol: "BTCUSDT", Side: execution.Buy, Qty: 1, Price: 1000}
	recorder.Record(fill)
	if err := recorder.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open recorded file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatalf("expected one line in recorder output")
	}
	var decoded execution.Fill
	if err := json.Unmarshal(scanner.Bytes(), &decoded); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if decoded.Symbol != fill.Symbol || decoded.Side != fill.Side {
		t.Fatalf("unexpected decoded fill")
	}
}
