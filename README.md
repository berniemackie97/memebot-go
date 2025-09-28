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
   - `exchange.name: "binance"` and set `exchange.symbols` to your spot pairs.
   - Tune bankroll + risk: `paper.starting_cash`, `paper.max_position_per_symbol`, `risk.kill_switch_drawdown`, `risk.max_notional_per_trade`.
   - Control execution realism: `paper.slippage_bps`, `paper.max_latency_ms`, `paper.partial_fill_probability`, `paper.max_partial_fills`.
   - Optional: set `paper.fills_path` to persist every simulated fill as JSONL.
   - Pick strategy thresholds under `strategy.params`.
2. Start metrics + paper loop:
   ```bash
   go run ./cmd/paper
   ```
3. Watch structured logs for fills, PnL, equity, slippage, and latency. Prometheus metrics are exposed at the configured `app.metrics_addr` (default `:9090`).
4. Inspect key gauges/counters:
   - `ticks_total{symbol}` – live trade ingest rate.
   - `orders_total{symbol,side}` – simulated order flow.
   - `paper_equity` – account equity (cash + mark-to-market positions).
   - `paper_position{symbol}` – current position size per symbol.

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
- [x] OBIMomentum strategy combining trade imbalance and momentum to emit live signals
- [x] Risk notional guard-rail + equity drawdown kill switch
- [x] Paper execution realism (slippage, latency, partial fills) with JSONL/in-memory trade ledger
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
- `exchange`: provider (`binance` today) and target symbols.
- `strategy`: implementation plus tunable parameters (OBI threshold, volatility window length).
- `risk`: per-trade notional guard-rails and `kill_switch_drawdown` equity stop.
- `dex`/`wallet`: Solana RPC + Jupiter endpoints and key material (used by `cmd/dexexec`).
- `paper`: bankroll (`starting_cash`), per-symbol cap, execution realism (`slippage_bps`, `max_latency_ms`, partial fill knobs), and optional fill log (`fills_path`).

## Documentation
Full subsystem documentation lives in `docs/architecture.md` with deep dives on binaries, dataflow, and outstanding work.
