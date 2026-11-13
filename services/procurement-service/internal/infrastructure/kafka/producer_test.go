package kafka_test

import (
	"context"
	"testing"

	kafkainfra "github.com/omamx/procurement-service/internal/infrastructure/kafka"
)

func TestNewProducerRequiresAtLeastOneBroker(t *testing.T) {
	if _, err := kafkainfra.NewProducer([]string{" ", ""}); err == nil {
		t.Fatal("expected empty broker list to fail")
	}
}

func TestPublishRejectsInvalidMessageWithoutConnecting(t *testing.T) {
	producer, err := kafkainfra.NewProducer([]string{"127.0.0.1:1"})
	if err != nil {
		t.Fatalf("NewProducer() error = %v", err)
	}
	defer producer.Close()
	if err := producer.Publish(context.Background(), "unexpected.topic", "SKU-1", []byte(`{}`)); err == nil {
		t.Fatal("expected unexpected topic to be rejected")
	}
	if err := producer.Publish(context.Background(), "procurement.events.v1", "", []byte(`{}`)); err == nil {
		t.Fatal("expected empty key to be rejected")
	}
}
