package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"memebot-go/internal/metrics"
	"memebot-go/internal/signal"
)

type dexscreenerTarget struct {
	Alias   string
	Chain   string
	Address string
}

type dexscreenerPairsResponse struct {
	Pairs []dexscreenerPair `json:"pairs"`
	Pair  *dexscreenerPair  `json:"pair"`
}

type dexscreenerPair struct {
	ChainID     string                 `json:"chainId"`
	PairAddress string                 `json:"pairAddress"`
	BaseToken   dexscreenerToken       `json:"baseToken"`
	QuoteToken  dexscreenerToken       `json:"quoteToken"`
	PriceUsd    string                 `json:"priceUsd"`
	PriceNative string                 `json:"priceNative"`
	Txns        dexscreenerTxns        `json:"txns"`
	Volume      dexscreenerVolumes     `json:"volume"`
	Liquidity   dexscreenerLiquidity   `json:"liquidity"`
	PriceChange dexscreenerPriceChange `json:"priceChange"`
}

type dexscreenerToken struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
}

type dexscreenerTxns struct {
	M5  dexscreenerTxn `json:"m5"`
	H1  dexscreenerTxn `json:"h1"`
	H6  dexscreenerTxn `json:"h6"`
	H24 dexscreenerTxn `json:"h24"`
}

type dexscreenerTxn struct {
	Buys  int `json:"buys"`
	Sells int `json:"sells"`
}

type dexscreenerVolumes struct {
	M5  float64 `json:"m5"`
	H1  float64 `json:"h1"`
	H6  float64 `json:"h6"`
	H24 float64 `json:"h24"`
}

type dexscreenerLiquidity struct {
	USD   float64 `json:"usd"`
	Base  float64 `json:"base"`
	Quote float64 `json:"quote"`
}

type dexscreenerPriceChange struct {
	M5  float64 `json:"m5"`
	H1  float64 `json:"h1"`
	H6  float64 `json:"h6"`
	H24 float64 `json:"h24"`
}

func (r *dexscreenerPairsResponse) firstPair() (*dexscreenerPair, bool) {
	if len(r.Pairs) > 0 {
		return &r.Pairs[0], true
	}
	if r.Pair != nil {
		return r.Pair, true
	}
	return nil, false
}

func (f *Feed) runDexScreener(ctx context.Context, out chan<- signal.Tick) error {
	client := &http.Client{Timeout: 10 * time.Second}
	if err := f.pollDexScreener(ctx, client, out); err != nil && !errors.Is(err, context.Canceled) {
		f.log.Warn().Err(err).Msg("initial dexscreener poll failed")
	}

	ticker := time.NewTicker(f.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := f.pollDexScreener(ctx, client, out); err != nil && !errors.Is(err, context.Canceled) {
				f.log.Warn().Err(err).Msg("dexscreener poll failed")
			}
		}
	}
}

func (f *Feed) pollDexScreener(ctx context.Context, client *http.Client, out chan<- signal.Tick) error {
	targets, err := parseDexScreenerSymbols(f.snapshotSymbols(), f.dexscreenerDefaultChain)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}
	return f.dispatchDexScreener(ctx, client, targets, out)
}

