package app

import (
	"context"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/domain"
)

func (s *Service) ReviewCancellation(ctx context.Context, p domain.Principal, id string, approve bool, reason string) (map[string]string, error) {
	if e := s.manualAccess(p); e != nil {
		return nil, e
	}
	if _, e := uuid.Parse(id); e != nil {
		return nil, domain.E("NOT_FOUND", 404)
	}
	if !bounded(reason, 500) {
		return nil, domain.E("REASON_REQUIRED", 422)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	var oid, status string
	var oldHash *string
	if e = tx.QueryRow(ctx, "SELECT order_id FROM cancellation_requests WHERE id=$1", id).Scan(&oid); e != nil {
		return nil, e
	}
	o, e := s.lockOrder(ctx, tx, oid)
	if e != nil {
		return nil, e
	}
	if !CanRead(p, o) {
		return nil, domain.E("NOT_FOUND", 404)
	}
	if e = tx.QueryRow(ctx, "SELECT status,decision_hash FROM cancellation_requests WHERE id=$1 FOR UPDATE", id).Scan(&status, &oldHash); e != nil {
		return nil, e
	}
	hash := domain.Hash(map[string]any{"approve": approve, "reason": reason})
	if oldHash != nil {
		if *oldHash != hash {
			return nil, domain.E("DECISION_CONFLICT", 409)
		}
		return map[string]string{"request_id": id, "status": status}, nil
	}
	if status != "PENDING_REVIEW" {
		return nil, domain.E("INVALID_CANCELLATION_STATE", 409)
	}
	status = "REJECTED"
	if approve {
		if o.Status != "PAID" || o.Stock != "COMMITTED" || o.Task != nil {
			return nil, domain.E("CANCELLATION_NOT_ALLOWED", 409)
		}
		var already bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM cancellation_requests WHERE order_id=$1 AND status='APPROVED')", oid).Scan(&already); e != nil {
			return nil, e
		}
		if already {
			return nil, domain.E("CANCELLATION_ALREADY_APPROVED", 409)
		}
		status = "APPROVED"
		o.Hold = true
		o.Ready = false
		if e = s.touch(ctx, tx, &o, "MVP_CANCELLATION_APPROVED"); e != nil {
			return nil, e
		}
		if e = s.insertJob(ctx, tx, oid, "CANCEL", "cancel-approved:"+id, map[string]any{"target": "CANCELLED_BY_ADMIN", "approval_id": id, "actor": p.ID}); e != nil {
			return nil, e
		}
	}
	_, e = tx.Exec(ctx, "UPDATE cancellation_requests SET status=$2,decided_by=$3,decision_hash=$4,decision_reason=$5,decided_at=clock_timestamp() WHERE id=$1", id, status, p.ID, hash, reason)
	if e != nil {
		return nil, e
	}
	if e = s.Audit(ctx, tx, oid, p.ID, "CANCELLATION_REVIEW", map[string]any{"request_id": id, "status": status, "decision_hash": hash}); e != nil {
		return nil, e
	}
	return map[string]string{"request_id": id, "status": status}, tx.Commit(ctx)
}

// Internal evidence for the sandbox Inventory adapter. Real 3.2 Inventory must
// validate the authenticated proof protocol before a production rollout.
func (s *Service) CancellationApproval(ctx context.Context, id string) (domain.Approval, error) {
	if !s.MVP() {
		return domain.Approval{}, domain.E("NOT_FOUND", 404)
	}
	if _, e := uuid.Parse(id); e != nil {
		return domain.Approval{}, domain.E("NOT_FOUND", 404)
	}
	var a domain.Approval
	e := s.DB.QueryRow(ctx, "SELECT id,order_id,status,decided_by FROM cancellation_requests WHERE id=$1 AND status='APPROVED'", id).Scan(&a.ID, &a.OrderID, &a.State, &a.Actor)
	a.Type = "CANCEL"
	return a, e
}
