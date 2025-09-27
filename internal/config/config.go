package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type App struct {
	Name        string
	Env         string
	MetricsAddr string
	LogLevel    string
}
type Exchange struct {
	Name      string
	Symbols   []string
	APIKey    string
	APISecret string
	Testnet   bool
}
type Risk struct {
	MaxNotionalPerTrade float64
	MaxDailyLoss        float64
	KillSwitchDrawdown  float64
}
type StrategyParams struct {
	OBILevels     int
	OBIThreshold  float64
	VolWindowSecs int
}
type Strategy struct {
	Mode   string
	Params StrategyParams
}

type Config struct {
	App      App      `yaml:"app"`
	Exchange Exchange `yaml:"exchange"`
	Risk     Risk     `yaml:"risk"`
	Strategy Strategy `yaml:"strategy"`
	Dex      Dex      `yaml:"dex"`
	Wallet   Wallet   `yaml:"wallet"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()
	var c Config
	if err := yaml.NewDecoder(f).Decode(&c); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}
	return &c, nil
}
