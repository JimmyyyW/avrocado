package config

import (
	"errors"
	"os"
)

type Config struct {
	RegistryURL string
	APIKey      string
	APISecret   string
}

func Load() (*Config, error) {
	url := os.Getenv("SCHEMA_REGISTRY_URL")
	if url == "" {
		return nil, errors.New("SCHEMA_REGISTRY_URL environment variable is required")
	}

	apiKey := os.Getenv("SCHEMA_REGISTRY_API_KEY")
	apiSecret := os.Getenv("SCHEMA_REGISTRY_API_SECRET")

	return &Config{
		RegistryURL: url,
		APIKey:      apiKey,
		APISecret:   apiSecret,
	}, nil
}

func (c *Config) HasAuth() bool {
	return c.APIKey != "" && c.APISecret != ""
}
