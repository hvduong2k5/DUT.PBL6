package mediaprovider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const defaultTimeout = 1500 * time.Millisecond

type ValidateMediaRequest struct {
	FileID          string
	ObjectKey       string
	ExpectedPurpose string
}

type ValidateMediaResponse struct {
	IsValid        bool
	Status         string // AWAITING_UPLOAD, PROCESSING, READY, PROCESSING_FAILED
	ObjectKey      string
	MimeType       string
	SizeBytes      int64
	SHA256Checksum string
}

type BindMediaRequest struct {
	FileID     string
	ObjectKey  string
	EntityType string // Product, TraceabilityProfile, FulfillmentTask, OrderRefund, B2BOrder
	EntityID   string
}

type BindMediaResponse struct {
	Success bool
}

type GetDownloadUrlRequest struct {
	FileID           string
	RequestingUserID string
	UserRole         string
	ExpirySeconds    int32
}

type GetDownloadUrlResponse struct {
	DownloadURL string
	ExpiresAt   string
}

// Client abstracts omamx.media.v1.MediaInternalService RPCs.
type Client interface {
	ValidateMediaReference(ctx context.Context, in ValidateMediaRequest) (ValidateMediaResponse, error)
	BindMediaToEntity(ctx context.Context, in BindMediaRequest) (BindMediaResponse, error)
	GetPresignedDownloadUrl(ctx context.Context, in GetDownloadUrlRequest) (GetDownloadUrlResponse, error)
}

type Adapter struct {
	Client  Client
	Token   string
	Timeout time.Duration
	Close   func() error
}

var (
	ErrUnavailable = errors.New("media service unavailable")
	ErrNotReady    = errors.New("media reference is not ready or invalid")
)

func Open(target, token, caFile, serverName string) (*Adapter, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, nil
	}
	var creds credentials.TransportCredentials
	if strings.TrimSpace(caFile) != "" {
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, errors.New("invalid media CA certificate")
		}
		cfg := &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
		if strings.TrimSpace(serverName) != "" {
			cfg.ServerName = strings.TrimSpace(serverName)
		}
		creds = credentials.NewTLS(cfg)
	} else {
		creds = insecure.NewCredentials()
	}
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}
	return &Adapter{
		Client:  &grpcClient{cc: conn},
		Token:   strings.TrimSpace(token),
		Timeout: defaultTimeout,
		Close:   conn.Close,
	}, nil
}

func (a *Adapter) withAuthAndTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := a.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if a.Token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+a.Token)
	}
	return context.WithTimeout(ctx, timeout)
}

// ValidateMedia verifies that fileID exists, matches expectedPurpose, and has status READY.
func (a *Adapter) ValidateMedia(ctx context.Context, fileID, expectedPurpose string) (ValidateMediaResponse, error) {
	if a == nil || a.Client == nil {
		return ValidateMediaResponse{}, ErrUnavailable
	}
	cctx, cancel := a.withAuthAndTimeout(ctx)
	defer cancel()
	res, err := a.Client.ValidateMediaReference(cctx, ValidateMediaRequest{
		FileID:          strings.TrimSpace(fileID),
		ExpectedPurpose: strings.TrimSpace(expectedPurpose),
	})
	if err != nil {
		return ValidateMediaResponse{}, err
	}
	if !res.IsValid || !strings.EqualFold(res.Status, "READY") {
		return res, ErrNotReady
	}
	return res, nil
}

// BindMedia binds a validated media file to a business entity (e.g. OrderRefund, B2BOrder) to prevent GC.
func (a *Adapter) BindMedia(ctx context.Context, fileID, objectKey, entityType, entityID string) error {
	if a == nil || a.Client == nil {
		return ErrUnavailable
	}
	cctx, cancel := a.withAuthAndTimeout(ctx)
	defer cancel()
	res, err := a.Client.BindMediaToEntity(cctx, BindMediaRequest{
		FileID:     strings.TrimSpace(fileID),
		ObjectKey:  strings.TrimSpace(objectKey),
		EntityType: strings.TrimSpace(entityType),
		EntityID:   strings.TrimSpace(entityID),
	})
	if err != nil {
		return err
	}
	if !res.Success {
		return errors.New("media bind failed")
	}
	return nil
}

// PresignedDownloadURL requests a short-lived presigned GET URL (defaults to 900s / 15m).
func (a *Adapter) PresignedDownloadURL(ctx context.Context, fileID, userID, role string, expirySeconds int32) (GetDownloadUrlResponse, error) {
	if a == nil || a.Client == nil {
		return GetDownloadUrlResponse{}, ErrUnavailable
	}
	if expirySeconds <= 0 {
		expirySeconds = 900
	}
	cctx, cancel := a.withAuthAndTimeout(ctx)
	defer cancel()
	return a.Client.GetPresignedDownloadUrl(cctx, GetDownloadUrlRequest{
		FileID:           strings.TrimSpace(fileID),
		RequestingUserID: strings.TrimSpace(userID),
		UserRole:         strings.TrimSpace(role),
		ExpirySeconds:    expirySeconds,
	})
}

type grpcClient struct {
	cc *grpc.ClientConn
}

func (c *grpcClient) ValidateMediaReference(ctx context.Context, in ValidateMediaRequest) (ValidateMediaResponse, error) {
	var out ValidateMediaResponse
	err := c.cc.Invoke(ctx, "/omamx.media.v1.MediaInternalService/ValidateMediaReference", &in, &out)
	return out, err
}

func (c *grpcClient) BindMediaToEntity(ctx context.Context, in BindMediaRequest) (BindMediaResponse, error) {
	var out BindMediaResponse
	err := c.cc.Invoke(ctx, "/omamx.media.v1.MediaInternalService/BindMediaToEntity", &in, &out)
	return out, err
}

func (c *grpcClient) GetPresignedDownloadUrl(ctx context.Context, in GetDownloadUrlRequest) (GetDownloadUrlResponse, error) {
	var out GetDownloadUrlResponse
	err := c.cc.Invoke(ctx, "/omamx.media.v1.MediaInternalService/GetPresignedDownloadUrl", &in, &out)
	return out, err
}
