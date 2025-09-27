package solana

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
	solana "github.com/gagliardetto/solana-go"
)

func LoadPrivateKeyFromEnv() (solana.PrivateKey, error) {
	_ = godotenv.Load() // best-effort
	b58 := os.Getenv("SOLANA_PRIVATE_KEY_BASE58")
	if b58 == "" {
		return nil, errors.New("SOLANA_PRIVATE_KEY_BASE58 not set")
	}
	return solana.PrivateKeyFromBase58(b58)
}
