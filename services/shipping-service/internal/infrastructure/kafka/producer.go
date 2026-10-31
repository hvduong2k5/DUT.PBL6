package kafka

import (
	"context"
	"sync"

	kafkago "github.com/segmentio/kafka-go"
)

type Producer struct {
	brokers []string
	mu      sync.Mutex
	writers map[string]*kafkago.Writer
}

func NewProducer(brokers []string) *Producer {
	return &Producer{brokers: brokers, writers: make(map[string]*kafkago.Writer)}
}

func (p *Producer) Publish(ctx context.Context, topic, key string, payload []byte) error {
	p.mu.Lock()
	writer := p.writers[topic]
	if writer == nil {
		writer = &kafkago.Writer{
			Addr: kafkago.TCP(p.brokers...), Topic: topic,
			Balancer: &kafkago.Hash{}, RequiredAcks: kafkago.RequireAll,
			Async: false,
		}
		p.writers[topic] = writer
	}
	p.mu.Unlock()
	return writer.WriteMessages(ctx, kafkago.Message{Key: []byte(key), Value: payload})
}

func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var first error
	for _, writer := range p.writers {
		if err := writer.Close(); err != nil && first == nil {
			first = err
		}
	}
	p.writers = make(map[string]*kafkago.Writer)
	return first
}
