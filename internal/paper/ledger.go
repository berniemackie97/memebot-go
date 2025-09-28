package paper

import (
	"sync"

	"memebot-go/internal/execution"
)

// Ledger stores paper fills in memory for quick inspection.
type Ledger struct {
	mu    sync.Mutex
	fills []execution.Fill
}

// NewLedger creates an empty ledger optionally pre-sizing storage.
func NewLedger(capacity int) *Ledger {
	if capacity < 0 {
		capacity = 0
	}
	return &Ledger{fills: make([]execution.Fill, 0, capacity)}
}

// Record appends a fill to the ledger.
func (l *Ledger) Record(fill execution.Fill) {
	l.mu.Lock()
	l.fills = append(l.fills, fill)
	l.mu.Unlock()
}

// Snapshot returns a copy of the recorded fills.
func (l *Ledger) Snapshot() []execution.Fill {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]execution.Fill, len(l.fills))
	copy(out, l.fills)
	return out
}

// Reset clears all stored fills.
func (l *Ledger) Reset() {
	l.mu.Lock()
	l.fills = l.fills[:0]
	l.mu.Unlock()
}
