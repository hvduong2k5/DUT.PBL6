package dependency

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	mediav1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/media/v1"
	"google.golang.org/grpc"
)

func (s Live) ValidateMedia(ctx context.Context, content domain.Content) error {
	ids, err := domain.CertificateMediaIDs(content)
	if err != nil {
		return err
	}
	if s.MediaClient == nil {
		return domain.ErrUnavailable
	}
	for _, id := range ids {
		result, err := s.MediaClient.ValidateMediaReference(ctx, &mediav1.ValidateMediaRequest{FileId: id, ExpectedPurpose: "OCOP_CERTIFICATE"}, grpc.MaxCallRecvMsgSize(8192))
		if err != nil || result == nil {
			return domain.ErrUnavailable
		}
		if !result.GetIsValid() || result.GetStatus() != "READY" {
			return &domain.ValidationError{Field: "certificate.media_reference"}
		}
		digest, err := hex.DecodeString(result.GetSha256Checksum())
		if err != nil || len(digest) != 32 || result.GetSizeBytes() <= 0 || result.GetObjectKey() == "" {
			return domain.ErrUnavailable
		}
		switch strings.ToLower(result.GetMimeType()) {
		case "application/pdf", "image/jpeg", "image/png", "image/webp":
		default:
			return &domain.ValidationError{Field: "certificate.media_reference"}
		}
	}
	return nil
}

func (s Live) BindMedia(ctx context.Context, profileID, fileID string) error {
	if s.MediaClient == nil {
		return domain.ErrUnavailable
	}
	response, err := s.MediaClient.BindMediaToEntity(ctx, &mediav1.BindMediaRequest{FileId: fileID, EntityType: "TraceabilityProfile", EntityId: profileID}, grpc.MaxCallRecvMsgSize(8192))
	if err != nil || response == nil || !response.GetSuccess() {
		return domain.ErrUnavailable
	}
	return nil
}
