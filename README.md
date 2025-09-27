# memebot-go

High-speed, adaptive crypto executor (CEX/DEX-ready). This repo includes:
- **cmd/executor**: live daemon (connects to exchange, runs strategies)
- **cmd/paper**: paper-trading daemon (simulated fills)
- **internal/**: clean packages for config, data ingest, signals, strategy, risk, execution, metrics

## Quick start
```bash
go run ./cmd/paper      # paper mode with metrics on :9090
go run ./cmd/executor   # live executor (stub)
```
