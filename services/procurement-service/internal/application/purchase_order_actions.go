package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/domain"
)

// ExecutePurchaseOrderAction returns the committed snapshot for an identical
// command, not the current PO. A retry never targets a newer lifecycle/revision.
func (s *Service) ExecutePurchaseOrderAction(ctx context.Context, id uuid.UUID, actor, key, action string, expectedVersion int64, reason string) (domain.PurchaseOrder, error) {
	actor, key, reason = strings.TrimSpace(actor), strings.TrimSpace(key), strings.TrimSpace(reason)
	history := map[string]string{"SUBMIT": "SUBMITTED", "APPROVE": "APPROVED", "REJECT": "REJECTED", "SEND": "SENT",
		"ACKNOWLEDGE": "SUPPLIER_ACKNOWLEDGED", "REQUEST_CHANGE": "SUPPLIER_CHANGE_REQUESTED", "DECLINE": "SUPPLIER_DECLINED", "CANCEL": "CANCELLED", "CLOSE": "CLOSED"}
	requiresReason := action == "REJECT" || action == "CANCEL" || action == "CLOSE" || action == "REQUEST_CHANGE" || action == "DECLINE"
	allowsReason := requiresReason || action == "ACKNOWLEDGE"
	if id == uuid.Nil || actor == "" || !validIdempotencyKey(key) || expectedVersion < 1 || history[action] == "" ||
		(requiresReason && reason == "") || (!allowsReason && reason != "") {
		return domain.PurchaseOrder{}, domain.ErrInvalidArgument
	}
	payload, err := json.Marshal(struct {
		ID              uuid.UUID
		Action          string
		ExpectedVersion int64
		Reason          string
	}{id, action, expectedVersion, reason})
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	digest := sha256.Sum256(payload)
	hash := hex.EncodeToString(digest[:])
	prior, err := s.repo.GetPurchaseOrderActionResult(ctx, actor, key)
	if err == nil {
		if prior.RequestHash != hash {
			return domain.PurchaseOrder{}, domain.ErrConflict
		}
		return prior.Snapshot, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.PurchaseOrder{}, err
	}
	current, err := s.repo.GetPurchaseOrder(ctx, id)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	if current.Version != expectedVersion {
		return s.replayPurchaseOrderAction(ctx, actor, key, hash)
	}
	previous := current.Status
	switch action {
	case "SUBMIT":
		err = current.Submit(actor)
	case "APPROVE":
		err = current.Approve(actor)
	case "REJECT":
		err = current.Reject(actor, reason)
	case "ACKNOWLEDGE", "REQUEST_CHANGE", "DECLINE":
		response := map[string]domain.SupplierPOResponse{"ACKNOWLEDGE": domain.SupplierPOAcknowledged, "REQUEST_CHANGE": domain.SupplierPOChangeRequested, "DECLINE": domain.SupplierPODeclined}[action]
		err = current.RecordSupplierResponse(response, actor, reason)
	case "CANCEL":
		err = current.Cancel(actor, reason)
	case "CLOSE":
		var open bool
		open, err = s.repo.HasUnresolvedGoodsReceiptVariances(ctx, id)
		if err == nil && open {
			err = domain.ErrInvalidState
		}
		if err == nil {
			err = current.CloseRemaining(actor, reason)
		}
	case "SEND":
		// The legacy internal helper accepts already-sent POs. New commands must
		// have durable evidence to replay, never fabricate another SENT action.
		if current.Status != domain.PurchaseOrderApproved {
			return domain.PurchaseOrder{}, domain.ErrInvalidState
		}
		err = current.MarkSent(actor)
	}
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	command := domain.PurchaseOrderActionCommand{Kind: action, HistoryAction: history[action], Actor: actor, Key: key, RequestHash: hash, ExpectedVersion: expectedVersion, Reason: reason}
	result, err := s.repo.SavePurchaseOrderAction(ctx, current, previous, command)
	if !errors.Is(err, domain.ErrConflict) {
		return result, err
	}
	// An identical transaction may have committed after our initial lookup.
	// The unique actor/key constraint also rolls back cross-resource key races.
	return s.replayPurchaseOrderAction(ctx, actor, key, hash)
}

func (s *Service) replayPurchaseOrderAction(ctx context.Context, actor, key, hash string) (domain.PurchaseOrder, error) {
	prior, lookupErr := s.repo.GetPurchaseOrderActionResult(ctx, actor, key)
	if lookupErr != nil {
		if errors.Is(lookupErr, domain.ErrNotFound) {
			return domain.PurchaseOrder{}, domain.ErrConflict
		}
		return domain.PurchaseOrder{}, lookupErr
	}
	if prior.RequestHash != hash {
		return domain.PurchaseOrder{}, domain.ErrConflict
	}
	return prior.Snapshot, nil
}
