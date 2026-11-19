package app

import (
	"context"
	"errors"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/shippingprovider"
	"github.com/segmentio/kafka-go"
	"log/slog"
	"os"
	"time"
)

func (s *Service) EnableShippingFromEnv(ctx context.Context) error {
	target := os.Getenv("ORDER_SHIPPING_GRPC")
	if target == "" {
		for _, key := range []string{"ORDER_SHIPPING_TOKEN_FILE", "ORDER_SHIPPING_CA_FILE", "ORDER_SHIPPING_SERVER_NAME", "ORDER_SHIPPING_ALLOW_LOCAL_INSECURE"} {
			if os.Getenv(key) != "" {
				return errors.New("Shipping configuration requires ORDER_SHIPPING_GRPC")
			}
		}
		return nil
	}
	if s.C.Kafka == "" {
		return errors.New("Shipping event consumption requires KAFKA_BROKERS")
	}
	token, err := shippingprovider.TokenFile(os.Getenv("ORDER_SHIPPING_TOKEN_FILE"))
	if err != nil {
		return err
	}
	adapter, conn, err := shippingprovider.Dial(ctx, target, token, os.Getenv("ORDER_SHIPPING_CA_FILE"), os.Getenv("ORDER_SHIPPING_SERVER_NAME"), s.C.Mode == "sandbox" && os.Getenv("ORDER_SHIPPING_ALLOW_LOCAL_INSECURE") == "true")
	if err != nil {
		return err
	}
	s.Shipping = adapter
	prior := s.PaymentClose
	s.PaymentClose = func() error {
		if prior != nil {
			_ = prior()
		}
		return conn.Close()
	}
	return nil
}

func (s *Service) ProcessShippingNotification(ctx context.Context, raw []byte) (string, error) {
	if s.Shipping == nil || s.CoreCheckout == nil {
		return "", domain.E("SHIPPING_PROVIDER_UNAVAILABLE", 503)
	}
	event, err := s.Shipping.Notification(ctx, raw)
	if errors.Is(err, shippingprovider.ErrIgnored) {
		return "IGNORED", nil
	}
	if err != nil {
		return "", err
	}
	event.Signature = domain.EventSignature(event, s.C.InternalToken)
	return s.ProcessEvent(ctx, event)
}

func (s *Service) consumeShipping(ctx context.Context) {
	defer s.stop.Done()
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: splitBrokers(s.C.Kafka), Topic: "shipping.events.v2", GroupID: consumerGroup(s.C) + "-shipping-v2", StartOffset: kafka.FirstOffset, MinBytes: 1, MaxBytes: 1 << 20, CommitInterval: 0})
	defer reader.Close()
	for ctx.Err() == nil {
		message, err := reader.FetchMessage(ctx)
		if err != nil {
			continue
		}
		for ctx.Err() == nil {
			_, err = s.ProcessShippingNotification(ctx, message.Value)
			if err != nil {
				_, code := s.ErrorCode(err)
				if code >= 500 {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
						continue
					}
				}
				slog.Warn("shipping notification rejected", "code", safeCode(s, err))
			}
			if err = reader.CommitMessages(ctx, message); err == nil {
				break
			}
		}
	}
}
