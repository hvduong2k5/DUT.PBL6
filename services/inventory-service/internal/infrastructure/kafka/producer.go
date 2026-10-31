package kafka

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

// KafkaProducer implements port.EventPublisher using segmentio/kafka-go.
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer creates a new KafkaProducer connected to the specified brokers.
func NewKafkaProducer(brokers []string) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	return &KafkaProducer{writer: writer}
}

// Publish writes a message to the target Kafka topic.
func (p *KafkaProducer) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: payload,
	}
	return p.writer.WriteMessages(ctx, msg)
}

// Close closes the underlying Kafka writer.
func (p *KafkaProducer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// LogEventPublisher is a mock/fallback publisher when Kafka broker is not configured.
type LogEventPublisher struct{}

func NewLogEventPublisher() *LogEventPublisher {
	return &LogEventPublisher{}
}

func (p *LogEventPublisher) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	log.Printf("[LogEventPublisher] Published event to topic '%s', key '%s', size %d bytes", topic, key, len(payload))
	return nil
}

func (p *LogEventPublisher) Close() error {
	return nil
}
