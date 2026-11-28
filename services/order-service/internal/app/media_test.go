package app

import (
	"context"
	"errors"
	"testing"

	"github.com/omamx/order-service/internal/mediaprovider"
)

type stubMediaClient struct {
	validateErr error
	boundIDs    []string
}

func (s *stubMediaClient) ValidateMediaReference(_ context.Context, in mediaprovider.ValidateMediaRequest) (mediaprovider.ValidateMediaResponse, error) {
	if s.validateErr != nil {
		return mediaprovider.ValidateMediaResponse{}, s.validateErr
	}
	return mediaprovider.ValidateMediaResponse{
		IsValid:   true,
		Status:    "READY",
		ObjectKey: "om-documents/" + in.FileID,
	}, nil
}

func (s *stubMediaClient) BindMediaToEntity(_ context.Context, in mediaprovider.BindMediaRequest) (mediaprovider.BindMediaResponse, error) {
	s.boundIDs = append(s.boundIDs, in.FileID+":"+in.EntityType+":"+in.EntityID)
	return mediaprovider.BindMediaResponse{Success: true}, nil
}

func (s *stubMediaClient) GetPresignedDownloadUrl(_ context.Context, in mediaprovider.GetDownloadUrlRequest) (mediaprovider.GetDownloadUrlResponse, error) {
	return mediaprovider.GetDownloadUrlResponse{
		DownloadURL: "https://minio.local/om-documents/" + in.FileID + "?sig=test",
		ExpiresAt:   "2026-11-27T22:00:00Z",
	}, nil
}

func TestValidateAndBindEvidence(t *testing.T) {
	mc := &stubMediaClient{}
	svc := &Service{Media: &mediaprovider.Adapter{Client: mc}}

	if err := svc.ValidateAndBindEvidence(context.Background(), []string{"med_1", "med_2"}, "REFUND_EVIDENCE", "OrderRefund", "ref_99"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(mc.boundIDs) != 2 {
		t.Fatalf("expected 2 bound files, got %d", len(mc.boundIDs))
	}

	svcErr := &Service{Media: &mediaprovider.Adapter{Client: &stubMediaClient{validateErr: errors.New("not found")}}}
	if err := svcErr.ValidateAndBindEvidence(context.Background(), []string{"med_bad"}, "REFUND_EVIDENCE", "OrderRefund", "ref_99"); err == nil {
		t.Fatal("expected error when validation fails")
	}
}
