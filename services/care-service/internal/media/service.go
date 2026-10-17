package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/domain"
	"github.com/omamx/care-service/internal/idgen"
)

type Service struct {
	repository domain.MediaRepository
	storage    domain.ObjectStorage
	scanner    domain.MalwareScanner
	bucket     string
	uploadTTL  time.Duration
	accessTTL  time.Duration
	orphanTTL  time.Duration
	now        func() time.Time
}

type Config struct {
	Bucket    string
	UploadTTL time.Duration
	AccessTTL time.Duration
	OrphanTTL time.Duration
}

func NewService(repository domain.MediaRepository, storage domain.ObjectStorage, scanner domain.MalwareScanner, cfg Config) *Service {
	return &Service{
		repository: repository,
		storage:    storage,
		scanner:    scanner,
		bucket:     cfg.Bucket,
		uploadTTL:  cfg.UploadTTL,
		accessTTL:  cfg.AccessTTL,
		orphanTTL:  cfg.OrphanTTL,
		now:        time.Now,
	}
}

type CreateUploadInput struct {
	OwnerID        uuid.UUID
	Purpose        domain.MediaPurpose
	Filename       string
	MIMEType       string
	SizeBytes      int64
	ChecksumSHA256 string
	IdempotencyKey string
}

type UploadSession struct {
	File         domain.MediaFile
	UploadURL    string
	Method       string
	Headers      domain.UploadHeaders
	URLExpiresAt time.Time
	Created      bool
}

func (s *Service) CreateUploadSession(ctx context.Context, input CreateUploadInput) (UploadSession, error) {
	if input.OwnerID == uuid.Nil {
		return UploadSession{}, domain.ErrForbidden
	}
	if err := validateIdempotencyKey(input.IdempotencyKey); err != nil {
		return UploadSession{}, err
	}
	input.Filename = sanitizeFilename(input.Filename)
	input.MIMEType = strings.ToLower(strings.TrimSpace(input.MIMEType))
	input.ChecksumSHA256 = strings.ToLower(strings.TrimSpace(input.ChecksumSHA256))
	policy, err := domain.PolicyForPurpose(input.Purpose)
	if err != nil {
		return UploadSession{}, err
	}
	if err := policy.Validate(input.Filename, input.MIMEType, input.SizeBytes); err != nil {
		return UploadSession{}, err
	}
	if input.ChecksumSHA256 != "" {
		decoded, decodeErr := hex.DecodeString(input.ChecksumSHA256)
		if decodeErr != nil || len(decoded) != sha256.Size {
			return UploadSession{}, fmt.Errorf("%w: checksum_sha256 must contain 64 hexadecimal characters", domain.ErrInvalidArgument)
		}
	}

	now := s.now().UTC()
	fileID := idgen.New()
	objectKey := fmt.Sprintf("care/%s/%s/%s", strings.ToLower(string(input.Purpose)), input.OwnerID, fileID)
	requestHash := hashRequest(struct {
		Purpose  domain.MediaPurpose `json:"purpose"`
		Filename string              `json:"filename"`
		MIMEType string              `json:"mime_type"`
		Size     int64               `json:"size"`
		Checksum string              `json:"checksum"`
	}{input.Purpose, input.Filename, input.MIMEType, input.SizeBytes, input.ChecksumSHA256})

	file, created, err := s.repository.CreatePending(ctx, domain.CreateMediaRecord{
		File: domain.MediaFile{
			ID: fileID, OwnerID: input.OwnerID, Purpose: input.Purpose,
			Provider: "S3", Bucket: s.bucket, ObjectKey: objectKey,
			OriginalFilename: input.Filename, DeclaredMIMEType: input.MIMEType,
			DeclaredSize: input.SizeBytes, ChecksumSHA256: input.ChecksumSHA256,
			Status: domain.MediaStatusPendingUpload, ScanStatus: domain.ScanStatusPending,
			Version: 1, CreatedAt: now, UpdatedAt: now, ExpiresAt: now.Add(s.orphanTTL),
		},
		IdempotencyKey: input.IdempotencyKey, RequestHash: requestHash,
		IdempotencyExpiry: now.Add(24 * time.Hour),
	})
	if err != nil {
		return UploadSession{}, err
	}
	sessionExpiresAt := file.CreatedAt.Add(s.uploadTTL)
	currentTime := s.now().UTC()
	if file.Status != domain.MediaStatusPendingUpload || !currentTime.Before(sessionExpiresAt) {
		return UploadSession{}, fmt.Errorf("%w: upload session is no longer active", domain.ErrConflict)
	}
	url, headers, err := s.storage.PresignUpload(
		ctx, file.ObjectKey, file.DeclaredMIMEType, file.ChecksumSHA256,
		sessionExpiresAt.Sub(currentTime),
	)
	if err != nil {
		return UploadSession{}, fmt.Errorf("create upload URL: %w", err)
	}
	return UploadSession{
		File: file, UploadURL: url, Method: "PUT", Headers: headers,
		URLExpiresAt: sessionExpiresAt, Created: created,
	}, nil
}

