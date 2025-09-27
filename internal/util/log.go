package util

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
)

func NewLogger(level string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil { lvl = zerolog.InfoLevel }
	return zerolog.New(os.Stdout).With().Timestamp().Logger().Level(lvl)
}
