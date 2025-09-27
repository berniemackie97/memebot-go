package config

type Dex struct {
  Chain       string `yaml:"chain"`        // "solana"
  RpcURL      string `yaml:"rpc_url"`
  Commitment  string `yaml:"commitment"`   // processed|confirmed|finalized
  JupiterBase string `yaml:"jupiter_base"` // https://quote-api.jup.ag
}

type Wallet struct {
  PrivateKeyBase58 string `yaml:"private_key_base58"`
}
