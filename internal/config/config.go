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
	Name      string
	Symbols   []string
	APIKey    string
	APISecret string
	Testnet   bool
}

// Risk encodes guard-rails for how much size the executor may take on.
type Risk struct {
	MaxNotionalPerTrade float64
	MaxDailyLoss        float64
	KillSwitchDrawdown  float64
}

// StrategyParams groups tunable knobs for a strategy implementation.
type StrategyParams struct {
	OBILevels     int
	OBIThreshold  float64
	VolWindowSecs int
}

// Strategy specifies which strategy is active along with the parameter bundle.
type Strategy struct {
	Mode   string
	Params StrategyParams
}

// Config collects every configuration leaf for easy marshaling from YAML.
type Config struct {
	App      App      `yaml:"app"`
	Exchange Exchange `yaml:"exchange"`
	Risk     Risk     `yaml:"risk"`
	Strategy Strategy `yaml:"strategy"`
	Dex      Dex      `yaml:"dex"`
	Wallet   Wallet   `yaml:"wallet"`
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
