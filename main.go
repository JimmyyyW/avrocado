package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/pflag"

	"github.com/JimmyyyW/avrocado/internal/config"
	"github.com/JimmyyyW/avrocado/internal/kafka"
	"github.com/JimmyyyW/avrocado/internal/registry"
	"github.com/JimmyyyW/avrocado/internal/ui"
)

func main() {
	// Parse command line flags
	selectConfig := pflag.BoolP("select-config", "s", false, "Show configuration selection menu")
	pflag.Parse()

	// Load configuration
	cfg, err := loadConfiguration(*selectConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	client := registry.NewClient(cfg)

	// Create Kafka producer if configured
	var producer *kafka.Producer
	if cfg.HasKafka() {
		producer, err = kafka.NewProducer(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not create Kafka producer: %v\n", err)
			fmt.Fprintln(os.Stderr, "Message production will be disabled.")
			producer = nil
		} else {
			defer producer.Close()
		}
	}

	model := ui.NewModel(client, producer, cfg)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

// loadConfiguration loads configuration from YAML file or environment variables
func loadConfiguration(selectConfig bool) (*config.Config, error) {
	configPath := config.GetConfigPath()
	configFile, err := config.LoadConfigFile(configPath)

	// If config file doesn't exist, create default
	if err != nil {
		if os.IsNotExist(err) {
			if err := config.CreateDefaultConfig(configPath); err != nil {
				return nil, fmt.Errorf("creating default config: %w", err)
			}
			configFile, _ = config.LoadConfigFile(configPath)
		} else {
			return nil, fmt.Errorf("loading config file: %w", err)
		}
	}

	var selectedProfile *config.ProfileConfig

	// Show selection menu if flag is set
	if selectConfig && configFile != nil && len(configFile.Configurations) > 0 {
		selector := ui.NewConfigSelector(configFile)
		p := tea.NewProgram(selector)
		model, _ := p.Run()
		if selectorModel, ok := model.(ui.ConfigSelectorModel); ok {
			selectedProfile = selectorModel.SelectedProfile()
		}
	}

	// If no profile selected, use default
	if selectedProfile == nil && configFile != nil {
		selectedProfile, err = configFile.GetProfile(configFile.Default)
		if err != nil {
			// Fall back to environment variables
			return config.Load()
		}
	}

	// If still no profile, fall back to environment variables
	if selectedProfile == nil {
		return config.Load()
	}

	return selectedProfile.ToConfig(), nil
}
