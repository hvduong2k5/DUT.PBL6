package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	d "github.com/omamx/fulfillment-service/internal/domain"
	"github.com/omamx/fulfillment-service/internal/persistence"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Object struct {
	Missing bool
	Size    int64
	Version string
}
type Storage interface {
	Delete(context.Context, string, string) error
	UploadURL(context.Context, string, time.Duration) (string, error)
	Stat(context.Context, string) (Object, error)
	Read(context.Context, string, string) (io.ReadCloser, error)
	Promote(context.Context, string, string, string) (string, error)
	AccessURL(context.Context, string, string, time.Duration) (string, error)
}

var ErrUnsafeMedia = errors.New("unsafe media")

type Scanner interface {
	Scan(context.Context, string) error
}
type Media struct {
	Store                        *persistence.Store
	Storage                      Storage
	Scanner                      Scanner
	UploadEnabled, AccessEnabled bool
}

func (m *Media) Create(ctx context.Context, p d.Principal, id, key string, in d.MediaRequest) (any, error) {
	if !m.UploadEnabled || m.Storage == nil || m.Scanner == nil {
		return nil, d.Fail("MEDIA_CAPABILITY_UNAVAILABLE", 503)
	}
	f, e := m.Store.CreateMedia(ctx, p, id, key, in)
	if e != nil {
		return nil, e
	}
	remaining := time.Until(f.Expires)
	if remaining <= 0 {
		return nil, d.Fail("MEDIA_SESSION_EXPIRED", 410)
	}
	url, e := m.Storage.UploadURL(ctx, f.StagingKey, remaining)
	if e != nil {
		return nil, e
	}
	return map[string]any{"file_id": f.ID, "status": "CREATED", "upload_url": url, "expires_at": f.Expires}, nil
}
func (m *Media) Access(ctx context.Context, p d.Principal, id, purpose string) (any, error) {
	if !m.AccessEnabled || m.Storage == nil {
		return nil, d.Fail("MEDIA_CAPABILITY_UNAVAILABLE", 503)
	}
	f, e := m.Store.File(ctx, id)
	if e != nil {
		return nil, e
	}
	t, e := m.Store.Task(ctx, f.TaskID)
	if e != nil {
		return nil, e
	}
	if !p.Can("fulfillment.media.view", t.Warehouse) {
		return nil, d.Fail("FORBIDDEN", 403)
	}
	if f.Status != "VERIFIED" {
		return nil, d.Fail("MEDIA_NOT_VERIFIED", 409)
	}
	audit, e := m.Store.MediaAccessAudit(ctx, p, f, purpose)
	if e != nil {
		return nil, e
	}
	url, e := m.Storage.AccessURL(ctx, f.FinalKey, f.ObjectVersion, 5*time.Minute)
	if e != nil {
		return nil, e
	}
	return map[string]any{"url": url, "expires_at": time.Now().UTC().Add(5 * time.Minute), "access_audit_id": audit}, nil
}
func (m *Media) Verify(ctx context.Context, id string) error {
	if m.Storage == nil || m.Scanner == nil {
		return d.Fail("MEDIA_CAPABILITY_UNAVAILABLE", 503)
	}
	f, e := m.Store.File(ctx, id)
	if e != nil {
		return e
	}
	if f.Status == "VERIFIED" || f.Status == "REJECTED" {
		return nil
	}
	if f.Status != "VERIFYING" {
		return d.Fail("INVALID_MEDIA_STATE", 409)
	}
	meta, e := m.Storage.Stat(ctx, f.StagingKey)
	if e != nil {
		return e
	}
	if meta.Missing {
		return d.Fail("MEDIA_OBJECT_NOT_OBSERVED", 503)
	}
	if meta.Version == "" || meta.Version == "null" {
		return d.Fail("STORAGE_VERSIONING_REQUIRED", 503)
	}
	if meta.Size != f.Size {
		return m.Store.FinishMedia(ctx, f, "", "", "", false)
	}
	reader, e := m.Storage.Read(ctx, f.StagingKey, meta.Version)
	if e != nil {
		return e
	}
	defer reader.Close()
	tmp, e := os.CreateTemp("", "fulfillment-verify-*")
	if e != nil {
		return e
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	hash := sha256.New()
	n, e := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(reader, f.Size+1))
	if e != nil {
		return e
	}
	checksum := hex.EncodeToString(hash.Sum(nil))
	if n != f.Size || checksum != f.DeclaredHash {
		return m.Store.FinishMedia(ctx, f, "", "", "", false)
	}
	t, e := m.Store.Task(ctx, f.TaskID)
	if e != nil {
		return e
	}
	if e = probe(ctx, tmp.Name(), f.Type, t.Inputs.Policy.MaxVideoSeconds); e != nil {
		return m.Store.FinishMedia(ctx, f, "", "", "", false)
	}
	if e = m.Scanner.Scan(ctx, tmp.Name()); e != nil {
		if errors.Is(e, ErrUnsafeMedia) {
			return m.Store.FinishMedia(ctx, f, "", "", "", false)
		}
		return e
	}
	final := "packing-evidence/verified/" + f.ID + "/" + checksum
	version, e := m.Storage.Promote(ctx, f.StagingKey, meta.Version, final)
	if e != nil {
		return e
	}
	if version == "" || version == "null" {
		return d.Fail("STORAGE_VERSIONING_REQUIRED", 503)
	}
	return m.Store.FinishMedia(ctx, f, final, version, checksum, true)
}
func probe(ctx context.Context, path, mediaType string, maxSeconds int32) error {
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, e := exec.CommandContext(c, "ffprobe", "-v", "error", "-show_entries", "format=duration,format_name:stream=codec_type", "-of", "json", path).Output()
	if e != nil {
		return e
	}
	var meta struct {
		Format struct {
			Duration string `json:"duration"`
			Name     string `json:"format_name"`
		} `json:"format"`
		Streams []struct {
			Type string `json:"codec_type"`
		} `json:"streams"`
	}
	if e = json.Unmarshal(out, &meta); e != nil {
		return e
	}
	seconds, e := strconv.ParseFloat(meta.Format.Duration, 64)
	if e != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 || seconds > float64(maxSeconds) {
		return fmt.Errorf("invalid media duration")
	}
	video := false
	for _, s := range meta.Streams {
		if s.Type == "video" {
			video = true
		}
	}
	if !video {
		return fmt.Errorf("video stream required")
	}
	if (mediaType == "video/mp4" && !strings.Contains(meta.Format.Name, "mp4")) || (mediaType == "video/webm" && !strings.Contains(meta.Format.Name, "webm")) {
		return fmt.Errorf("media container/type mismatch")
	}
	return nil
}

func (m *Media) Cleanup(ctx context.Context) error {
	if m.Storage == nil {
		return nil
	}
	f, found, e := m.Store.ClaimCleanup(ctx)
	if e != nil || !found {
		return e
	}
	meta, e := m.Storage.Stat(ctx, f.StagingKey)
	if e != nil {
		return e
	}
	if meta.Missing {
		return m.Store.Deleted(ctx, f)
	}
	if e = m.Storage.Delete(ctx, f.StagingKey, meta.Version); e != nil {
		return e
	}
	return m.Store.Deleted(ctx, f)
}
