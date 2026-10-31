package kafka_test

import (
	"context"
	"testing"

	kafkainfra "dut-pbl6/inventory-service/internal/infrastructure/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogEventPublisher(t *testing.T) {
	pub := kafkainfra.NewLogEventPublisher()
	require.NotNil(t, pub)

	ctx := context.Background()
	err := pub.Publish(ctx, "test.topic", "key-1", []byte(`{"test":true}`))
	assert.NoError(t, err)

	err = pub.Close()
	assert.NoError(t, err)
}

func TestKafkaProducer_Close_NilWriter(t *testing.T) {
	pub := kafkainfra.NewKafkaProducer([]string{"localhost:9092"})
	require.NotNil(t, pub)
	err := pub.Close()
	assert.NoError(t, err)
}
