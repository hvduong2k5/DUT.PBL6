package mediaprovider

import (
	"context"
	"errors"
	"testing"
)

type mockMediaClient struct {
	validateRes ValidateMediaResponse
	validateErr error
	bindRes     BindMediaResponse
	bindErr     error
	dlRes       GetDownloadUrlResponse
	dlErr       error
	lastDLReq   GetDownloadUrlRequest
}

func (m *mockMediaClient) ValidateMediaReference(_ context.Context, _ ValidateMediaRequest) (ValidateMediaResponse, error) {
	return m.validateRes, m.validateErr
}

func (m *mockMediaClient) BindMediaToEntity(_ context.Context, _ BindMediaRequest) (BindMediaResponse, error) {
	return m.bindRes, m.bindErr
}

func (m *mockMediaClient) GetPresignedDownloadUrl(_ context.Context, in GetDownloadUrlRequest) (GetDownloadUrlResponse, error) {
	m.lastDLReq = in
	return m.dlRes, m.dlErr
}

func TestValidateMediaChecksReadyStatus(t *testing.T) {
	a := &Adapter{Client: &mockMediaClient{
		validateRes: ValidateMediaResponse{IsValid: true, Status: "READY", ObjectKey: "om-documents/refund/1.jpg"},
	}}
	res, err := a.ValidateMedia(context.Background(), "med_123", "REFUND_EVIDENCE")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.ObjectKey != "om-documents/refund/1.jpg" {
		t.Fatalf("unexpected object key: %s", res.ObjectKey)
	}

	aNotReady := &Adapter{Client: &mockMediaClient{
		validateRes: ValidateMediaResponse{IsValid: true, Status: "AWAITING_UPLOAD"},
	}}
	if _, err := aNotReady.ValidateMedia(context.Background(), "med_123", "REFUND_EVIDENCE"); !errors.Is(err, ErrNotReady) {
		t.Fatalf("expected ErrNotReady, got %v", err)
	}
}

func TestBindAndPresignedDownloadURL(t *testing.T) {
	mc := &mockMediaClient{
		bindRes: BindMediaResponse{Success: true},
		dlRes:   GetDownloadUrlResponse{DownloadURL: "https://minio.local/om-documents/1.jpg?sig=abc", ExpiresAt: "2026-11-27T22:00:00Z"},
	}
	a := &Adapter{Client: mc}

	if err := a.BindMedia(context.Background(), "med_1", "om-documents/1.jpg", "OrderRefund", "ref_1"); err != nil {
		t.Fatalf("expected bind success, got %v", err)
	}

	dl, err := a.PresignedDownloadURL(context.Background(), "med_1", "usr_1", "CSKH", 0)
	if err != nil {
		t.Fatalf("expected download url success, got %v", err)
	}
	if mc.lastDLReq.ExpirySeconds != 900 {
		t.Fatalf("expected default 900s expiry, got %d", mc.lastDLReq.ExpirySeconds)
	}
	if dl.DownloadURL == "" {
		t.Fatal("expected non-empty download URL")
	}
}
