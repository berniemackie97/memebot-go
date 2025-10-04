# memebot-go

High-speed, adaptive crypto executor (CEX/DEX-ready). This repo includes:
- **cmd/executor**: live daemon (connects to exchange, runs strategies)
- **cmd/paper**: paper-trading daemon (live data + simulated fills)
- **cmd/dexexec**: Solana Jupiter swap exerciser
- **internal/**: clean packages for config, data ingest, signals, strategy, risk, execution, metrics, paper accounting

## Build
```bash
go build ./cmd/paper      # compile paper daemon
go build ./cmd/executor   # compile (still stubbed)
go build ./cmd/dexexec    # compile Solana swap exerciser
```

## Paper Trading Quickstart
1. Edit `internal/config/config.yaml`:
   - Pick your data source. Use `exchange.name: "dexscreener"` with symbols formatted as `ALIAS@chain/pairAddress` for on-chain meme coins (see the sample `WIFSOL`/`BODENSOL` entries), or keep `binance` for CEX spot feeds.
   - Enable automatic meme-coin discovery via `exchange.discovery` (keywords, min liquidity/volume, per-keyword caps) to let the bot crawl Dexscreener in addition to any manually listed symbols.
   - Tune bankroll + risk: `paper.starting_cash`, `paper.max_position_notional_usd`, `risk.max_daily_loss`, `risk.max_notional_per_trade`, `risk.kill_switch_drawdown` (`risk.kill_switch_drawdown` also seeds the intratrade kill switch at 50%).
   - Control execution realism: `paper.slippage_bps`, `paper.max_latency_ms`, `paper.partial_fill_probability`, `paper.max_partial_fills`.
   - Optional: set `paper.fills_path` to persist every simulated fill as JSONL.
   - Select the trading engine with `strategy.mode` (`obi_momentum` imbalance model or `trend_follow` windowed momentum) and tune thresholds/volume filters under `strategy.params`.
2. Start metrics + paper loop:
   ```bash
   go run ./cmd/paper
   ```
   or launch the interactive console:
   ```bash
   go run ./cmd/tui
   ```
3. Observe the bot:
   - Structured logs describe fills (qty, price, slippage, latency), equity, exposures, and PnL.
   - Prometheus metrics at `app.metrics_addr` (default `:9090`).
   - Paper REST API (default `:8081`) exposes `/paper/fills` (JSON array of fills) and `/paper/account` (mark-to-market snapshot) for testers.

## Run Other Binaries
```bash
SOLANA_PRIVATE_KEY_BASE58=... \  # only needed for dexexec
  go run ./cmd/dexexec
```

## Test & Format
```bash
go fmt ./...          # format all Go code
go test ./...         # run unit + integration tests
go test ./... -cover  # inspect coverage by package
```

## Progress
- [x] Strongly typed configuration loader with sample YAML
- [x] Paper config (starting cash, per-symbol caps) and PnL-aware virtual account ledger
- [x] Binance live trade feed wired into paper execution loop with retry/resume
- [x] Dexscreener HTTP feed for Solana meme pairs with configurable polling cadence
- [x] Automatic Dexscreener discovery that continuously enriches the meme universe
- [x] Strategy factory with mode selection + logging for configured engine
- [x] Trend follower strategy (percent change + volume gate) for fast meme momentum plays
- [x] Per-symbol notional caps plus daily loss guardrails for the paper engine
- [x] OBIMomentum strategy combining trade imbalance and momentum to emit live signals
- [x] Risk notional guard-rail + equity/intratrade drawdown kill switches with exposure analytics
- [x] Paper execution realism (slippage, latency, partial fills) with JSONL/in-memory trade ledger + HTTP exposure
- [x] Prometheus metrics server (`ticks_total`, `orders_total`, `paper_equity`, `paper_position`)
- [x] Solana/Jupiter DEX client and environment-driven wallet loader
- [x] Unit + integration tests covering every subsystem, including paper flow

### Remaining To Hit "Complete"
1. Replace heuristic strategy logic with production-ready order book imbalance calculations and regression fixtures.
2. Expand risk to track exposure, PnL vectors, and add multi-stage kill switches.
3. Implement real order routing (REST/WebSocket) in `internal/execution` plus reconciliation.
4. Build richer paper fills engine (latency, slippage, order states) and persistence for analytics.
5. Harden DEX path with dynamic route selection, retries, and failure telemetry.
6. Add historical/backtest tooling for strategies alongside live paper validation.

## Configuration Cheatsheet
`internal/config/config.yaml` drives every binary. Key sections:
- `app`: process metadata, log level, Prometheus bind address.
- `exchange`: provider (`dexscreener` for memecoins, `binance` for CEX) and target symbols/options, including `exchange.discovery` for Dexscreener crawling with liquidity/volume heuristics.
- `strategy`: implementation plus tunable parameters (OBI threshold, volatility window length, trend thresholds/volume).
- `risk`: per-trade notional guard-rails, daily loss caps, and drawdown kill switches.
- `dex`/`wallet`: Solana RPC + Jupiter endpoints and key material (used by `cmd/dexexec`).
- `paper`: bankroll (`starting_cash`), per-symbol quantity/notional caps, execution realism (`slippage_bps`, `max_latency_ms`, partial fill knobs), fill log (`fills_path`).

## Documentation
Full subsystem documentation lives in `docs/architecture.md` with deep dives on binaries, dataflow, and outstanding work.
