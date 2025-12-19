package config

import (
	"errors"
	"os"
	"strings"
)

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
