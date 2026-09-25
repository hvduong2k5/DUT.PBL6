package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	kafkaGo "github.com/segmentio/kafka-go"
)

type Producer interface {
	Publish(ctx context.Context, topic, key string, payload []byte) error
	PublishBatch(ctx context.Context, topic string, msgs []kafkaGo.Message) error
	Close() error
}

type KafkaProducer struct {
	writer *kafkaGo.Writer
}

func NewKafkaProducer(brokers string) *KafkaProducer {
	brokerList := strings.Split(brokers, ",")
	writer := &kafkaGo.Writer{
		Addr:         kafkaGo.TCP(brokerList...),
		Balancer:     &kafkaGo.LeastBytes{},
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
		RequiredAcks: kafkaGo.RequireOne, // Fast, reliable for Outbox worker
		Async:        false,
	}

	return &KafkaProducer{
		writer: writer,
	}
}

func (p *KafkaProducer) Publish(ctx context.Context, topic, key string, payload []byte) error {
	msg := kafkaGo.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: payload,
		Time:  time.Now().UTC(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed writing message to kafka topic %s: %w", topic, err)
	}

	return nil
}

func (p *KafkaProducer) PublishBatch(ctx context.Context, topic string, msgs []kafkaGo.Message) error {
	for i := range msgs {
		msgs[i].Topic = topic
		if msgs[i].Time.IsZero() {
			msgs[i].Time = time.Now().UTC()
		}
	}

	if err := p.writer.WriteMessages(ctx, msgs...); err != nil {
		return fmt.Errorf("failed writing message batch to kafka topic %s: %w", topic, err)
	}

	return nil
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
