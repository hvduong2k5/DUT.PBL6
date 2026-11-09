package worker

import (
	"context"
	"encoding/json"
	d "github.com/omamx/fulfillment-service/internal/domain"
	f "github.com/omamx/fulfillment-service/internal/gen/fulfillment/v1"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"time"
)

type PermitAcquirer interface {
	AcquirePermit(context.Context, d.Ready, string) (*f.AcceptTaskRequest, error)
}

// Serial processing per partition/group: offsets advance only after a durable
// inbox disposition. No message is acknowledged solely from JSON validation.
func (r *Runtime) ConsumeReady(ctx context.Context, reader *kafka.Reader) {
	defer reader.Close()
	for {
		message, e := reader.FetchMessage(ctx)
		if e != nil {
			return
		}
		for {
			e = r.Store.IngestReady(ctx, message.Value, message.Key, message.Topic, message.Partition, message.Offset)
			if e == nil {
				break
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
		if e = reader.CommitMessages(ctx, message); e != nil {
			return
		}
	}
}
func (r *Runtime) ReadyStep(ctx context.Context) {
	if r.Service == nil {
		return
	}
	j, found, e := r.Store.ClaimJob(ctx, r.Owner, "READY_ADMISSION")
	if e != nil || !found {
		return
	}
	var request f.AcceptTaskRequest
	if len(j.Original) == 0 {
		var ready d.Ready
		e = json.Unmarshal(j.Input, &ready)
		if e == nil {
			provider, ok := r.Service.Authority.(PermitAcquirer)
			if !ok {
				e = d.Fail("CAPABILITY_UNAVAILABLE", 503)
			} else {
				var req *f.AcceptTaskRequest
				req, e = provider.AcquirePermit(ctx, ready, j.CommandKey)
				if e == nil {
					raw, err := protojson.Marshal(req)
					e = err
					if e == nil {
						e = r.Store.SaveJobRequest(ctx, r.Owner, j, raw)
						if e == nil {
							proto.Merge(&request, req)
						}
					}
				}
			}
		}
	} else {
		e = protojson.Unmarshal(j.Original, &request)
	}
	if e == nil {
		var result d.Decision
		result, e = r.Service.Accept(ctx, d.Principal{Service: "fulfillment-service"}, &request)
		if e == nil && result.Command != nil && result.Command.Result == "REJECTED" {
			e = d.Fail("ADMISSION_REQUIRES_REVIEW", 409)
		}
	}
	code := ""
	if e != nil {
		code = "ADMISSION_RECONCILIATION_PENDING"
	}
	_ = r.Store.FinishJob(ctx, r.Owner, j, e == nil, code)
}
