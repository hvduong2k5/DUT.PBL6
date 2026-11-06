package broker

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/segmentio/kafka-go"
)

type Kafka struct {
	writer    *kafka.Writer
	transport *kafka.Transport
}

// This plaintext adapter is only for the isolated sandbox broker. Integration
// TLS/SASL and shared broker policy remain explicit integration configuration.
func NewKafka(mode string, brokers []string) (*Kafka, error) {
	if mode != "sandbox" || len(brokers) == 0 {
		return nil, errors.New("sandbox broker endpoints required")
	}
	for _, addr := range brokers {
		host, port, err := net.SplitHostPort(addr)
		n, convert := strconv.Atoi(port)
		if err != nil || convert != nil || host == "" || n < 1 || n > 65535 || strings.ContainsAny(addr, "@/?#") {
			return nil, errors.New("invalid broker address")
		}
	}
	transport := &kafka.Transport{DialTimeout: 2 * time.Second, IdleTimeout: 10 * time.Second, ClientID: "traceability-service-sandbox"}
	writer := &kafka.Writer{Addr: kafka.TCP(brokers...), Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll, Async: false, MaxAttempts: 1, BatchSize: 1, BatchTimeout: time.Millisecond, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second, AllowAutoTopicCreation: false, Transport: transport}
	return &Kafka{writer: writer, transport: transport}, nil
}
func (k *Kafka) Publish(ctx context.Context, m domain.OutboxMessage) error {
	if k == nil || k.writer == nil {
		return domain.ErrUnavailable
	}
	if err := domain.ValidateEvent(m); err != nil {
		return err
	}
	return k.writer.WriteMessages(ctx, kafka.Message{Topic: m.Topic, Key: []byte(m.Key), Value: []byte(m.Payload), Headers: []kafka.Header{{Key: "ce_id", Value: []byte(m.ID)}, {Key: "ce_type", Value: []byte(m.Type)}}})
}
func (k *Kafka) Close() error {
	err := k.writer.Close()
	k.transport.CloseIdleConnections()
	return err
}
