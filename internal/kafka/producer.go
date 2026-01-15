package kafka

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
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
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	// Configure SASL_SSL if needed
	if cfg.KafkaSecurityProtocol == "SASL_SSL" {
		// Configure TLS with system CA certificates
		dialer.TLS = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		// Configure SASL PLAIN mechanism (for Confluent Cloud)
		if cfg.KafkaSASLUsername != "" && cfg.KafkaSASLPassword != "" {
			dialer.SASLMechanism = plain.Mechanism{
				Username: cfg.KafkaSASLUsername,
				Password: cfg.KafkaSASLPassword,
			}
		}
	}

	// Create writer with configured dialer
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.KafkaBootstrapServers),
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireOne,
	}

	// If we have a dialer with special config, use transport
	if dialer.TLS != nil || dialer.SASLMechanism != nil {
		writer.Transport = &kafka.Transport{
			Dial: dialer.DialFunc,
		}
	}

	return &Producer{writer: writer}, nil
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
