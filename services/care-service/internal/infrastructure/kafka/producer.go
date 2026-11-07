package kafka

import (
	"context"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type Producer interface {
	Publish(context.Context, string, string, []byte) error
	Close() error
}

type WriterProducer struct {
	brokers []string
	writers map[string]*kafkago.Writer
}

func NewProducer(brokers []string) *WriterProducer {
	return &WriterProducer{brokers: brokers, writers: make(map[string]*kafkago.Writer)}
}

func (p *WriterProducer) Publish(ctx context.Context, topic, key string, value []byte) error {
	writer, ok := p.writers[topic]
	if !ok {
		writer = &kafkago.Writer{
			Addr:         kafkago.TCP(p.brokers...),
			Topic:        topic,
			Balancer:     &kafkago.Hash{},
			RequiredAcks: kafkago.RequireAll,
			Async:        false,
			BatchTimeout: 10 * time.Millisecond,
			WriteTimeout: 10 * time.Second,
			ReadTimeout:  10 * time.Second,
		}
		p.writers[topic] = writer
	}
	return writer.WriteMessages(ctx, kafkago.Message{Key: []byte(key), Value: value, Time: time.Now().UTC()})
}

func (p *WriterProducer) Close() error {
	var firstErr error
	for _, writer := range p.writers {
		if err := writer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
