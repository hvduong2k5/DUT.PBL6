package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/domain"
	"github.com/rs/zerolog"
	kafkago "github.com/segmentio/kafka-go"
)

const (
	packingCompletedV1Type = "vn.omama.fulfillment.packing.completed.v1"
	packingCompletedV2Type = "vn.omama.fulfillment.packing.completed.v2"
	fulfillmentEventSource = "https://omama.vn/services/fulfillment-service"
)

var (
	checksumPattern    = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
	traceparentPattern = regexp.MustCompile(`^00-[a-f0-9]{32}-[a-f0-9]{16}-[a-f0-9]{2}$`)
)

type packingEnvelope struct {
	SpecVersion string      `json:"specversion"`
	ID          string      `json:"id"`
	Source      string      `json:"source"`
	Type        string      `json:"type"`
	Subject     string      `json:"subject"`
	Time        time.Time   `json:"time"`
	ContentType string      `json:"datacontenttype"`
	TraceParent string      `json:"traceparent"`
	Data        packingData `json:"data"`
}

type packingData struct {
	OrderID             string        `json:"order_id"`
	PackageID           string        `json:"package_id"`
	PackerStaffID       string        `json:"packer_staff_id"`
	SealCode            string        `json:"seal_code"`
	Parcel              domain.Parcel `json:"parcel"`
	VideoObjectKey      string        `json:"s3_video_object_key"`
	VideoChecksumSHA256 string        `json:"video_checksum_sha256"`
	ScannedBatchCodes   []string      `json:"scanned_batch_codes"`
	PackedAt            time.Time     `json:"packed_at"`
	TaskID              string        `json:"task_id"`
	ResourceVersion     int64         `json:"resource_version"`
	ReadyGeneration     int64         `json:"ready_generation"`
}

type PackingConsumer struct {
	reader         *kafkago.Reader
	service        *application.Service
	snapshots      application.ShippingSnapshotProvider
	defaultCarrier string
	defaultService string
	eventType      string
	retryInterval  time.Duration
	logger         zerolog.Logger
}

func NewPackingConsumer(brokers []string, topic, group string, service *application.Service, snapshots application.ShippingSnapshotProvider, defaultCarrier, defaultService string, logger zerolog.Logger) *PackingConsumer {
	return newPackingConsumer(brokers, topic, group, packingCompletedV1Type, service, snapshots, defaultCarrier, defaultService, logger)
}

func NewPackingV2Consumer(brokers []string, topic, group string, service *application.Service, snapshots application.ShippingSnapshotProvider, defaultCarrier, defaultService string, logger zerolog.Logger) *PackingConsumer {
	return newPackingConsumer(brokers, topic, group, packingCompletedV2Type, service, snapshots, defaultCarrier, defaultService, logger)
}

func newPackingConsumer(brokers []string, topic, group, eventType string, service *application.Service, snapshots application.ShippingSnapshotProvider, defaultCarrier, defaultService string, logger zerolog.Logger) *PackingConsumer {
	return &PackingConsumer{
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers: brokers, Topic: topic, GroupID: group,
			MinBytes: 1, MaxBytes: 10e6, CommitInterval: 0,
		}),
		service: service, snapshots: snapshots,
		defaultCarrier: defaultCarrier, defaultService: defaultService,
		eventType:     eventType,
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
		event, err := decodePackingCompletedForType(message.Value, c.eventType)
		if err != nil {
			c.logger.Error().Err(err).Int("partition", message.Partition).Int64("offset", message.Offset).Msg("invalid fulfillment event skipped")
			if commitErr := c.reader.CommitMessages(ctx, message); commitErr != nil {
				c.logger.Error().Err(commitErr).Msg("commit invalid fulfillment event")
				return
			}
			continue
		}

		for {
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

func decodePackingCompleted(payload []byte) (application.PackingCompleted, error) {
	return decodePackingCompletedForType(payload, "")
}

func decodePackingCompletedForType(payload []byte, expectedType string) (application.PackingCompleted, error) {
	var envelope packingEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return application.PackingCompleted{}, fmt.Errorf("decode CloudEvent: %w", err)
	}
	if envelope.SpecVersion != "1.0" || (envelope.Type != packingCompletedV1Type && envelope.Type != packingCompletedV2Type) ||
		(expectedType != "" && envelope.Type != expectedType) {
		return application.PackingCompleted{}, fmt.Errorf("unsupported CloudEvent spec/type: %q/%q", envelope.SpecVersion, envelope.Type)
	}
	eventVersion := 1
	if envelope.Type == packingCompletedV2Type {
		eventVersion = 2
	}
	event := application.PackingCompleted{
		EventID: envelope.ID, EventVersion: eventVersion, OrderID: envelope.Data.OrderID,
		PackageID: envelope.Data.PackageID, SealCode: envelope.Data.SealCode,
		Parcel: envelope.Data.Parcel, TaskID: envelope.Data.TaskID,
		TaskResourceVersion: envelope.Data.ResourceVersion,
		ReadyGeneration:     envelope.Data.ReadyGeneration, PackedAt: envelope.Data.PackedAt,
	}
	if event.EventID == "" || event.OrderID == "" || event.PackageID == "" || event.SealCode == "" {
		return application.PackingCompleted{}, fmt.Errorf("PackingCompleted identity fields are required")
	}
	if err := event.Parcel.Validate(); err != nil {
		return application.PackingCompleted{}, err
	}
	if eventVersion == 2 {
		if envelope.Source != fulfillmentEventSource || envelope.Subject != "order_id:"+event.OrderID ||
			envelope.Time.IsZero() || envelope.ContentType != "application/json" ||
			!traceparentPattern.MatchString(envelope.TraceParent) {
			return application.PackingCompleted{}, fmt.Errorf("PackingCompleted v2 CloudEvent evidence is invalid")
		}
		if strings.TrimSpace(envelope.Data.PackerStaffID) == "" || strings.TrimSpace(envelope.Data.VideoObjectKey) == "" ||
			!checksumPattern.MatchString(envelope.Data.VideoChecksumSHA256) || envelope.Data.ScannedBatchCodes == nil ||
			event.PackedAt.IsZero() || event.TaskResourceVersion < 1 || event.ReadyGeneration < 1 {
			return application.PackingCompleted{}, fmt.Errorf("PackingCompleted v2 proof fields are invalid")
		}
		if _, err := uuid.Parse(event.TaskID); err != nil {
			return application.PackingCompleted{}, fmt.Errorf("PackingCompleted v2 task_id must be a UUID")
		}
	}
	return event, nil
}
