package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"memebot-go/internal/metrics"
	"memebot-go/internal/signal"
)

type binanceEnvelope struct {
	Stream string       `json:"stream"`
	Data   binanceTrade `json:"data"`
}

type binanceTrade struct {
	Price        string `json:"p"`
	Quantity     string `json:"q"`
	TradeTime    int64  `json:"T"`
	IsBuyerMaker bool   `json:"m"`
}

func (f *Feed) runBinance(ctx context.Context, out chan<- signal.Tick) error {
	if len(f.Symbols) == 0 {
		return fmt.Errorf("binance feed requires at least one symbol")
	}

	streams := make([]string, len(f.Symbols))
	for i, sym := range f.Symbols {
		streams[i] = strings.ToLower(sym) + "@trade"
	}

	url := fmt.Sprintf("wss://stream.binance.com:9443/stream?streams=%s", strings.Join(streams, "/"))
	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := f.consumeBinanceStream(ctx, url, out); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			f.log.Warn().Err(err).Msg("binance feed disconnected, retrying")
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
			backoff = time.Duration(math.Min(float64(maxBackoff), float64(backoff)*1.8))
			continue
		}
		return nil
	}
}

func (f *Feed) consumeBinanceStream(ctx context.Context, url string, out chan<- signal.Tick) error {
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	f.log.Info().Str("provider", ProviderBinance).Strs("symbols", f.Symbols).Msg("connected market data feed")

	conn.SetReadLimit(1 << 20)
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})

	pingCtx, pingCancel := context.WithCancel(ctx)
	defer pingCancel()
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					f.log.Warn().Err(err).Msg("binance ping failed")
					return
				}
			case <-pingCtx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		_, message, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var env binanceEnvelope
		if err := json.Unmarshal(message, &env); err != nil {
			f.log.Warn().Err(err).Msg("failed to decode binance message")
			continue
		}

		symbol := parseBinanceSymbol(env.Stream)
		px, err := strconv.ParseFloat(env.Data.Price, 64)
		if err != nil {
			f.log.Warn().Err(err).Msg("invalid price from binance")
			continue
		}
		qty, err := strconv.ParseFloat(env.Data.Quantity, 64)
		if err != nil {
			f.log.Warn().Err(err).Msg("invalid quantity from binance")
			continue
		}
		side := 1
		if env.Data.IsBuyerMaker {
			side = -1
		}
		tick := signal.Tick{
			Symbol: symbol,
			Price:  px,
			Size:   qty,
			Side:   side,
			Ts:     time.UnixMilli(env.Data.TradeTime),
		}

		select {
		case out <- tick:
			metrics.TicksTotal.WithLabelValues(symbol).Inc()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func parseBinanceSymbol(stream string) string {
	parts := strings.Split(stream, "@")
	if len(parts) == 0 || parts[0] == "" {
		return strings.ToUpper(stream)
	}
	return strings.ToUpper(parts[0])
}
