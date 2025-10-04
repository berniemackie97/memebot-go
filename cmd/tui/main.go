package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"memebot-go/internal/config"
)

const defaultConfigPath = "internal/config/config.yaml"

func main() {
	reader := bufio.NewReader(os.Stdin)

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	for {
		fmt.Println("\n=== MemeBot Control ===")
		fmt.Println("1) Show configuration summary")
		fmt.Println("2) Edit bankroll and risk knobs")
		fmt.Println("3) Edit discovery settings")
		fmt.Println("4) Save config")
		fmt.Println("5) Launch paper bot")
		fmt.Println("6) Reload config from disk")
		fmt.Println("0) Exit")
		fmt.Print("Select option: ")

		input, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(input)

		switch choice {
		case "1":
			printSummary(cfg)
		case "2":
			editRisk(reader, cfg)
		case "3":
			editDiscovery(reader, cfg)
		case "4":
			if err := saveConfig(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "save failed: %v\n", err)
			} else {
				fmt.Println("config saved")
			}
		case "5":
			launchPaper(reader)
		case "6":
			reloaded, err := loadConfig()
			if err != nil {
				fmt.Fprintf(os.Stderr, "reload failed: %v\n", err)
			} else {
				cfg = reloaded
				fmt.Println("config reloaded")
			}
		case "0":
			return
		default:
			fmt.Println("unknown option")
		}
	}
}

func printSummary(cfg *config.Config) {
	fmt.Println("\n--- Configuration Summary ---")
	fmt.Printf("Starting cash: $%.2f\n", cfg.Paper.StartingCash)
	fmt.Printf("Per-symbol notional cap: $%.2f\n", cfg.Paper.MaxPositionNotionalUSD)
	fmt.Printf("Per-trade notional cap: $%.2f\n", cfg.Risk.MaxNotionalPerTrade)
	fmt.Printf("Portfolio notional cap: $%.2f\n", cfg.Risk.MaxPortfolioNotional)
	fmt.Printf("Daily loss limit: $%.2f\n", cfg.Risk.MaxDailyLoss)
	fmt.Printf("Kill switch drawdown: %.2f%%\n", cfg.Risk.KillSwitchDrawdown*100)
	fmt.Println("Discovery keywords:", strings.Join(cfg.Exchange.Discovery.Keywords, ", "))
	fmt.Printf("Discovery max pairs: %d (per keyword %d)\n", cfg.Exchange.Discovery.MaxPairs, cfg.Exchange.Discovery.MaxPairsPerKeyword)
	fmt.Printf("Discovery min liquidity: $%.0f | min volume: $%.0f\n", cfg.Exchange.Discovery.MinLiquidityUSD, cfg.Exchange.Discovery.MinVolumeUSD)
}

func editRisk(reader *bufio.Reader, cfg *config.Config) {
	fmt.Println("\n--- Edit Risk / Bankroll ---")
	cfg.Paper.StartingCash = promptFloat(reader, "Starting cash", cfg.Paper.StartingCash)
	cfg.Paper.MaxPositionNotionalUSD = promptFloat(reader, "Max position notional (USD)", cfg.Paper.MaxPositionNotionalUSD)
	cfg.Risk.MaxNotionalPerTrade = promptFloat(reader, "Max notional per trade (USD)", cfg.Risk.MaxNotionalPerTrade)
	cfg.Risk.MaxPortfolioNotional = promptFloat(reader, "Max portfolio notional (USD)", cfg.Risk.MaxPortfolioNotional)
	cfg.Risk.MaxDailyLoss = promptFloat(reader, "Max daily loss (USD)", cfg.Risk.MaxDailyLoss)
	cfg.Risk.KillSwitchDrawdown = promptPercent(reader, "Kill switch drawdown (%)", cfg.Risk.KillSwitchDrawdown)
}

func editDiscovery(reader *bufio.Reader, cfg *config.Config) {
	fmt.Println("\n--- Edit Discovery ---")
	fmt.Printf("Current keywords: %s\n", strings.Join(cfg.Exchange.Discovery.Keywords, ", "))
	fmt.Print("Enter keywords comma-separated (blank to keep): ")
	if line, _ := reader.ReadString('\n'); strings.TrimSpace(line) != "" {
		parts := strings.Split(strings.TrimSpace(line), ",")
		cfg.Exchange.Discovery.Keywords = nil
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				cfg.Exchange.Discovery.Keywords = append(cfg.Exchange.Discovery.Keywords, trimmed)
			}
		}
	}
	cfg.Exchange.Discovery.MaxPairs = int(promptFloat(reader, "Max pairs overall", float64(cfg.Exchange.Discovery.MaxPairs)))
	cfg.Exchange.Discovery.MaxPairsPerKeyword = int(promptFloat(reader, "Max pairs per keyword", float64(cfg.Exchange.Discovery.MaxPairsPerKeyword)))
	cfg.Exchange.Discovery.MinLiquidityUSD = promptFloat(reader, "Min liquidity (USD)", cfg.Exchange.Discovery.MinLiquidityUSD)
	cfg.Exchange.Discovery.MinVolumeUSD = promptFloat(reader, "Min volume (USD)", cfg.Exchange.Discovery.MinVolumeUSD)
}

func launchPaper(reader *bufio.Reader) {
	fmt.Println("Launching paper bot (Ctrl+C to stop)...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/paper")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start bot: %v\n", err)
		return
	}

	go func() {
		_ = cmd.Wait()
		cancel()
	}()

	fmt.Print("\nPress ENTER to stop the bot and return to menu...")
	_, _ = reader.ReadString('\n')
	cancel()
	time.Sleep(500 * time.Millisecond)
}

func promptFloat(reader *bufio.Reader, label string, current float64) float64 {
	fmt.Printf("%s [%.2f]: ", label, current)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return current
	}
	val, err := strconv.ParseFloat(line, 64)
	if err != nil {
		fmt.Printf("invalid number, keeping %.2f\n", current)
		return current
	}
	return val
}

func promptPercent(reader *bufio.Reader, label string, current float64) float64 {
	pct := promptFloat(reader, label, current*100)
	return pct / 100
}

func loadConfig() (*config.Config, error) {
	return config.Load(locateConfig())
}

func saveConfig(cfg *config.Config) error {
	return config.Save(locateConfig(), cfg)
}

func locateConfig() string {
	if filepath.IsAbs(defaultConfigPath) {
		return defaultConfigPath
	}
	return filepath.Clean(defaultConfigPath)
}
