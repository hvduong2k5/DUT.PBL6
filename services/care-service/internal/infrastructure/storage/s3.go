package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/omamx/care-service/internal/domain"
)

type S3Config struct {
	Endpoint         string
	AccessKey        string
	SecretKey        string
	Bucket           string
	Region           string
	UseSSL           bool
	AutoCreateBucket bool
}

type S3 struct {
	client           *minio.Client
	bucket           string
	region           string
	autoCreateBucket bool
}

func NewS3(cfg S3Config) (*S3, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}
	return &S3{client: client, bucket: cfg.Bucket, region: cfg.Region, autoCreateBucket: cfg.AutoCreateBucket}, nil
}

func (s *S3) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if !s.autoCreateBucket {
		return fmt.Errorf("required bucket %q does not exist", s.bucket)
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: s.region})
}

func (s *S3) Health(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("required bucket %q does not exist", s.bucket)
	}
	return nil
}

func (s *S3) PresignUpload(ctx context.Context, objectKey, contentType, checksum string, ttl time.Duration) (string, domain.UploadHeaders, error) {
	headers := make(http.Header)
	headers.Set("Content-Type", contentType)
	if checksum != "" {
		headers.Set("X-Amz-Meta-Sha256", strings.ToLower(checksum))
	}
	presigned, err := s.client.PresignHeader(ctx, http.MethodPut, s.bucket, objectKey, ttl, make(url.Values), headers)
	if err != nil {
		return "", nil, err
	}
	resultHeaders := domain.UploadHeaders{"Content-Type": contentType}
	if checksum != "" {
		resultHeaders["X-Amz-Meta-Sha256"] = strings.ToLower(checksum)
	}
	return presigned.String(), resultHeaders, nil
}

func (s *S3) Stat(ctx context.Context, objectKey string) (domain.StoredObject, error) {
	info, err := s.client.StatObject(ctx, s.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return domain.StoredObject{}, err
	}
	return domain.StoredObject{
		Size:           info.Size,
		ContentType:    info.ContentType,
		ETag:           info.ETag,
		ChecksumSHA256: strings.ToLower(info.Metadata.Get("X-Amz-Meta-Sha256")),
	}, nil
}

func (s *S3) PresignDownload(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	presigned, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, ttl, nil)
	if err != nil {
		return "", err
	}
	return presigned.String(), nil
}

func (s *S3) Remove(ctx context.Context, objectKey string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}
