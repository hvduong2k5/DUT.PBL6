package media

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/domain"
	"github.com/stretchr/testify/require"
)

type fakeMediaRepository struct {
	file domain.MediaFile
}

func (f *fakeMediaRepository) CreatePending(_ context.Context, record domain.CreateMediaRecord) (domain.MediaFile, bool, error) {
	if f.file.ID != uuid.Nil {
		return f.file, false, nil
	}
	f.file = record.File
	return f.file, true, nil
}

func (f *fakeMediaRepository) GetOwned(_ context.Context, id, ownerID uuid.UUID) (domain.MediaFile, error) {
	if f.file.ID != id || f.file.OwnerID != ownerID {
		return domain.MediaFile{}, domain.ErrNotFound
	}
	return f.file, nil
}

func (f *fakeMediaRepository) MarkReady(_ context.Context, _, _ uuid.UUID, object domain.StoredObject) (domain.MediaFile, error) {
	f.file.Status = domain.MediaStatusReady
	f.file.ScanStatus = domain.ScanStatusClean
	f.file.ActualSize = object.Size
	f.file.DetectedMIMEType = object.ContentType
	return f.file, nil
}

func (f *fakeMediaRepository) DeleteUnattached(context.Context, uuid.UUID, uuid.UUID) (domain.MediaFile, error) {
	return f.file, nil
}

type fakeStorage struct {
	object domain.StoredObject
}

func (f *fakeStorage) EnsureBucket(context.Context) error { return nil }
func (f *fakeStorage) Health(context.Context) error       { return nil }
func (f *fakeStorage) PresignUpload(context.Context, string, string, string, time.Duration) (string, domain.UploadHeaders, error) {
	return "https://upload.invalid", domain.UploadHeaders{"Content-Type": "image/jpeg"}, nil
}
func (f *fakeStorage) Stat(context.Context, string) (domain.StoredObject, error) {
	return f.object, nil
}
func (f *fakeStorage) PresignDownload(context.Context, string, time.Duration) (string, error) {
	return "https://download.invalid", nil
}
func (f *fakeStorage) Remove(context.Context, string) error { return nil }

type fakeScanner struct{}

func (fakeScanner) Scan(context.Context, string) error { return nil }

func TestCreateAndCompleteUpload(t *testing.T) {
	now := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)
	repository := &fakeMediaRepository{}
	storage := &fakeStorage{object: domain.StoredObject{Size: 1024, ContentType: "image/jpeg", ETag: "etag"}}
	service := NewService(repository, storage, fakeScanner{}, Config{
		Bucket: "care-private", UploadTTL: 15 * time.Minute,
		AccessTTL: 10 * time.Minute, OrphanTTL: 24 * time.Hour,
	})
	service.now = func() time.Time { return now }
	ownerID := uuid.New()

	session, err := service.CreateUploadSession(context.Background(), CreateUploadInput{
		OwnerID: ownerID, Purpose: domain.MediaPurposeTicketAttachment,
		Filename: "C:\\fakepath\\proof.jpg", MIMEType: "image/jpeg", SizeBytes: 1024,
		IdempotencyKey: "media-create-1",
	})
	require.NoError(t, err)
	require.Equal(t, "proof.jpg", session.File.OriginalFilename)
	require.Equal(t, "PUT", session.Method)

	file, err := service.Complete(context.Background(), ownerID, session.File.ID)
	require.NoError(t, err)
	require.Equal(t, domain.MediaStatusReady, file.Status)
	require.Equal(t, domain.ScanStatusClean, file.ScanStatus)
}

func TestCompleteRejectsMismatchedObject(t *testing.T) {
	now := time.Now().UTC()
	ownerID := uuid.New()
	fileID := uuid.New()
	repository := &fakeMediaRepository{file: domain.MediaFile{
		ID: fileID, OwnerID: ownerID, Purpose: domain.MediaPurposeTicketAttachment,
		DeclaredMIMEType: "image/jpeg", DeclaredSize: 1024,
		Status: domain.MediaStatusPendingUpload, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}}
	storage := &fakeStorage{object: domain.StoredObject{Size: 2048, ContentType: "image/jpeg"}}
	service := NewService(repository, storage, fakeScanner{}, Config{UploadTTL: time.Hour, AccessTTL: time.Minute})
	service.now = func() time.Time { return now }

	_, err := service.Complete(context.Background(), ownerID, fileID)
	require.ErrorIs(t, err, domain.ErrMediaPolicy)
}

func TestUploadSessionCannotBeRenewedPastOriginalTTL(t *testing.T) {
	now := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	repository := &fakeMediaRepository{}
	service := NewService(repository, &fakeStorage{}, fakeScanner{}, Config{
		Bucket: "care-private", UploadTTL: 15 * time.Minute,
		AccessTTL: 10 * time.Minute, OrphanTTL: 24 * time.Hour,
	})
	service.now = func() time.Time { return now }
	input := CreateUploadInput{
		OwnerID: uuid.New(), Purpose: domain.MediaPurposeTicketAttachment,
		Filename: "proof.jpg", MIMEType: "image/jpeg", SizeBytes: 1024,
		IdempotencyKey: "media-expiry-1",
	}
	first, err := service.CreateUploadSession(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, now.Add(15*time.Minute), first.URLExpiresAt)

	service.now = func() time.Time { return now.Add(15 * time.Minute) }
	_, err = service.CreateUploadSession(context.Background(), input)
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestCompleteRejectsExpiredUploadSession(t *testing.T) {
	now := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	ownerID := uuid.New()
	fileID := uuid.New()
	repository := &fakeMediaRepository{file: domain.MediaFile{
		ID: fileID, OwnerID: ownerID, Purpose: domain.MediaPurposeTicketAttachment,
		DeclaredMIMEType: "image/jpeg", DeclaredSize: 1024,
		Status: domain.MediaStatusPendingUpload, CreatedAt: now.Add(-15 * time.Minute),
		ExpiresAt: now.Add(23 * time.Hour),
	}}
	service := NewService(repository, &fakeStorage{}, fakeScanner{}, Config{UploadTTL: 15 * time.Minute})
	service.now = func() time.Time { return now }

	_, err := service.Complete(context.Background(), ownerID, fileID)
	require.ErrorIs(t, err, domain.ErrMediaNotReady)
}

func TestAccessRejectsExpiredUnattachedMediaButAllowsAttachedMedia(t *testing.T) {
	now := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	ownerID := uuid.New()
	fileID := uuid.New()
	repository := &fakeMediaRepository{file: domain.MediaFile{
		ID: fileID, OwnerID: ownerID, Status: domain.MediaStatusReady,
		ExpiresAt: now.Add(-time.Second),
	}}
	service := NewService(repository, &fakeStorage{}, fakeScanner{}, Config{AccessTTL: 10 * time.Minute})
	service.now = func() time.Time { return now }

	_, _, err := service.Access(context.Background(), ownerID, fileID)
	require.ErrorIs(t, err, domain.ErrMediaNotReady)

	repository.file.Status = domain.MediaStatusAttached
	_, _, err = service.Access(context.Background(), ownerID, fileID)
	require.NoError(t, err)
}
