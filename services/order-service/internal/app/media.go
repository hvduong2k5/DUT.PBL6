package app

import (
	"context"
	"strings"

	"github.com/omamx/order-service/internal/mediaprovider"
)

// ValidateAndBindEvidence verifies that each media file_id is READY for expectedPurpose
// and binds it to the target business entity (e.g. OrderRefund, B2BOrder).
func (s *Service) ValidateAndBindEvidence(ctx context.Context, fileIDs []string, expectedPurpose, entityType, entityID string) error {
	if len(fileIDs) == 0 {
		return nil
	}
	if s.Media == nil {
		return mediaprovider.ErrUnavailable
	}
	validated := make([]mediaprovider.ValidateMediaResponse, 0, len(fileIDs))
	for _, id := range fileIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		res, err := s.Media.ValidateMedia(ctx, id, expectedPurpose)
		if err != nil {
			return err
		}
		validated = append(validated, res)
	}
	for i, id := range fileIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if err := s.Media.BindMedia(ctx, id, validated[i].ObjectKey, entityType, entityID); err != nil {
			return err
		}
	}
	return nil
}

// GetEvidenceDownloadURL requests a short-lived presigned download URL (15m default) from Media Service.
func (s *Service) GetEvidenceDownloadURL(ctx context.Context, fileID, userID, role string, expirySeconds int32) (mediaprovider.GetDownloadUrlResponse, error) {
	if s.Media == nil {
		return mediaprovider.GetDownloadUrlResponse{}, mediaprovider.ErrUnavailable
	}
	return s.Media.PresignedDownloadURL(ctx, fileID, userID, role, expirySeconds)
}
