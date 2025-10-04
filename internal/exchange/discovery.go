package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"memebot-go/internal/config"
)

// DexScreenerDiscovery continuously enriches the feed symbol list using Dexscreener search endpoints.
type DexScreenerDiscovery struct {
	log          zerolog.Logger
	feed         *Feed
	manual       []string
	client       *http.Client
	baseURL      string
	defaultChain string
	cfg          config.Discovery
	mu           sync.Mutex
	lastSet      []string
}

type candidatePair struct {
	symbol    string
	liquidity float64
	volume    float64
	change24  float64
	score     float64
}

// NewDexScreenerDiscovery constructs a discovery service; returns nil if disabled or nil feed.
func NewDexScreenerDiscovery(log zerolog.Logger, feed *Feed, manual []string, dexCfg config.DexScreener, cfg config.Discovery) *DexScreenerDiscovery {
	if feed == nil || !cfg.Enabled {
		return nil
	}
	baseURL := dexCfg.BaseURL
	if baseURL == "" {
		baseURL = defaultDexScreenerBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &DexScreenerDiscovery{
		log:          log,
		feed:         feed,
		manual:       append([]string(nil), manual...),
		client:       &http.Client{Timeout: 10 * time.Second},
		baseURL:      baseURL,
		defaultChain: strings.ToLower(dexCfg.DefaultChain),
		cfg:          cfg,
	}
}

// Start launches the discovery loop in a goroutine.
func (d *DexScreenerDiscovery) Start(ctx context.Context) {
	if d == nil {
		return
	}
	go d.loop(ctx)
}

func (d *DexScreenerDiscovery) loop(ctx context.Context) {
	interval := time.Duration(d.cfg.RefreshInterval) * time.Millisecond
	if interval <= 0 {
		interval = 15 * time.Second
	}
	if err := d.Refresh(ctx); err != nil {
		d.log.Warn().Err(err).Msg("symbol discovery refresh failed")
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := d.Refresh(ctx); err != nil {
				d.log.Warn().Err(err).Msg("symbol discovery refresh failed")
			}
		}
	}
}

// Refresh performs a single discovery cycle.
func (d *DexScreenerDiscovery) Refresh(ctx context.Context) error {
	if d == nil {
		return nil
	}
	candidates, err := d.discover(ctx)
	if err != nil {
		return err
	}
	discovered := make([]string, len(candidates))
	for i, cand := range candidates {
		discovered[i] = cand.symbol
	}
	combined := mergeSymbols(d.manual, discovered)
	d.feed.SetSymbols(combined)
	d.logDiscoveryChange(combined, candidates)
	return nil
}

func (d *DexScreenerDiscovery) discover(ctx context.Context) ([]candidatePair, error) {
	limits := d.cfg.MaxPairs
	if limits <= 0 {
		limits = 12
	}
	minLiquidity := d.cfg.MinLiquidityUSD
	minVolume := d.cfg.MinVolumeUSD
	chainAllow := make(map[string]struct{}, len(d.cfg.Chains))
	for _, chain := range d.cfg.Chains {
		chainAllow[strings.ToLower(strings.TrimSpace(chain))] = struct{}{}
	}
	if len(chainAllow) == 0 && d.defaultChain != "" {
		chainAllow[d.defaultChain] = struct{}{}
	}
	keywords := d.cfg.Keywords
	if len(keywords) == 0 {
		keywords = []string{"wif", "boden", "pepe", "doge"}
	}

	seen := make(map[string]struct{})
	candidates := make([]candidatePair, 0, limits*2)
	perKeywordLimit := d.cfg.MaxPairsPerKeyword
	if perKeywordLimit <= 0 {
		perKeywordLimit = limits
	}
	for _, keyword := range keywords {
		if len(candidates) >= limits {
			break
		}
		added := 0
		pairs, err := d.search(ctx, keyword)
		if err != nil {
			d.log.Debug().Err(err).Str("keyword", keyword).Msg("dexscreener search failed")
			continue
		}
		for _, pair := range pairs {
			if len(candidates) >= limits {
				break
			}
			if perKeywordLimit > 0 && added >= perKeywordLimit {
				break
			}
			chain := strings.ToLower(pair.ChainID)
			if len(chainAllow) > 0 {
				if _, ok := chainAllow[chain]; !ok {
					continue
				}
			}
			if minLiquidity > 0 && pair.Liquidity.USD < minLiquidity {
				continue
			}
			address := pair.PairAddress
			if address == "" {
				continue
			}
			if _, ok := seen[address]; ok {
				continue
			}
			volumeUSD := pair.Volume.H24
			if volumeUSD <= 0 {
				volumeUSD = pair.Volume.H6
			}
			if volumeUSD <= 0 {
				volumeUSD = pair.Volume.H1
			}
			if minVolume > 0 && volumeUSD < minVolume {
				continue
			}
			aliasBase := pair.BaseToken.Symbol
			if aliasBase == "" {
				aliasBase = pair.BaseToken.Name
			}
			quote := pair.QuoteToken.Symbol
			if quote == "" {
				quote = pair.QuoteToken.Name
			}
			aliasBase = aliasBase + quote
			sym := fmt.Sprintf("%s@%s/%s", composeDexAlias(aliasBase, address), chain, address)
			score := pair.Liquidity.USD*0.6 + volumeUSD*0.35
			if pair.PriceChange.H24 > 0 {
				score += pair.PriceChange.H24 * 1000
			}
			candidates = append(candidates, candidatePair{
				symbol:    sym,
				liquidity: pair.Liquidity.USD,
				volume:    volumeUSD,
				change24:  pair.PriceChange.H24,
				score:     score,
			})
			seen[address] = struct{}{}
			added++
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if math.Abs(candidates[i].score-candidates[j].score) > 1 {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].liquidity > candidates[j].liquidity
	})
	if len(candidates) > limits {
		candidates = candidates[:limits]
	}
	return candidates, nil
}

func (d *DexScreenerDiscovery) search(ctx context.Context, keyword string) ([]dexscreenerPair, error) {
	endpoint := fmt.Sprintf("%s/latest/dex/search?q=%s", d.baseURL, url.QueryEscape(keyword))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "memebot-go/1.0 (discovery)")
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var payload dexscreenerPairsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.Pairs) > 0 {
		return payload.Pairs, nil
	}
	if payload.Pair != nil {
		return []dexscreenerPair{*payload.Pair}, nil
	}
	return nil, nil
}

func (d *DexScreenerDiscovery) logDiscoveryChange(combined []string, discovered []candidatePair) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if slicesEqual(combined, d.lastSet) {
		return
	}
	prev := append([]string(nil), d.lastSet...)
	d.lastSet = append([]string(nil), combined...)
	detail := make([]string, len(discovered))
	for i, cand := range discovered {
		detail[i] = fmt.Sprintf("%s(liq=%.0f vol=%.0f Δ24=%.2f)", cand.symbol, cand.liquidity, cand.volume, cand.change24)
	}
	d.log.Info().
		Strs("symbols", combined).
		Strs("discovered", detail).
		Strs("manual", d.manual).
		Strs("previous", prev).
		Msg("updated symbol universe")
}

func mergeSymbols(manual, discovered []string) []string {
	set := make(map[string]struct{}, len(manual)+len(discovered))
	for _, sym := range manual {
		if sym = strings.TrimSpace(sym); sym != "" {
			set[sym] = struct{}{}
		}
	}
	for _, sym := range discovered {
		if sym = strings.TrimSpace(sym); sym != "" {
			set[sym] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for sym := range set {
		out = append(out, sym)
	}
	sort.Strings(out)
	return out
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
