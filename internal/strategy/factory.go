package strategy

import (
	"strings"

	sig "memebot-go/internal/signal"
)

// Strategy defines behaviour shared by strategy implementations used by the bot.
type Strategy interface {
	OnTick(t sig.Tick) *sig.Signal
	Name() string
}

// Params expresses tunable knobs required by strategy constructors.
type Params struct {
	OBILevels         int
	OBIThreshold      float64
	VolWindowSecs     int
	TrendThreshold    float64
	TrendWindowSecs   int
	TrendMinVolumeUSD float64
}

// Build returns a strategy implementation matching the configured mode.
func Build(mode string, params Params) Strategy {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "obi", "obi_momentum":
		return NewOBIMomentum(params.OBIThreshold, params.VolWindowSecs)
	case "trend", "trend_follow", "trend_follower":
		return NewTrendFollower(params.TrendThreshold, params.TrendWindowSecs, params.TrendMinVolumeUSD)
	default:
		return NewOBIMomentum(params.OBIThreshold, params.VolWindowSecs)
	}
}
