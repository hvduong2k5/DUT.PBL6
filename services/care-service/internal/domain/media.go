package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type MediaPurpose string

const (
	MediaPurposeReturnEvidence   MediaPurpose = "RETURN_EVIDENCE"
	MediaPurposeTicketAttachment MediaPurpose = "TICKET_ATTACHMENT"
	MediaPurposeReviewMedia      MediaPurpose = "REVIEW_MEDIA"
)

type MediaStatus string

const (
	MediaStatusPendingUpload MediaStatus = "PENDING_UPLOAD"
	MediaStatusUploaded      MediaStatus = "UPLOADED"
	MediaStatusScanning      MediaStatus = "SCANNING"
	MediaStatusReady         MediaStatus = "READY"
	MediaStatusAttached      MediaStatus = "ATTACHED"
	MediaStatusRejected      MediaStatus = "REJECTED"
	MediaStatusExpired       MediaStatus = "EXPIRED"
	MediaStatusDeleted       MediaStatus = "DELETED"
)

type ScanStatus string

const (
	ScanStatusPending  ScanStatus = "PENDING"
	ScanStatusClean    ScanStatus = "CLEAN"
	ScanStatusInfected ScanStatus = "INFECTED"
	ScanStatusFailed   ScanStatus = "FAILED"
)

type MediaFile struct {
	ID               uuid.UUID
	OwnerID          uuid.UUID
	Purpose          MediaPurpose
	Provider         string
	Bucket           string
	ObjectKey        string
	OriginalFilename string
	DeclaredMIMEType string
	DetectedMIMEType string
	DeclaredSize     int64
	ActualSize       int64
	ChecksumSHA256   string
	StorageETag      string
	Status           MediaStatus
	ScanStatus       ScanStatus
	Version          int
	CreatedAt        time.Time
	UpdatedAt        time.Time
	UploadedAt       *time.Time
	AttachedAt       *time.Time
	ExpiresAt        time.Time
	DeletedAt        *time.Time
}

type MediaPolicy struct {
	Purpose       MediaPurpose
	MaxSizeByMIME map[string]int64
}

func PolicyForPurpose(purpose MediaPurpose) (MediaPolicy, error) {
	images := map[string]int64{
		"image/jpeg": 10 << 20,
		"image/png":  10 << 20,
		"image/webp": 10 << 20,
	}
	videoAndImages := cloneMIMELimits(images)
	videoAndImages["video/mp4"] = 100 << 20

	switch purpose {
	case MediaPurposeReturnEvidence, MediaPurposeReviewMedia:
		return MediaPolicy{Purpose: purpose, MaxSizeByMIME: videoAndImages}, nil
	case MediaPurposeTicketAttachment:
		allowed := cloneMIMELimits(videoAndImages)
		allowed["application/pdf"] = 10 << 20
		return MediaPolicy{Purpose: purpose, MaxSizeByMIME: allowed}, nil
	default:
		return MediaPolicy{}, fmt.Errorf("%w: unsupported purpose %q", ErrMediaPolicy, purpose)
	}
}

func (p MediaPolicy) Validate(filename, mimeType string, size int64) error {
	if strings.TrimSpace(filename) == "" || len(filename) > 255 {
		return fmt.Errorf("%w: invalid filename", ErrMediaPolicy)
	}
	maxSize, allowed := p.MaxSizeByMIME[strings.ToLower(strings.TrimSpace(mimeType))]
	if !allowed {
		return fmt.Errorf("%w: MIME type is not allowed", ErrMediaPolicy)
	}
	if size <= 0 || size > maxSize {
		return fmt.Errorf("%w: invalid file size", ErrMediaPolicy)
	}
	return nil
}

func cloneMIMELimits(source map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
