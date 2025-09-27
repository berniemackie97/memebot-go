# memebot-go

High-speed, adaptive crypto executor (CEX/DEX-ready). This repo includes:
- **cmd/executor**: live daemon (connects to exchange, runs strategies)
- **cmd/paper**: paper-trading daemon (simulated fills)
- **cmd/dexexec**: Solana Jupiter swap exerciser
- **internal/**: clean packages for config, data ingest, signals, strategy, risk, execution, metrics

## Quick start
```bash
go run ./cmd/paper      # paper mode with metrics on :9090
go run ./cmd/executor   # live executor (stub)
```

## Progress
- ✅ Strongly typed configuration loader with sample YAML
- ✅ Synthetic exchange feed producing ticks for all configured symbols
- ✅ OBIMomentum strategy scaffold returning deterministic signals
- ✅ Risk notional guard-rail and logging executor stub
- ✅ Prometheus metrics server (`ticks_total`, `orders_total`)
- ✅ Solana/Jupiter DEX client and environment-driven wallet loader
- ✅ Unit + integration tests covering every subsystem and the paper flow

### Remaining To Hit "Complete"
1. Replace synthetic tick feed with real exchange connectivity and enrich metrics.
2. Flesh out OBIMomentum (true order book imbalance, volatility windows, thresholds).
3. Expand risk to track exposure, PnL, drawdown, and add global kill switches.
4. Implement real order routing (REST/WS) in `internal/execution` plus reconciliation.
5. Build paper-trading fills engine (latency, slippage, fill records) and persistence.
6. Harden DEX path with dynamic route selection, retries, and failure telemetry.

## Documentation
Full subsystem documentation lives in `docs/architecture.md`.
