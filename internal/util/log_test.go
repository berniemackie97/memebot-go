package util

import (
	"testing"

	"github.com/rs/zerolog"
)

func TestNewLoggerLevel(t *testing.T) {
	logger := NewLogger("debug")
	if logger.GetLevel() != zerolog.DebugLevel {
		t.Fatalf("expected debug level, got %s", logger.GetLevel())
	}

	logger = NewLogger("invalid")
	if logger.GetLevel() != zerolog.InfoLevel {
		t.Fatalf("expected info fallback, got %s", logger.GetLevel())
	}
}
