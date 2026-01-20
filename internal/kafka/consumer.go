package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"

	"github.com/JimmyyyW/avrocado/internal/config"
)

// Message represents a Kafka message
type Message struct {
	Key       string
	Value     string
	Offset    int64
	Timestamp time.Time
}

// Consumer wraps a Kafka consumer for reading messages
type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer creates a new Kafka consumer for the given topic
func NewConsumer(cfg *config.Config, topic string) (*Consumer, error) {
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

	// Create reader with configured dialer
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{cfg.KafkaBootstrapServers},
		Topic:       topic,
		Dialer:      dialer,
		StartOffset: kafka.LastOffset, // Start from the end, will seek to beginning
	})

	// Seek to beginning
	reader.SetOffset(0)

	return &Consumer{reader: reader}, nil
}

// FetchMessages fetches up to maxMessages from the topic
func (c *Consumer) FetchMessages(ctx context.Context, maxMessages int) ([]Message, error) {
	messages := []Message{}

	for i := 0; i < maxMessages; i++ {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			// No more messages available, return what we have
			if err == context.DeadlineExceeded {
				break
			}
			// If it's the first message and we get an error, return it
			if len(messages) == 0 {
				return nil, err
			}
			break
		}

		messages = append(messages, Message{
			Key:       string(msg.Key),
			Value:     string(msg.Value),
			Offset:    msg.Offset,
			Timestamp: msg.Time,
		})
	}

	return messages, nil
}

// Close closes the consumer
func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}
