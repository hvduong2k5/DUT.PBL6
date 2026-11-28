package dependency

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/testutil"
	mediav1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/media/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const mediaFile = "66666666-6666-4666-8666-666666666666"

func mediaContent() domain.Content {
	return domain.Content{Certificates: []domain.Certificate{{Type: "OCOP", EvidenceIDs: []string{mediaFile}}}, Evidence: []domain.Evidence{{ID: mediaFile, Reference: "document:" + mediaFile}}}
}

type mediaProvider struct {
	mediav1.UnimplementedMediaInternalServiceServer
	result *mediav1.ValidateMediaResponse
	err    error
	bindOK bool
}

func (s mediaProvider) ValidateMediaReference(_ context.Context, r *mediav1.ValidateMediaRequest) (*mediav1.ValidateMediaResponse, error) {
	if r.GetFileId() != mediaFile || r.GetObjectKey() != "" || r.GetExpectedPurpose() != "OCOP_CERTIFICATE" {
		return nil, status.Error(codes.InvalidArgument, "incorrect reference")
	}
	return s.result, s.err
}
func (s mediaProvider) BindMediaToEntity(_ context.Context, r *mediav1.BindMediaRequest) (*mediav1.BindMediaResponse, error) {
	if r.GetFileId() != mediaFile || r.GetEntityType() != "TraceabilityProfile" || !domain.ValidUUID(r.GetEntityId()) {
		return nil, status.Error(codes.InvalidArgument, "incorrect binding")
	}
	return &mediav1.BindMediaResponse{Success: s.bindOK}, s.err
}
func TestMediaRPCValidationAndBinding(t *testing.T) {
	for _, tc := range []struct {
		name, sha, mime, state string
		valid                  bool
		size                   int64
		err                    error
		want                   error
	}{
		{"ready", strings.Repeat("a", 64), "application/pdf", "READY", true, 100, nil, nil},
		{"wrong purpose or not ready", strings.Repeat("a", 64), "application/pdf", "PROCESSING", false, 100, nil, &domain.ValidationError{}},
		{"bad checksum", "not-a-checksum", "application/pdf", "READY", true, 100, nil, domain.ErrUnavailable},
		{"bad size", strings.Repeat("a", 64), "application/pdf", "READY", true, 0, nil, domain.ErrUnavailable},
		{"wrong MIME", strings.Repeat("a", 64), "video/mp4", "READY", true, 100, nil, &domain.ValidationError{}},
		{"peer down", "", "", "", false, 0, status.Error(codes.Unavailable, "offline"), domain.ErrUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn := testutil.GRPC(t, func(options ...grpc.ServerOption) (*grpc.Server, error) {
				server := grpc.NewServer(options...)
				mediav1.RegisterMediaInternalServiceServer(server, mediaProvider{result: &mediav1.ValidateMediaResponse{IsValid: tc.valid, Status: tc.state, ObjectKey: "certificates/file.pdf", SizeBytes: tc.size, MimeType: tc.mime, Sha256Checksum: tc.sha}, err: tc.err, bindOK: true})
				return server, nil
			})
			source := Live{MediaClient: mediav1.NewMediaInternalServiceClient(conn)}
			err := source.ValidateMedia(context.Background(), mediaContent())
			if _, ok := tc.want.(*domain.ValidationError); ok {
				var validation *domain.ValidationError
				if !errors.As(err, &validation) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, tc.want) {
				t.Fatal(err)
			}
			if tc.want == nil {
				if err := source.BindMedia(context.Background(), "22222222-2222-4222-8222-222222222222", mediaFile); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestMediaReferencesAreImmutableFileIDs(t *testing.T) {
	content := mediaContent()
	content.Certificates = append(content.Certificates, content.Certificates[0])
	ids, err := domain.CertificateMediaIDs(content)
	if err != nil || len(ids) != 1 || ids[0] != mediaFile {
		t.Fatal(ids, err)
	}
	for _, ref := range []string{"https://storage.example/file.pdf", "certificates/file.pdf", "document:invalid", ""} {
		content.Evidence[0].Reference = ref
		if _, err := domain.CertificateMediaIDs(content); err == nil {
			t.Fatal("unsafe reference accepted", ref)
		}
	}
}
