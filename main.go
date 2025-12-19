package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/JimmyyyW/avrocado/internal/config"
	"github.com/JimmyyyW/avrocado/internal/kafka"
	"github.com/JimmyyyW/avrocado/internal/registry"
	"github.com/JimmyyyW/avrocado/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		fmt.Fprintln(os.Stderr, "\nRequired environment variables:")
		fmt.Fprintln(os.Stderr, "  SCHEMA_REGISTRY_URL - Schema registry URL (e.g., https://your-registry.confluent.cloud)")
		fmt.Fprintln(os.Stderr, "\nOptional environment variables:")
		fmt.Fprintln(os.Stderr, "  SCHEMA_REGISTRY_API_KEY    - API key for authentication")
		fmt.Fprintln(os.Stderr, "  SCHEMA_REGISTRY_API_SECRET - API secret for authentication")
		fmt.Fprintln(os.Stderr, "  KAFKA_BOOTSTRAP_SERVERS    - Kafka broker addresses (for message production)")
		fmt.Fprintln(os.Stderr, "  KAFKA_SASL_USERNAME        - SASL username (optional)")
		fmt.Fprintln(os.Stderr, "  KAFKA_SASL_PASSWORD        - SASL password (optional)")
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

	model := ui.NewModel(client, producer)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
