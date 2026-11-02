package port

import "context"

// EventPublisher defines the contract for sending asynchronous events to message brokers (e.g. Kafka).
type EventPublisher interface {
	Publish(ctx context.Context, topic string, key string, payload []byte) error
	Close() error
}
