# Architecture Overview

This document captures the complete current shape of the `memebot-go` trading system, detailing the runtime binaries, internal packages, and how subsystems collaborate during an execution loop.

## Runtime Binaries

- `cmd/paper`: paper-trading daemon that wires the full pipeline (config -> feed -> strategy -> risk -> execution -> virtual account) against **live** market data.
- `cmd/executor`: placeholder for real-money trading. We intentionally keep it inert until all exchange integrations and safety controls are production-ready.
- `cmd/dexexec`: Solana/Jupiter swap exerciser. Useful for validating DeFi connectivity and wallet management without touching centralised venues.

## Configuration Layer

`internal/config` exposes typed structs for application, exchange, risk, strategy, paper-account, and DEX parameters. The `Load` helper reads YAML and yields a strongly typed `Config`. Configuration fans out to every other module so that behavioural changes remain declarative.

## Data Ingestion Layer

`internal/exchange` now supports multiple providers. In development/tests we can fall back to the synthetic stub, while production paper runs can consume Binance aggregated trades via public websockets with retry/ping handling or poll Dexscreener for Solana meme coin pairs using configurable HTTP intervals. A companion discovery loop continuously calls Dexscreener search endpoints (keyword + liquidity/volume filters), scores results by liquidity/volume/price change, and updates the feed with newly surfaced meme pairs so that strategies automatically expand their universe without manual intervention. The feed pushes `signal.Tick` messages into buffered channels consumed by strategies and also increments Prometheus tick counters.

## Signal Generation

`internal/strategy.OBIMomentum` maintains per-symbol rolling windows of trade data. It computes a simple order-flow imbalance (buy volume vs sell volume) and combines it with price momentum (tanh-normalised change over the window). Weighted scores exceeding the configured threshold emit `signal.Signal` objects for downstream consumers. A lightweight `strategy.Build` factory selects the configured engine (OBI or the new TrendFollower momentum strategy that requires both windowed percent change and USD volume) so operators can toggle playbooks from configuration.

## Risk Management

`internal/risk` now supplies notional guards plus dual drawdown controls (equity-based and intratrade relative to the latest peak) alongside a daily realised-loss kill switch. Helper functions compute gross/net exposure and aggregate unrealised PnL so operators can monitor risk in real time.

## Paper Accounting

`internal/paper.Account` maintains simulated cash balances, realised PnL, and per-symbol positions. It enforces starting bankroll, per-symbol quantity caps, optional per-symbol USD notional caps, and ensures sells only execute against available inventory. Mark-to-market snapshots feed logs, Prometheus gauges, risk checks, and the optional `paper.Ledger`/`paper.JSONLRecorder` for post-run analysis.

## Execution

`internal/execution.Executor` is a logging shim that records every order request, applies configurable slippage/latency, optionally breaks fills into partial executions, bumps Prometheus counters, and returns simulated fills. The executor will later route to the configured venue (CEX REST/WebSocket APIs or the Solana Jupiter aggregator) while emitting metrics.

## Metrics and Observability

`internal/metrics` registers Prometheus counters and gauges:
- `ticks_total{symbol}`: live market data ingest rate.
- `orders_total{symbol,side}`: simulated order flow.
- `paper_equity`: paper account equity (cash + positions).
- `paper_position{symbol}`: open size per symbol.

`metrics.Serve` exposes `/metrics` so dashboards can scrape the bot while it runs. Additionally, the paper daemon exposes an HTTP API on `:8081` providing `/paper/fills` and `/paper/account` for testers.

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
3. The feed streams live Binance ticks (or stub data in tests) onto a buffered channel inside a goroutine.
4. The strategy consumes ticks synchronously, transforms them into trading signals via imbalance + momentum heuristics, and emits metadata such as reasoning and timestamps.
5. Risk checks gate the downstream execution path.
6. The paper account validates bankroll/position limits, mutates balances on partial fills, updates realised/unrealised PnL, and records fills to the ledger/recorder.
7. Eligible orders are submitted through the executor, which logs intent, simulates slippage/latency/partials, and increments metrics.

## Testing Philosophy

Unit tests cover configuration loading, risk limit checks, logger behaviour, feed streaming, paper account transitions, execution fill modelling, and the Solana client request composition. Strategy tests validate that the OBIMomentum heuristic emits long/short signals when data supports it. Integration tests focus on the paper engine wiring (ensuring ticks flow to strategies and produce orders when thresholds are met).

As functionality hardens we will expand these suites with:

- Strategy-specific statistical backtests.
- Deterministic simulations for order book imbalance and volatility windows.
- Exchange contract tests verifying REST/WebSocket payloads.

## Outstanding Work

- Replace heuristic OBIMomentum logic with real order book imbalance processing and calibrated thresholds.
- Implement account/risk state tracking for drawdown limits and global kill switches.
- Promote the executor beyond logging: add actual REST/WS adapters plus order reconciliation.
- Extend the paper fills engine with order state machines, latency/slippage modelling, and persistence for analytics.
- Replace hand-rolled Binance client with pluggable connectors per venue (Bybit, OKX, etc.) and add reconnection telemetry.
- Extend DEX tooling with position swapping, quoting for multiple routes, and failure handling.
