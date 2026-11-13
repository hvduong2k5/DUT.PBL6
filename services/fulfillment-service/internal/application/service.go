package application

import (
	"context"
	"encoding/json"
	d "github.com/omamx/fulfillment-service/internal/domain"
	f "github.com/omamx/fulfillment-service/internal/gen/fulfillment/v1"
	"github.com/omamx/fulfillment-service/internal/persistence"
	"time"
)

// Authority is deliberately fail-closed. Implementations must resolve owning
// peers, never treat caller-provided inputs/booleans as authoritative proofs.
type Authority interface {
	Admission(context.Context, *f.AcceptTaskRequest) (d.Permit, d.Inputs, d.Stock, error)
	ValidatePacking(context.Context, d.Task) (*d.Validation, error)
}
type Service struct {
	Store                                     *persistence.Store
	Authority                                 Authority
	AdmissionEnabled, SealEnabled, Integrated bool
}

func (s *Service) Accept(ctx context.Context, p d.Principal, in *f.AcceptTaskRequest) (d.Decision, error) {
	var out d.Decision
	if p.Service != "fulfillment-service" {
		return out, d.Fail("FORBIDDEN", 403)
	}
	if in == nil || in.ContractVersion != d.Contract {
		return out, d.Fail("UNSUPPORTED_CONTRACT", 400)
	}
	hash, e := ProtoHash(in)
	if e != nil {
		return out, e
	}
	if hash != in.RequestHash {
		return out, d.Fail("HASH_MISMATCH", 409)
	}
	c := d.Command{Caller: p.Caller(), Action: "AcceptTask", OrderID: in.OrderId, Generation: in.Generation, Key: in.IdempotencyKey, Hash: hash}
	if e = c.Validate(); e != nil {
		return out, e
	}
	if b, found, e := s.Store.Replay(ctx, c); e != nil {
		return out, e
	} else if found {
		_ = b
		return s.Store.Decision(ctx, c.OrderID, c.Generation, c.Key, c.Action)
	}
	if !s.AdmissionEnabled || s.Authority == nil {
		return out, d.Fail("CAPABILITY_UNAVAILABLE", 503)
	}
	permit, inputs, stock, e := s.Authority.Admission(ctx, in)
	if e != nil {
		return out, e
	}
	return s.Store.Accept(ctx, d.AdmissionInput{Command: c, Permit: permit, Inputs: inputs, Stock: stock, Integrated: s.Integrated})
}
func (s *Service) Cancel(ctx context.Context, p d.Principal, in *f.CancelTaskRequest) (d.Decision, error) {
	if p.Service != "order-service" {
		return d.Decision{}, d.Fail("FORBIDDEN", 403)
	}
	if in == nil || in.ContractVersion != d.Contract {
		return d.Decision{}, d.Fail("UNSUPPORTED_CONTRACT", 400)
	}
	hash, e := ProtoHash(in)
	if e != nil {
		return d.Decision{}, e
	}
	if hash != in.RequestHash {
		return d.Decision{}, d.Fail("HASH_MISMATCH", 409)
	}
	return s.Store.Cancel(ctx, d.Command{Caller: p.Caller(), Action: "CancelTask", OrderID: in.OrderId, Generation: in.Generation, Key: in.IdempotencyKey, Hash: hash}, in.Reason)
}
func (s *Service) Mutate(ctx context.Context, p d.Principal, id, key, hash string, m d.Mutation) (d.Task, error) {
	t, e := s.Store.Task(ctx, id)
	if e != nil {
		return t, e
	}
	permission := t.Permission(m.Action)
	if m.Action == "AssignTask" && m.AssigneeID != "" && m.AssigneeID != p.ID {
		permission = "FULFILLMENT_TASK_ASSIGN_OTHER"
	}
	if m.Action == "UnassignTask" && (t.Assignee == nil || *t.Assignee != p.ID) {
		permission = "FULFILLMENT_TASK_ASSIGN_OTHER"
	}
	if !p.Can(permission, t.Warehouse) {
		return d.Task{}, d.Fail("FORBIDDEN", 403)
	}
	c := d.Command{Caller: p.Caller(), Action: m.Action, OrderID: t.OrderID, Generation: t.Generation, Key: key, Hash: hash}
	if b, found, e := s.Store.Replay(ctx, c); e != nil {
		return d.Task{}, e
	} else if found {
		var out d.Task
		e = json.Unmarshal(b, &out)
		return out, e
	}
	var proof *d.Validation
	if m.Action == "SealTask" {
		if !s.SealEnabled || s.Authority == nil {
			return d.Task{}, d.Fail("CAPABILITY_UNAVAILABLE", 503)
		}
		proof, e = s.Authority.ValidatePacking(ctx, t)
		if e != nil {
			return d.Task{}, e
		}
	}
	return s.Store.Mutate(ctx, p, id, key, hash, m, proof)
}

type Unavailable struct{}

func (Unavailable) Admission(context.Context, *f.AcceptTaskRequest) (d.Permit, d.Inputs, d.Stock, error) {
	return d.Permit{}, d.Inputs{}, d.Stock{}, d.Fail("CAPABILITY_UNAVAILABLE", 503)
}
func (Unavailable) ValidatePacking(context.Context, d.Task) (*d.Validation, error) {
	return nil, d.Fail("CAPABILITY_UNAVAILABLE", 503)
}

const PeerTimeout = 2 * time.Second