func (f *Feed) dispatchDexScreener(ctx context.Context, client *http.Client, targets []dexscreenerTarget, out chan<- signal.Tick) error {
	for _, target := range targets {
		tick, err := f.fetchDexScreener(ctx, client, target)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			f.log.Warn().Err(err).Str("symbol", target.Alias).Msg("dexscreener fetch failed")
			continue
		}
		select {
		case out <- *tick:
			metrics.TicksTotal.WithLabelValues(tick.Symbol).Inc()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (f *Feed) fetchDexScreener(ctx context.Context, client *http.Client, target dexscreenerTarget) (*signal.Tick, error) {
	base := strings.TrimSuffix(f.dexscreenerBaseURL, "/")
	url := fmt.Sprintf("%s/latest/dex/pairs/%s/%s", base, target.Chain, target.Address)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "memebot-go/1.0 (paper)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var payload dexscreenerPairsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	pair, ok := payload.firstPair()
	if !ok {
		return nil, fmt.Errorf("no pair data returned")
	}
	price, err := parseDexScreenerPrice(pair)
	if err != nil {
		return nil, err
	}
	qty := estimateDexScreenerSize(pair, price)
	if qty <= 0 {
		qty = math.Max(1e-6, 10/price)
	}
	side := determineDexScreenerSide(pair, f.lastPrices[target.Alias], price)
	f.lastPrices[target.Alias] = price

	return &signal.Tick{
		Symbol: target.Alias,
		Price:  price,
		Size:   qty,
		Side:   side,
		Ts:     time.Now().UTC(),
	}, nil
}

func parseDexScreenerPrice(pair *dexscreenerPair) (float64, error) {
	if pair == nil {
		return 0, fmt.Errorf("pair missing")
	}
	if pair.PriceUsd != "" {
		if px, err := strconv.ParseFloat(pair.PriceUsd, 64); err == nil && px > 0 {
			return px, nil
		}
	}
	if pair.PriceNative != "" {
		if px, err := strconv.ParseFloat(pair.PriceNative, 64); err == nil && px > 0 {
			return px, nil
		}
	}
	return 0, fmt.Errorf("pair missing price")
}

func determineDexScreenerSide(pair *dexscreenerPair, lastPrice, price float64) int {
	if pair != nil {
		buys := pair.Txns.M5.Buys
		sells := pair.Txns.M5.Sells
		total := buys + sells
		if total > 0 {
			if buys >= sells {
				return 1
			}
			return -1
		}
	}
	if lastPrice > 0 && price < lastPrice {
		return -1
	}
	return 1
}

func estimateDexScreenerSize(pair *dexscreenerPair, price float64) float64 {
	if pair == nil || price <= 0 {
		return 0
	}
	periods := []struct {
		volume float64
		txns   dexscreenerTxn
	}{
		{pair.Volume.M5, pair.Txns.M5},
		{pair.Volume.H1, pair.Txns.H1},
		{pair.Volume.H6, pair.Txns.H6},
		{pair.Volume.H24, pair.Txns.H24},
	}
	for _, window := range periods {
		trades := window.txns.Buys + window.txns.Sells
		if window.volume > 0 && trades > 0 {
			avgUSD := window.volume / float64(trades)
			qty := avgUSD / price
			if qty > 0 {
				return qty
			}
		}
	}
	if pair.Liquidity.USD > 0 {
		notion := pair.Liquidity.USD * 0.0005
		if notion > 0 {
			qty := notion / price
			if qty > 0 {
				return qty
			}
		}
	}
	return 0
}

func parseDexScreenerSymbols(symbols []string, defaultChain string) ([]dexscreenerTarget, error) {
	defaultChain = strings.ToLower(strings.TrimSpace(defaultChain))
	targets := make([]dexscreenerTarget, 0, len(symbols))
	for _, raw := range symbols {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		aliasPart := raw
		targetPart := raw
		if parts := strings.SplitN(raw, "@", 2); len(parts) == 2 {
			aliasPart = parts[0]
			targetPart = parts[1]
		}
		chain := defaultChain
		address := targetPart
		if parts := strings.SplitN(targetPart, "/", 2); len(parts) == 2 {
			if parts[0] != "" {
				chain = strings.ToLower(strings.TrimSpace(parts[0]))
			}
			address = parts[1]
		}
		chain = strings.ToLower(strings.TrimSpace(chain))
		address = strings.TrimSpace(address)
		if chain == "" || address == "" {
			return nil, fmt.Errorf("dexscreener symbol %q missing chain or address", raw)
		}
		alias := composeDexAlias(aliasPart, address)
		targets = append(targets, dexscreenerTarget{Alias: alias, Chain: chain, Address: address})
	}
	return targets, nil
}