func (s *Service) Complete(ctx context.Context, ownerID, fileID uuid.UUID) (domain.MediaFile, error) {
	file, err := s.repository.GetOwned(ctx, fileID, ownerID)
	if err != nil {
		return domain.MediaFile{}, err
	}
	if file.Status == domain.MediaStatusReady || file.Status == domain.MediaStatusAttached {
		return file, nil
	}
	if file.Status != domain.MediaStatusPendingUpload && file.Status != domain.MediaStatusUploaded && file.Status != domain.MediaStatusScanning {
		return domain.MediaFile{}, domain.ErrMediaNotReady
	}
	currentTime := s.now().UTC()
	if !currentTime.Before(file.CreatedAt.Add(s.uploadTTL)) || !currentTime.Before(file.ExpiresAt) {
		return domain.MediaFile{}, domain.ErrMediaNotReady
	}
	object, err := s.storage.Stat(ctx, file.ObjectKey)
	if err != nil {
		return domain.MediaFile{}, fmt.Errorf("stat uploaded object: %w", err)
	}
	if object.Size != file.DeclaredSize || !strings.EqualFold(object.ContentType, file.DeclaredMIMEType) {
		return domain.MediaFile{}, fmt.Errorf("%w: uploaded object does not match declared size or MIME type", domain.ErrMediaPolicy)
	}
	if file.ChecksumSHA256 != "" && !strings.EqualFold(object.ChecksumSHA256, file.ChecksumSHA256) {
		return domain.MediaFile{}, fmt.Errorf("%w: uploaded object checksum does not match", domain.ErrMediaPolicy)
	}
	if err := s.scanner.Scan(ctx, file.ObjectKey); err != nil {
		return domain.MediaFile{}, fmt.Errorf("scan media: %w", err)
	}
	return s.repository.MarkReady(ctx, file.ID, ownerID, object)
}

func (s *Service) Access(ctx context.Context, ownerID, fileID uuid.UUID) (string, time.Time, error) {
	file, err := s.repository.GetOwned(ctx, fileID, ownerID)
	if err != nil {
		return "", time.Time{}, err
	}
	if file.Status != domain.MediaStatusReady && file.Status != domain.MediaStatusAttached {
		return "", time.Time{}, domain.ErrMediaNotReady
	}
	now := s.now().UTC()
	if file.Status == domain.MediaStatusReady && !now.Before(file.ExpiresAt) {
		return "", time.Time{}, domain.ErrMediaNotReady
	}
	expiresAt := now.Add(s.accessTTL)
	url, err := s.storage.PresignDownload(ctx, file.ObjectKey, s.accessTTL)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("create media access URL: %w", err)
	}
	return url, expiresAt, nil
}

func (s *Service) DeleteUnattached(ctx context.Context, ownerID, fileID uuid.UUID) error {
	file, err := s.repository.DeleteUnattached(ctx, fileID, ownerID)
	if err != nil {
		return err
	}
	if err := s.storage.Remove(ctx, file.ObjectKey); err != nil {
		return fmt.Errorf("delete media object after tombstone: %w", err)
	}
	return nil
}

func sanitizeFilename(filename string) string {
	filename = strings.ReplaceAll(strings.TrimSpace(filename), "\\", "/")
	return path.Base(filename)
}

func validateIdempotencyKey(key string) error {
	if strings.TrimSpace(key) == "" || len(key) > 200 {
		return fmt.Errorf("%w: Idempotency-Key is required and must not exceed 200 characters", domain.ErrInvalidArgument)
	}
	return nil
}

func hashRequest(value any) string {
	payload, _ := json.Marshal(value)
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}
