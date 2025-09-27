// Package config also contains DEX-specific configuration surfaces.
package config

// Dex defines network endpoints and defaults for decentralized execution.
type Dex struct {
	Chain       string `yaml:"chain"` // e.g. "solana"
	RpcURL      string `yaml:"rpc_url"`
	Commitment  string `yaml:"commitment"`   // processed|confirmed|finalized
	JupiterBase string `yaml:"jupiter_base"` // https://quote-api.jup.ag
}

// Wallet stores encrypted or env-backed signing material metadata.
type Wallet struct {
	PrivateKeyBase58 string `yaml:"private_key_base58"`
}
