package solana

import (
	"os"
	"testing"

	solana "github.com/gagliardetto/solana-go"
)

func TestLoadPrivateKeyFromEnv(t *testing.T) {
	wallet := solana.NewWallet()
	os.Setenv("SOLANA_PRIVATE_KEY_BASE58", wallet.PrivateKey.String())
	defer os.Unsetenv("SOLANA_PRIVATE_KEY_BASE58")

	key, err := LoadPrivateKeyFromEnv()
	if err != nil {
		t.Fatalf("expected key, got error: %v", err)
	}
	if !key.PublicKey().Equals(wallet.PublicKey()) {
		t.Fatalf("expected public key %s, got %s", wallet.PublicKey(), key.PublicKey())
	}
}

func TestLoadPrivateKeyFromEnvMissing(t *testing.T) {
	os.Unsetenv("SOLANA_PRIVATE_KEY_BASE58")
	if _, err := LoadPrivateKeyFromEnv(); err == nil {
		t.Fatalf("expected error when env missing")
	}
}
