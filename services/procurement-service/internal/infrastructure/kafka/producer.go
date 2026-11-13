package kafka

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/omamx/procurement-service/internal/domain"
	kafkago "github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafkago.Writer
}

func NewProducer(brokers []string) (*Producer, error) {
	clean := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		if value := strings.TrimSpace(broker); value != "" {
			clean = append(clean, value)
		}
	}
	if len(clean) == 0 {
		return nil, errors.New("at least one Kafka broker is required")
	}
	return &Producer{writer: &kafkago.Writer{
		Addr:         kafkago.TCP(clean...),
		Topic:        domain.GoodsReceivedEventTopic,
		Balancer:     &kafkago.Hash{},
		RequiredAcks: kafkago.RequireAll,
		Async:        false,
		BatchTimeout: 250 * time.Millisecond,
	}}, nil
}

func (p *Producer) Publish(ctx context.Context, topic, key string, payload []byte) error {
	if topic != domain.GoodsReceivedEventTopic || strings.TrimSpace(key) == "" || len(payload) == 0 {
		return errors.New("invalid goods-received Kafka message")
	}
	return p.writer.WriteMessages(ctx, kafkago.Message{Key: []byte(key), Value: payload})
}

func (p *Producer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
