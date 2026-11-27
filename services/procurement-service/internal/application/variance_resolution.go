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

func (s *Service) ResolveGoodsReceiptVarianceWithIdempotency(ctx context.Context, receiptID, varianceID uuid.UUID, decision, actor, note, key string, expectedVersion int64) (domain.GoodsReceipt, error) {
	decision, actor, note, key = strings.ToUpper(strings.TrimSpace(decision)), strings.TrimSpace(actor), strings.TrimSpace(note), strings.TrimSpace(key)
	if receiptID == uuid.Nil || varianceID == uuid.Nil || actor == "" || note == "" || !validIdempotencyKey(key) || expectedVersion < 1 || (decision != "ACCEPT_VARIANCE" && decision != "REJECT_VARIANCE") {
		return domain.GoodsReceipt{}, domain.ErrInvalidArgument
	}
	encoded, err := json.Marshal(struct {
		ReceiptID, VarianceID uuid.UUID
		Decision, Note        string
		ExpectedVersion       int64
	}{receiptID, varianceID, decision, note, expectedVersion})
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	digest := sha256.Sum256(encoded)
	hash := hex.EncodeToString(digest[:])
	prior, err := s.repo.GetGoodsReceiptResolutionResult(ctx, actor, key)
	if err == nil {
		if prior.RequestHash != hash {
			return domain.GoodsReceipt{}, domain.ErrConflict
		}
		return prior.Snapshot, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.GoodsReceipt{}, err
	}
	cmd := domain.VarianceResolutionCommand{Actor: actor, Key: key, RequestHash: hash, Decision: decision, Note: note, VarianceID: varianceID, ExpectedVersion: expectedVersion}
	receipt, err := s.repo.ResolveGoodsReceiptVarianceAction(ctx, receiptID, cmd)
	if !errors.Is(err, domain.ErrConflict) {
		return receipt, err
	}
	prior, lookupErr := s.repo.GetGoodsReceiptResolutionResult(ctx, actor, key)
	if errors.Is(lookupErr, domain.ErrNotFound) {
		return domain.GoodsReceipt{}, domain.ErrConflict
	}
	if lookupErr != nil {
		return domain.GoodsReceipt{}, lookupErr
	}
	if prior.RequestHash != hash {
		return domain.GoodsReceipt{}, domain.ErrConflict
	}
	return prior.Snapshot, nil
}
