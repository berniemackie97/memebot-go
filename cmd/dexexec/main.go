package main

import (
	"context"
	"log"
	"os"
	"time"

	"memebot-go/internal/config"
	dex "memebot-go/internal/dex/solana"
)

func main() {
	cfg, err := config.Load("internal/config/config.yaml")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	SolanaPrivateKey, err := dex.LoadPrivateKeyFromEnv()
	if err != nil {
		log.Fatalf("wallet: %v", err)
	}

	JupiterClient := dex.NewJupiterClient(
		getEnv("SOLANA_RPC_URL", cfg.Dex.RpcURL),
		getEnv("JUPITER_BASE_URL", cfg.Dex.JupiterBase),
		SolanaPrivateKey,
		getEnv("SOLANA_COMMITMENT", cfg.Dex.Commitment),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Example: swap 0.01 SOL -> USDC
	const (
		SOL  = "So11111111111111111111111111111111111111112"
		USDC = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	)
	amountLamports := uint64(10_000_000)                                      // 0.01 SOL
	quote, err := JupiterClient.GetQuote(ctx, SOL, USDC, amountLamports, 150) // 1.5% slippage
	if err != nil {
		log.Fatalf("quote: %v", err)
	}

	sig, err := JupiterClient.BuildAndSendSwap(ctx, quote)
	if err != nil {
		log.Fatalf("swap: %v", err)
	}
	log.Printf("submitted tx: %s", sig.String())
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
