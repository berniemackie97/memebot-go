package paper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"memebot-go/internal/execution"
)

// JSONLRecorder appends fills as JSON lines for later analysis.
type JSONLRecorder struct {
	mu   sync.Mutex
	file *os.File
	enc  *json.Encoder
}

// NewJSONLRecorder creates/opens the target file and returns a recorder.
func NewJSONLRecorder(path string) (*JSONLRecorder, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	return &JSONLRecorder{
		file: file,
		enc:  json.NewEncoder(file),
	}, nil
}

// Record writes a single fill to the underlying JSONL file.
func (r *JSONLRecorder) Record(fill execution.Fill) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_ = r.enc.Encode(fill)
}

// Close flushes and closes the file handle.
func (r *JSONLRecorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	return err
}
