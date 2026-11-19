package adapters

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/omamx/fulfillment-service/internal/application"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

type S3 struct {
	Client *minio.Client
	Bucket string
}

func NewS3(endpoint, access, secret, bucket string, secure bool) (*S3, error) {
	client, e := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(access, secret, ""), Secure: secure})
	if e != nil {
		return nil, e
	}
	return &S3{client, bucket}, nil
}
func (s *S3) UploadURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, e := s.Client.PresignedPutObject(ctx, s.Bucket, key, ttl)
	if e != nil {
		return "", e
	}
	return u.String(), nil
}
func (s *S3) Stat(ctx context.Context, key string) (application.Object, error) {
	x, e := s.Client.StatObject(ctx, s.Bucket, key, minio.StatObjectOptions{})
	if e != nil {
		code := minio.ToErrorResponse(e).Code
		if code == "NoSuchKey" || code == "NoSuchObject" || code == "NotFound" {
			return application.Object{Missing: true}, nil
		}
	}
	return application.Object{Size: x.Size, Version: x.VersionID}, e
}
func (s *S3) Read(ctx context.Context, key, version string) (io.ReadCloser, error) {
	return s.Client.GetObject(ctx, s.Bucket, key, minio.GetObjectOptions{VersionID: version})
}
func (s *S3) Promote(ctx context.Context, key, version, final string) (string, error) {
	x, e := s.Client.CopyObject(ctx, minio.CopyDestOptions{Bucket: s.Bucket, Object: final}, minio.CopySrcOptions{Bucket: s.Bucket, Object: key, VersionID: version})
	return x.VersionID, e
}
func (s *S3) AccessURL(ctx context.Context, key, version string, ttl time.Duration) (string, error) {
	u, e := s.Client.PresignedGetObject(ctx, s.Bucket, key, ttl, url.Values{"versionId": []string{version}})
	if e != nil {
		return "", e
	}
	return u.String(), nil
}

// Clamd uses INSTREAM, never a shell command or a permissive production stub.
type Clamd struct{ Address string }

func (c Clamd) Scan(ctx context.Context, path string) error {
	conn, e := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "tcp", c.Address)
	if e != nil {
		return e
	}
	defer conn.Close()
	deadline := time.Now().Add(30 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if e = conn.SetDeadline(deadline); e != nil {
		return e
	}
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	if _, e = conn.Write([]byte("zINSTREAM\x00")); e != nil {
		return e
	}
	buf := make([]byte, 32768)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			var size [4]byte
			binary.BigEndian.PutUint32(size[:], uint32(n))
			if _, e = conn.Write(size[:]); e != nil {
				return e
			}
			if _, e = conn.Write(buf[:n]); e != nil {
				return e
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	if _, e = conn.Write([]byte{0, 0, 0, 0}); e != nil {
		return e
	}
	out, e := io.ReadAll(io.LimitReader(conn, 1024))
	if e != nil {
		return e
	}
	verdict := strings.TrimRight(string(out), "\x00\r\n")
	if strings.HasSuffix(verdict, " FOUND") {
		return application.ErrUnsafeMedia
	}
	if verdict != "stream: OK" {
		return fmt.Errorf("media scan not clean")
	}
	return nil
}

func (s *S3) Delete(ctx context.Context, key, version string) error {
	return s.Client.RemoveObject(ctx, s.Bucket, key, minio.RemoveObjectOptions{VersionID: version})
}
