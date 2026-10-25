package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/domain"
	"github.com/rs/zerolog"
	kafkago "github.com/segmentio/kafka-go"
)

const packingCompletedType = "vn.omama.fulfillment.packing.completed.v1"

type packingEnvelope struct {
	SpecVersion string      `json:"specversion"`
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	Data        packingData `json:"data"`
}

type packingData struct {
	OrderID   string        `json:"order_id"`
	PackageID string        `json:"package_id"`
	SealCode  string        `json:"seal_code"`
	Parcel    domain.Parcel `json:"parcel"`
}

type PackingConsumer struct {
	reader         *kafkago.Reader
	service        *application.Service
	snapshots      application.ShippingSnapshotProvider
	defaultCarrier string
	defaultService string
	retryInterval  time.Duration
	logger         zerolog.Logger
}

func NewPackingConsumer(brokers []string, topic, group string, service *application.Service, snapshots application.ShippingSnapshotProvider, defaultCarrier, defaultService string, logger zerolog.Logger) *PackingConsumer {
	return &PackingConsumer{
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers: brokers, Topic: topic, GroupID: group,
			MinBytes: 1, MaxBytes: 10e6, CommitInterval: 0,
		}),
		service: service, snapshots: snapshots,
		defaultCarrier: defaultCarrier, defaultService: defaultService,
		retryInterval: 2 * time.Second, logger: logger,
	}
}

func (c *PackingConsumer) Close() error { return c.reader.Close() }

func (c *PackingConsumer) Run(ctx context.Context) {
	for {
		message, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				c.logger.Error().Err(err).Msg("fetch PackingCompleted event")
			}
			return
		}
		if err := c.handle(message.Value); err != nil {
			c.logger.Error().Err(err).Int("partition", message.Partition).Int64("offset", message.Offset).Msg("invalid fulfillment event skipped")
			if commitErr := c.reader.CommitMessages(ctx, message); commitErr != nil {
				c.logger.Error().Err(commitErr).Msg("commit invalid fulfillment event")
				return
			}
			continue
		}

		for {
			event, _ := decodePackingCompleted(message.Value)
			_, err = c.service.CreateFromPacking(ctx, event, c.snapshots, c.defaultCarrier, c.defaultService)
			if err == nil {
				break
			}
			c.logger.Error().Err(err).Str("order_id", event.OrderID).Str("package_id", event.PackageID).Msg("process PackingCompleted event; will retry")
			select {
			case <-ctx.Done():
				return
			case <-time.After(c.retryInterval):
			}
		}
		if err := c.reader.CommitMessages(ctx, message); err != nil {
			c.logger.Error().Err(err).Msg("commit PackingCompleted event")
			return
		}
	}
}

func (c *PackingConsumer) handle(payload []byte) error {
	_, err := decodePackingCompleted(payload)
	return err
}

func decodePackingCompleted(payload []byte) (application.PackingCompleted, error) {
	var envelope packingEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return application.PackingCompleted{}, fmt.Errorf("decode CloudEvent: %w", err)
	}
	if envelope.SpecVersion != "1.0" || envelope.Type != packingCompletedType {
		return application.PackingCompleted{}, fmt.Errorf("unsupported CloudEvent spec/type: %q/%q", envelope.SpecVersion, envelope.Type)
	}
	event := application.PackingCompleted{
		EventID: envelope.ID, OrderID: envelope.Data.OrderID,
		PackageID: envelope.Data.PackageID, SealCode: envelope.Data.SealCode,
		Parcel: envelope.Data.Parcel,
	}
	if event.EventID == "" || event.OrderID == "" || event.PackageID == "" || event.SealCode == "" {
		return application.PackingCompleted{}, fmt.Errorf("PackingCompleted identity fields are required")
	}
	if err := event.Parcel.Validate(); err != nil {
		return application.PackingCompleted{}, err
	}
	return event, nil
}
