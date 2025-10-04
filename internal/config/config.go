// Package config exposes strongly typed application configuration structs loaded from YAML.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// App captures process-wide runtime settings such as name, environment, metrics, and logging levels.
type App struct {
	Name        string
	Env         string
	MetricsAddr string
	LogLevel    string
}

// Exchange describes the centralized exchange connectivity parameters the bot expects.
type Exchange struct {
	Name        string
	Symbols     []string
	APIKey      string
	APISecret   string
	Testnet     bool
	DexScreener DexScreener `yaml:"dexscreener"`
	Discovery   Discovery   `yaml:"discovery"`
}

// DexScreener configures the HTTP polling feed targeting Dexscreener pairs.
type DexScreener struct {
	BaseURL      string `yaml:"base_url"`
	DefaultChain string `yaml:"default_chain"`
	PollInterval int    `yaml:"poll_interval_ms"`
}

// Discovery configures automatic symbol discovery.
type Discovery struct {
	Enabled            bool     `yaml:"enabled"`
	Keywords           []string `yaml:"keywords"`
	Chains             []string `yaml:"chains"`
	MaxPairs           int      `yaml:"max_pairs"`
	RefreshInterval    int      `yaml:"refresh_interval_ms"`
	MinLiquidityUSD    float64  `yaml:"min_liquidity_usd"`
	MinVolumeUSD       float64  `yaml:"min_volume_usd"`
	MaxPairsPerKeyword int      `yaml:"max_pairs_per_keyword"`
}

// Risk encodes guard-rails for how much size the executor may take on.
type Risk struct {
	MaxNotionalPerTrade  float64 `yaml:"max_notional_per_trade"`
	MaxDailyLoss         float64 `yaml:"max_daily_loss"`
	KillSwitchDrawdown   float64 `yaml:"kill_switch_drawdown"`
	MaxPortfolioNotional float64 `yaml:"max_portfolio_notional"`
}

// StrategyParams groups tunable knobs for a strategy implementation.
type StrategyParams struct {
	OBILevels         int
	OBIThreshold      float64
	VolWindowSecs     int
	TrendThreshold    float64 `yaml:"trend_threshold"`
	TrendWindowSecs   int     `yaml:"trend_window_secs"`
	TrendMinVolumeUSD float64 `yaml:"trend_min_volume_usd"`
}

// Strategy specifies which strategy is active along with the parameter bundle.
type Strategy struct {
	Mode   string
	Params StrategyParams
}

// Paper captures paper-trading account settings such as starting cash, per-symbol caps, and execution tuning.
type Paper struct {
	StartingCash           float64 `yaml:"starting_cash"`
	MaxPositionPerSymbol   float64 `yaml:"max_position_per_symbol"`
	MaxPositionNotionalUSD float64 `yaml:"max_position_notional_usd"`
	SlippageBps            float64 `yaml:"slippage_bps"`
	MaxLatencyMs           int     `yaml:"max_latency_ms"`
	PartialFillProbability float64 `yaml:"partial_fill_probability"`
	MaxPartialFills        int     `yaml:"max_partial_fills"`
	FillsPath              string  `yaml:"fills_path"`
}

// Config collects every configuration leaf for easy marshaling from YAML.
type Config struct {
	App      App      `yaml:"app"`
	Exchange Exchange `yaml:"exchange"`
	Risk     Risk     `yaml:"risk"`
	Strategy Strategy `yaml:"strategy"`
	Dex      Dex      `yaml:"dex"`
	Wallet   Wallet   `yaml:"wallet"`
	Paper    Paper    `yaml:"paper"`
}

// Load reads a YAML file from disk and hydrates a Config struct.
func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	var config Config
	if err := yaml.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}
	return &config, nil
}

// Save persists a Config struct to disk as YAML.
func Save(path string, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("nil config")
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
