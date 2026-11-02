package app

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/domain/core"
	"github.com/omamx/order-service/internal/persistence"
	"net/url"
	"strconv"
)

func (s *Service) setCoreHold(ctx context.Context, p domain.Principal, id string, version int64, active bool, key string) error {
	if !p.Manager() {
		return domain.E("PERMISSION_DENIED", 403)
	}
	o, e := s.getCoreOrder(ctx, id, p, false)
	if e != nil {
		return e
	}
	var prior string
	if e = s.DB.QueryRow(ctx, "SELECT request_hash FROM command_registry WHERE scope=$1 AND kind=$2 AND key=$3", p.Scope(), "CORE_HOLD:"+id, key).Scan(&prior); e == nil {
		if prior != domain.Hash([]any{id, version, active}) {
			return domain.E("IDEMPOTENCY_CONFLICT", 409)
		}
		return nil
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	if o.Status != "PENDING_PAYMENT" && o.Status != "PAID" && o.Status != "CONFIRMED_COD" {
		return domain.E("REMOTE_STOP_REQUIRED", 409)
	}
	a, e := s.Payment.Store.LoadOrder(ctx, id, o.Scope)
	if e != nil {
		return e
	}
	var stock core.ResourceProof
	if !active {
		stock, e = s.CoreCheckout.Inventory.Lookup(ctx, a.Snapshot().Data(), "inventory:reserve:"+id)
		if e != nil {
			return e
		}
	}
	purpose := "HOLD"
	if !active {
		purpose = "RESUME"
	}
	var barrier persistence.AdmissionBarrier
	if e = s.Call(ctx, "POST", "/order-core/admission-barrier", map[string]any{"order_id": id, "order_version": version, "generation": o.Generation, "purpose": purpose}, &barrier); e != nil {
		return e
	}
	return coreFailure(s.Payment.Store.AdmissionHold(ctx, id, o.Scope, p.Scope(), key, version, active, barrier, stock))
}

func (s *Service) processCoreEvent(ctx context.Context, event domain.Event) (string, error) {
	var proof persistence.SourceEvidence
	if e := s.Call(ctx, "GET", "/order-core/resources/"+url.PathEscape(event.ResourceID)+"?source="+url.QueryEscape(event.Source)+"&version="+strconv.FormatInt(event.Version, 10), nil, &proof); e != nil {
		return "", e
	}
	status, e := s.CoreCheckout.Store.ApplySource(ctx, event, proof)
	return status, coreFailure(e)
}
func (c coreSource) Completion(ctx context.Context, id string, version int64) (out persistence.CompletionDecision, e error) {
	e = c.s.Call(ctx, "POST", "/order-core/completion", map[string]any{"order_id": id, "order_version": version}, &out)
	return
}
func (s *Service) CompleteCoreOrders(ctx context.Context) error {
	if s.CoreCheckout == nil || !s.C.AutoComplete {
		return nil
	}
	return s.CoreCheckout.Store.CompleteDue(ctx, coreSource{s})
}
