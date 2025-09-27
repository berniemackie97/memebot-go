package solana

import (
	"errors"
	"os"

	solana "github.com/gagliardetto/solana-go"
	"github.com/joho/godotenv"
)

// LoadPrivateKeyFromEnv retrieves a base58 private key and converts it to a Solana keypair.
func LoadPrivateKeyFromEnv() (solana.PrivateKey, error) {
	_ = godotenv.Load() // best-effort load from .env if present.
	b58 := os.Getenv("SOLANA_PRIVATE_KEY_BASE58")
	if b58 == "" {
		return nil, errors.New("SOLANA_PRIVATE_KEY_BASE58 not set")
	}
	return solana.PrivateKeyFromBase58(b58)
}
