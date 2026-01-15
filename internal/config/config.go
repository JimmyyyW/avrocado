package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Legacy Config struct for backward compatibility and internal usage
type Config struct {
	// Schema Registry
	RegistryURL string
	APIKey      string
	APISecret   string

	// Kafka
	KafkaBootstrapServers string
	KafkaSASLUsername     string
	KafkaSASLPassword     string
	KafkaSecurityProtocol string
}

// ConfigFile represents the YAML configuration file structure
type ConfigFile struct {
	Default        string                     `yaml:"default"`
	Configurations map[string]*ProfileConfig `yaml:"configurations"`
}

// ProfileConfig represents a named configuration profile
type ProfileConfig struct {
	Name           string                 `yaml:"name"`
	SchemaRegistry SchemaRegistryConfig   `yaml:"schema_registry"`
	Kafka          KafkaConfig            `yaml:"kafka"`
}

// SchemaRegistryConfig holds Schema Registry settings
type SchemaRegistryConfig struct {
	URL              string `yaml:"url"`
	AuthMethod       string `yaml:"auth_method,omitempty"` // "none", "basic", "sasl"
	APIKey           string `yaml:"api_key,omitempty"`     // For basic auth
	APISecret        string `yaml:"api_secret,omitempty"`  // For basic auth
	SASLUsername     string `yaml:"sasl_username,omitempty"`
	SASLPassword     string `yaml:"sasl_password,omitempty"`
	SecurityProtocol string `yaml:"security_protocol,omitempty"` // For SASL connections
}

// KafkaConfig holds Kafka settings
type KafkaConfig struct {
	BootstrapServers string `yaml:"bootstrap_servers"`
	SecurityProtocol string `yaml:"security_protocol"`
	SASLMechanism    string `yaml:"sasl_mechanism,omitempty"`
	SASLUsername     string `yaml:"sasl_username,omitempty"`
	SASLPassword     string `yaml:"sasl_password,omitempty"`
}

// Load loads configuration from environment variables (legacy mode)
func Load() (*Config, error) {
	url := os.Getenv("SCHEMA_REGISTRY_URL")
	if url == "" {
		return nil, errors.New("SCHEMA_REGISTRY_URL environment variable is required")
	}

	apiKey := os.Getenv("SCHEMA_REGISTRY_API_KEY")
	apiSecret := os.Getenv("SCHEMA_REGISTRY_API_SECRET")

	kafkaServers := os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	kafkaUsername := os.Getenv("KAFKA_SASL_USERNAME")
	kafkaPassword := os.Getenv("KAFKA_SASL_PASSWORD")
	kafkaProtocol := os.Getenv("KAFKA_SECURITY_PROTOCOL")
	if kafkaProtocol == "" {
		kafkaProtocol = "PLAINTEXT"
	}

	return &Config{
		RegistryURL:           url,
		APIKey:                apiKey,
		APISecret:             apiSecret,
		KafkaBootstrapServers: kafkaServers,
		KafkaSASLUsername:     kafkaUsername,
		KafkaSASLPassword:     kafkaPassword,
		KafkaSecurityProtocol: kafkaProtocol,
	}, nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback if home dir can't be determined
		return filepath.Join(".", ".config", "avrocado", "config.yaml")
	}
	return filepath.Join(home, ".config", "avrocado", "config.yaml")
}

// LoadConfigFile loads configuration from YAML file
func LoadConfigFile(path string) (*ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ConfigFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return &cfg, nil
}

// CreateDefaultConfig creates a default config file if it doesn't exist
func CreateDefaultConfig(path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Create default config
	cfg := &ConfigFile{
		Default: "local",
		Configurations: map[string]*ProfileConfig{
			"local": {
				Name: "Local Development",
				SchemaRegistry: SchemaRegistryConfig{
					URL: "http://localhost:8081",
				},
				Kafka: KafkaConfig{
					BootstrapServers: "localhost:9092",
					SecurityProtocol: "PLAINTEXT",
				},
			},
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Write to file with restricted permissions
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// GetProfile retrieves a profile by name
func (cf *ConfigFile) GetProfile(name string) (*ProfileConfig, error) {
	if profile, ok := cf.Configurations[name]; ok {
		return profile, nil
	}
	return nil, fmt.Errorf("profile %q not found", name)
}

// ToConfig converts a ProfileConfig to a legacy Config struct
func (pc *ProfileConfig) ToConfig() *Config {
	return &Config{
		RegistryURL:           pc.SchemaRegistry.URL,
		APIKey:                pc.SchemaRegistry.APIKey,
		APISecret:             pc.SchemaRegistry.APISecret,
		KafkaBootstrapServers: pc.Kafka.BootstrapServers,
		KafkaSASLUsername:     pc.Kafka.SASLUsername,
		KafkaSASLPassword:     pc.Kafka.SASLPassword,
		KafkaSecurityProtocol: pc.Kafka.SecurityProtocol,
	}
}

func (c *Config) HasAuth() bool {
	return c.APIKey != "" && c.APISecret != ""
}

func (c *Config) HasKafka() bool {
	return c.KafkaBootstrapServers != ""
}

// SubjectToTopic converts a schema registry subject name to a Kafka topic.
// It strips the -value or -key suffix if present.
func SubjectToTopic(subject string) string {
	if strings.HasSuffix(subject, "-value") {
		return strings.TrimSuffix(subject, "-value")
	}
	if strings.HasSuffix(subject, "-key") {
		return strings.TrimSuffix(subject, "-key")
	}
	return subject
}
