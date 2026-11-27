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

// RevisePurchaseOrderWithIdempotency checks committed evidence before loading
// supplier offers. A retry must not reprice an already committed draft.
func (s *Service) RevisePurchaseOrderWithIdempotency(ctx context.Context, id uuid.UUID, request PurchaseOrderRequest, actor, key string, expectedVersion int64, reason string) (domain.PurchaseOrder, error) {
	return s.executePurchaseOrderTerms(ctx, id, request, actor, key, expectedVersion, reason, "REVISION")
}

func (s *Service) UpdatePurchaseOrderDraftWithIdempotency(ctx context.Context, id uuid.UUID, request PurchaseOrderRequest, actor, key string, expectedVersion int64) (domain.PurchaseOrder, error) {
	return s.executePurchaseOrderTerms(ctx, id, request, actor, key, expectedVersion, "", "EDIT_DRAFT")
}

func (s *Service) executePurchaseOrderTerms(ctx context.Context, id uuid.UUID, request PurchaseOrderRequest, actor, key string, expectedVersion int64, reason, action string) (domain.PurchaseOrder, error) {
	actor, key, reason = strings.TrimSpace(actor), strings.TrimSpace(key), strings.TrimSpace(reason)
	if id == uuid.Nil || actor == "" || !validIdempotencyKey(key) || expectedVersion < 1 || (action == "REVISION" && reason == "") || request.SupplierID == uuid.Nil || len(request.Lines) == 0 {
		return domain.PurchaseOrder{}, domain.ErrInvalidArgument
	}
	request.Notes = strings.TrimSpace(request.Notes)
	request.Lines = append([]PurchaseOrderLineRequest(nil), request.Lines...)
	for i := range request.Lines {
		request.Lines[i].Quantity = strings.TrimSpace(request.Lines[i].Quantity)
	}
	if request.ExpectedDeliveryDate != nil {
		date := request.ExpectedDeliveryDate.UTC()
		request.ExpectedDeliveryDate = &date
	}
	payload, err := json.Marshal(struct {
		ID              uuid.UUID
		Action          string
		ExpectedVersion int64
		Reason          string
		Request         PurchaseOrderRequest
	}{id, action, expectedVersion, reason, request})
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
	expectedStatus := domain.PurchaseOrderChangeRequested
	if action == "EDIT_DRAFT" {
		expectedStatus = domain.PurchaseOrderDraft
	}
	if current.Status != expectedStatus {
		return domain.PurchaseOrder{}, domain.ErrInvalidState
	}
	if action == "REVISION" && request.SupplierID != current.SupplierID {
		return domain.PurchaseOrder{}, domain.ErrInvalidArgument
	}
	creator := actor
	if action == "EDIT_DRAFT" {
		creator = current.CreatedBy
	}
	updated, err := s.buildPurchaseOrder(ctx, current.Number, request, creator)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	command := domain.PurchaseOrderActionCommand{Kind: action, HistoryAction: "REVISION_CREATED", Actor: actor, Key: key, RequestHash: hash, ExpectedVersion: expectedVersion, Reason: reason}
	var result domain.PurchaseOrder
	if action == "EDIT_DRAFT" {
		if err := current.ReplaceDraft(updated); err != nil {
			return domain.PurchaseOrder{}, err
		}
		command.HistoryAction = "DRAFT_UPDATED"
		result, err = s.repo.SavePurchaseOrderAction(ctx, current, expectedStatus, command)
	} else {
		if err := current.ReviseAsDraft(updated, actor, reason); err != nil {
			return domain.PurchaseOrder{}, err
		}
		result, err = s.repo.RevisePurchaseOrderAction(ctx, current, command)
	}
	if errors.Is(err, domain.ErrConflict) {
		return s.replayPurchaseOrderAction(ctx, actor, key, hash)
	}
	return result, err
}
