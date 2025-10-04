// Package exchange hosts connectors for centralized venues and tick sources.
package exchange

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"memebot-go/internal/metrics"
	"memebot-go/internal/signal"
)

const (
	// ProviderStub emits deterministic synthetic ticks (useful for tests/offline work).
	ProviderStub = "stub"
	// ProviderBinance streams live trades from Binance public websockets.
	ProviderBinance = "binance"
	// ProviderDexScreener polls the Dexscreener HTTP API for on-chain meme coin pairs.
	ProviderDexScreener = "dexscreener"
)

// Feed represents a pluggable market data stream implementation.
type Feed struct {
	provider                string
	symbols                 []string
	log                     zerolog.Logger
	pollInterval            time.Duration
	dexscreenerBaseURL      string
	dexscreenerDefaultChain string
	lastPrices              map[string]float64
	mu                      sync.RWMutex
}

// Option configures Feed construction parameters.
type Option func(*Feed)

const (
	defaultPollInterval       = 2 * time.Second
	defaultDexScreenerBaseURL = "https://api.dexscreener.com"
)

// WithPollInterval overrides the default polling cadence for HTTP-based feeds.
func WithPollInterval(d time.Duration) Option {
	return func(f *Feed) {
		if d > 0 {
			f.pollInterval = d
		}
	}
}

// WithDexScreenerConfig injects base URL and default chain metadata for Dexscreener.
func WithDexScreenerConfig(baseURL, defaultChain string) Option {
	return func(f *Feed) {
		if baseURL != "" {
			f.dexscreenerBaseURL = strings.TrimSuffix(baseURL, "/")
		}
		if defaultChain != "" {
			f.dexscreenerDefaultChain = strings.ToLower(defaultChain)
		}
	}
}

// NewFeed constructs a feed backed by the requested provider.
func NewFeed(provider string, symbols []string, log zerolog.Logger, opts ...Option) *Feed {
	if provider == "" {
		provider = ProviderStub
	}
	f := &Feed{
		provider:                strings.ToLower(provider),
		log:                     log,
		pollInterval:            defaultPollInterval,
		dexscreenerBaseURL:      defaultDexScreenerBaseURL,
		dexscreenerDefaultChain: "",
		lastPrices:              make(map[string]float64),
	}
	f.setSymbols(symbols)
	for _, opt := range opts {
		opt(f)
	}
	if f.pollInterval <= 0 {
		f.pollInterval = defaultPollInterval
	}
	if f.dexscreenerBaseURL == "" {
		f.dexscreenerBaseURL = defaultDexScreenerBaseURL
	}
	return f
}

// SetSymbols replaces the tracked symbol list (deduplicated, sorted for determinism).
func (f *Feed) SetSymbols(symbols []string) {
	f.setSymbols(symbols)
}

func (f *Feed) setSymbols(symbols []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	unique := make(map[string]struct{}, len(symbols))
	for _, sym := range symbols {
		sym = strings.TrimSpace(sym)
		if sym == "" {
			continue
		}
		unique[sym] = struct{}{}
	}
	f.symbols = f.symbols[:0]
	for sym := range unique {
		f.symbols = append(f.symbols, sym)
	}
	sort.Strings(f.symbols)
}

func (f *Feed) snapshotSymbols() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]string, len(f.symbols))
	copy(out, f.symbols)
	return out
}

// Run pushes ticks onto the provided channel until the context is canceled.
func (f *Feed) Run(ctx context.Context, out chan<- signal.Tick) error {
	switch f.provider {
	case ProviderBinance:
		return f.runBinance(ctx, out)
	case ProviderDexScreener:
		return f.runDexScreener(ctx, out)
	default:
		return f.runStub(ctx, out)
	}
}

func (f *Feed) runStub(ctx context.Context, out chan<- signal.Tick) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var px float64 = 100.0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ts := <-ticker.C:
			px += 0.1
			symbols := f.snapshotSymbols()
			for _, s := range symbols {
				tick := signal.Tick{Symbol: s, Price: px, Size: 1, Side: 1, Ts: ts}
				select {
				case out <- tick:
					metrics.TicksTotal.WithLabelValues(s).Inc()
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}
