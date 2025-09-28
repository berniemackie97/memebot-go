package paper

import (
	"testing"

	"memebot-go/internal/execution"
)

func TestLedgerRecordSnapshot(t *testing.T) {
	ledger := NewLedger(2)
	fill := execution.Fill{Symbol: "BTCUSDT", Qty: 1}
	ledger.Record(fill)

	snapshot := ledger.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(snapshot))
	}
	if snapshot[0].Symbol != fill.Symbol {
		t.Fatalf("unexpected fill symbol")
	}

	ledger.Reset()
	if len(ledger.Snapshot()) != 0 {
		t.Fatalf("expected ledger reset")
	}
}
