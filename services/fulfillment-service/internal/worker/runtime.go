package worker

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/omamx/fulfillment-service/internal/application"
	"github.com/omamx/fulfillment-service/internal/persistence"
	"github.com/segmentio/kafka-go"
	"time"
)

type Runtime struct {
	Store   *persistence.Store
	Service *application.Service
	Media   *application.Media
	Kafka   *kafka.Writer
	Owner   string
}

func (r *Runtime) Run(ctx context.Context) {
	if r.Owner == "" {
		r.Owner = uuid.NewString()
	}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			c, cancel := context.WithTimeout(ctx, 45*time.Second)
			r.Step(c)
			cancel()
		}
	}
}
func (r *Runtime) Step(ctx context.Context) {
	r.ReadyStep(ctx)
	_ = r.Store.SLA(ctx)
	if r.Media != nil && r.Media.Storage != nil {
		_ = r.Media.Cleanup(ctx)
	}
	if r.Kafka != nil {
		v, found, e := r.Store.ClaimDelivery(ctx, r.Owner)
		if e == nil && found {
			c, cancel := context.WithTimeout(ctx, 5*time.Second)
			e = r.Kafka.WriteMessages(c, kafka.Message{Topic: v.Topic, Key: []byte(v.Key), Value: v.Payload})
			cancel()
			_ = r.Store.DeliveryResult(ctx, r.Owner, v, e == nil)
		}
	}
	if r.Media != nil && r.Media.Storage != nil && r.Media.Scanner != nil {
		j, found, e := r.Store.ClaimJob(ctx, r.Owner, "MEDIA_VERIFY")
		if e == nil && found {
			var in struct {
				File string `json:"file_id"`
			}
			e = json.Unmarshal(j.Input, &in)
			if e == nil {
				c, cancel := context.WithTimeout(ctx, 45*time.Second)
				e = r.Media.Verify(c, in.File)
				cancel()
			}
			code := ""
			if e != nil {
				code = "MEDIA_VERIFICATION_RETRY"
			}
			_ = r.Store.FinishJob(ctx, r.Owner, j, e == nil, code)
		}
	}
}
