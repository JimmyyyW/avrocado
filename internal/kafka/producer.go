package kafka

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

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

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.KafkaBootstrapServers),
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireOne,
	}

	// TODO: Add SASL support when security protocol is SASL_SSL or SASL_PLAINTEXT
	// This would require configuring the Dialer with SASL mechanism

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
