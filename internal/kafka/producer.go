package kafka

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"

	"github.com/JimmyyyW/avrocado/internal/config"
)

// Producer wraps a Kafka producer with Avro serialization support.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a new Kafka producer from config.
func NewProducer(cfg *config.Config) (*Producer, error) {
	if cfg.KafkaBootstrapServers == "" {
		return nil, fmt.Errorf("KAFKA_BOOTSTRAP_SERVERS not configured")
	}

	// Create dialer with optional SASL/TLS support
	dialer, err := newDialer(cfg)
	if err != nil {
		return nil, fmt.Errorf("dialer error: %w", err)
	}

	// Create writer with configured dialer
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{cfg.KafkaBootstrapServers},
		Dialer: dialer,
		Balancer: &kafka.LeastBytes{},
		RequiredAcks: int(kafka.RequireAll),
	})

	return &Producer{writer: writer}, nil
}

func newDialer(cfg *config.Config) (*kafka.Dialer, error) {
	dialer := &kafka.Dialer{
		Timeout: 10 * time.Second,
		DualStack: true,
	}

	switch strings.ToUpper(cfg.KafkaSecurityProtocol) {
	case "PLAINTEXT":
		return dialer, nil
	case "SASL_SSL":
		if cfg.KafkaSASLUsername == "" || cfg.KafkaSASLPassword == "" {
			return nil, fmt.Errorf("SASL creds missing")
		}

		dialer.SASLMechanism = plain.Mechanism{
			Username: cfg.KafkaSASLUsername,
			Password: cfg.KafkaSASLPassword,
		}

		dialer.TLS = &tls.Config{}
		return dialer, nil

	default:
		return nil, fmt.Errorf("unsupported kafka security protocol")
	}
}

// Produce sends a message to the specified topic.
// The value should be Avro binary data (without wire format header).
// schemaID is used to prepend the Schema Registry wire format header.
func (p *Producer) Produce(ctx context.Context, topic string, schemaID int, key, value []byte) error {
	// Prepend Schema Registry wire format:
	// - Magic byte (0x00)
	// - Schema ID (4 bytes, big-endian)
	wireValue := make([]byte, 5+len(value))
	wireValue[0] = 0x00 // Magic byte
	binary.BigEndian.PutUint32(wireValue[1:5], uint32(schemaID))
	copy(wireValue[5:], value)

	msg := kafka.Message{
		Topic: topic,
		Value: wireValue,
	}

	if key != nil {
		msg.Key = key
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("producing message: %w", err)
	}

	return nil
}

// ProduceWithStringKey sends a message with a string key.
func (p *Producer) ProduceWithStringKey(ctx context.Context, topic string, schemaID int, key string, value []byte) error {
	var keyBytes []byte
	if key != "" {
		keyBytes = []byte(key)
	}
	return p.Produce(ctx, topic, schemaID, keyBytes, value)
}

// Close closes the producer.
func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
