# Architecture Overview

This document captures the complete current shape of the `memebot-go` trading system, detailing the runtime binaries, internal packages, and how subsystems collaborate during an execution loop.

## Runtime Binaries

- `cmd/paper`: paper-trading daemon that wires the full pipeline (config → feed → strategy → risk → execution) against simulated data. Designed for development and integration testing.
- `cmd/executor`: placeholder for real-money trading. We intentionally keep it inert until all exchange integrations and safety controls are production-ready.
- `cmd/dexexec`: Solana/Jupiter swap exerciser. Useful for validating DeFi connectivity and wallet management without touching centralised venues.

## Configuration Layer

`internal/config` exposes typed structs for application, exchange, risk, strategy, and DEX parameters. The `Load` helper reads YAML and yields a strongly typed `Config`. Configuration fans out to every other module so that behavioural changes remain declarative.

## Data Ingestion Layer

`internal/exchange.Feed` currently emits synthetic ticks. In production it becomes a websocket or FIX streaming client per venue. The feed pushes `signal.Tick` messages into buffered channels consumed by strategies.

## Signal Generation

`internal/strategy.OBIMomentum` listens to ticks and produces directional `signal.Signal` outputs. Today the signal is a stub (score always zero) but the scaffolding is in place for order-book imbalance + momentum logic using the `threshold` and `window` knobs.

## Risk Management

`internal/risk.Limits` enforces simple notional caps. The paper engine can cheaply reject orders that exceed policy, and we will enrich this package with drawdown monitoring, kill-switch logic, and position awareness.

## Execution

`internal/execution.Executor` is a logging shim that records every order request. The executor will later route to the configured venue (CEX REST/WebSocket APIs or the Solana Jupiter aggregator) while emitting Prometheus metrics.

## Metrics and Observability

`internal/metrics` registers Prometheus counters (`ticks_total`, `orders_total`) and serves them via HTTP. The paper binary starts this server immediately so dashboards always receive data.

## Utilities

`internal/util` currently contains the structured logger setup that the binaries and packages reuse.

## DEX Connectivity

`internal/dex/solana` provides two building blocks:

1. `LoadPrivateKeyFromEnv` loads a base58-encoded keypair from environment variables (and optional `.env`).
2. `JupiterClient` wraps Jupiter quote retrieval plus transaction building and submission against an RPC node.

The `dexexec` binary demonstrates how to wire the client end-to-end.

## Flow Summary

1. The paper binary loads configuration and bootstraps logging plus Prometheus metrics.
2. A cancellable context listens for OS signals to guarantee clean shutdown.
3. The feed streams ticks onto a buffered channel inside a goroutine.
4. The strategy consumes ticks synchronously, transforms them into trading signals, and emits metadata such as reasoning and timestamps.
5. Risk checks gate the downstream execution path.
6. Eligible orders are submitted through the executor, which, for now, only logs the intent but forms the seam for production exchanges.

## Testing Philosophy

Unit tests cover configuration loading, risk limit checks, logger behaviour, feed streaming, and the Solana client request composition. Integration tests focus on the paper engine wiring (ensuring ticks flow to strategies and produce orders when thresholds are met).

As functionality hardens we will expand these suites with:

- Strategy-specific statistical backtests.
- Deterministic simulations for order book imbalance and volatility windows.
- Exchange contract tests verifying REST/WebSocket payloads.

## Outstanding Work

- Replace synthetic feed with real exchange data connectors and integrate Prometheus counters at ingestion.
- Flesh out `OBIMomentum` logic and calibrate thresholds.
- Implement account/risk state tracking (positions, PnL, drawdown) and enforce global kill switches.
- Promote the executor beyond logging: add actual REST/WS adapters plus order reconciliation.
- Build paper-trading fills model (latency, slippage) and record fills for later analysis.
- Extend DEX tooling with position swapping, quoting for multiple routes, and failure handling.
