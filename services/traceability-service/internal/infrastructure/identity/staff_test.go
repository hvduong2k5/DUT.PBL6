package identity

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/testutil"
	identityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/identity/v1"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const testStaffID = "22222222-2222-4222-8222-222222222222"
const testIssuer = "https://identity.example/realms/omama"
const iamCredential = "synthetic-dedicated-trace-identity-credential"

type iamProvider struct {
	identityv1.UnimplementedStaffAuthorizationServer
	lookup func(context.Context, *identityv1.CurrentStaffGrantsRequest) (*identityv1.CurrentStaffGrantsResponse, error)
}

func (s iamProvider) GetCurrentStaffGrants(ctx context.Context, r *identityv1.CurrentStaffGrantsRequest) (*identityv1.CurrentStaffGrantsResponse, error) {
	return s.lookup(ctx, r)
}
func signedStaff(t *testing.T, audience string, grants []string) (string, *oidc.IDTokenVerifier) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	data, _ := json.Marshal(map[string]any{"iss": testIssuer, "aud": audience, "sub": "staff-1", "user_id": testStaffID, "typ": "Bearer", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(), "permission_grants": grants})
	signed, err := signer.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := signed.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	verifier := oidc.NewVerifier(testIssuer, &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&key.PublicKey}}, &oidc.Config{ClientID: "omama-api"})
	return raw, verifier
}
func iamClient(t *testing.T, raw string, lookup func() (*identityv1.CurrentStaffGrantsResponse, error)) identityv1.StaffAuthorizationClient {
	t.Helper()
	connection := testutil.GRPC(t, func(options ...grpc.ServerOption) (*grpc.Server, error) {
		server := grpc.NewServer(options...)
		identityv1.RegisterStaffAuthorizationServer(server, iamProvider{lookup: func(ctx context.Context, _ *identityv1.CurrentStaffGrantsRequest) (*identityv1.CurrentStaffGrantsResponse, error) {
			md, _ := metadata.FromIncomingContext(ctx)
			if values := md.Get("authorization"); len(values) != 1 || values[0] != "Bearer "+iamCredential {
				t.Error("incorrect service credential metadata")
			}
			if values := md.Get("x-staff-authorization"); len(values) != 1 || values[0] != "Bearer "+raw {
				t.Error("incorrect staff credential metadata")
			}
			return lookup()
		}})
		return server, nil
	})
	return identityv1.NewStaffAuthorizationClient(connection)
}
func iamAuthFailure(code codes.Code, reason string) error {
	result, _ := status.New(code, "Safe authorization failure").WithDetails(&errdetails.ErrorInfo{Domain: "identity.omama.internal", Reason: reason})
	return result.Err()
}
func currentGrant(permission, scope, id string) *identityv1.CurrentStaffGrantsResponse {
	return &identityv1.CurrentStaffGrantsResponse{PrincipalId: testStaffID, IdpSubject: "staff-1", PrincipalType: "EMPLOYEE", ProvisioningStatus: "ACTIVE", PermissionGrants: []*identityv1.StaffPermissionGrant{{PermissionCode: permission, ScopeType: scope, ScopeId: id}}}
}
func TestStaffRequiresLiveGlobalGrant(t *testing.T) {
	for _, tc := range []struct {
		name, audience, scope string
		rpcErr, want          error
		allowed               bool
	}{
		{name: "global grant", audience: "omama-api", scope: "GLOBAL", allowed: true},
		{name: "grant revoked", audience: "omama-api", scope: "DEPARTMENT"},
		{name: "disabled staff", audience: "omama-api", rpcErr: iamAuthFailure(codes.PermissionDenied, "STAFF_FORBIDDEN"), want: domain.ErrForbidden},
		{name: "staff session invalid", audience: "omama-api", rpcErr: iamAuthFailure(codes.Unauthenticated, "STAFF_UNAUTHENTICATED"), want: domain.ErrUnauthenticated},
		{name: "service credential invalid", audience: "omama-api", rpcErr: iamAuthFailure(codes.Unauthenticated, "SERVICE_AUTHENTICATION_FAILED"), want: domain.ErrUnavailable},
		{name: "service capability denied", audience: "omama-api", rpcErr: iamAuthFailure(codes.PermissionDenied, "SERVICE_PERMISSION_DENIED"), want: domain.ErrUnavailable},
		{name: "missing auth reason", audience: "omama-api", rpcErr: status.Error(codes.Unauthenticated, "Unknown auth failure"), want: domain.ErrUnavailable},
		{name: "Identity offline", audience: "omama-api", rpcErr: status.Error(codes.Unavailable, "Offline"), want: domain.ErrUnavailable},
		{name: "RPC not implemented", audience: "omama-api", rpcErr: status.Error(codes.Unimplemented, "Old provider"), want: domain.ErrUnavailable},
		{name: "wrong audience", audience: "other-api", want: domain.ErrUnauthenticated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, tokens := signedStaff(t, tc.audience, []string{domain.Approve + "#GLOBAL"})
			client := iamClient(t, raw, func() (*identityv1.CurrentStaffGrantsResponse, error) {
				return currentGrant(domain.Approve, tc.scope, ""), tc.rpcErr
			})
			p, err := (Staff{Tokens: tokens, Client: client, ServiceToken: iamCredential}).Verify(context.Background(), raw)
			if !errors.Is(err, tc.want) || p.Can(domain.Approve) != tc.allowed {
				t.Fatalf("unexpected authorization %+v %v", p, err)
			}
		})
	}
}
func TestStaffIntersectsGlobalAndWarehouseCoverage(t *testing.T) {
	const a = "77777777-7777-4777-8777-777777777777"
	const b = "88888888-8888-4888-8888-888888888888"
	for _, tc := range []struct {
		name, tokenScope, currentScope, currentID, principalType string
		allowed                                                  bool
	}{
		{"global narrowed by current warehouse", "GLOBAL", "WAREHOUSE", a, "EMPLOYEE", true},
		{"warehouse token cannot expand", "WAREHOUSE#" + a, "GLOBAL", "", "EMPLOYEE", true},
		{"disjoint warehouse denied", "WAREHOUSE#" + a, "WAREHOUSE", b, "EMPLOYEE", false},
		{"department not warehouse", "GLOBAL", "DEPARTMENT", a, "EMPLOYEE", false},
		{"customer rejected", "GLOBAL", "GLOBAL", "", "CUSTOMER", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, tokens := signedStaff(t, "omama-api", []string{domain.ReadInternal + "#" + tc.tokenScope})
			client := iamClient(t, raw, func() (*identityv1.CurrentStaffGrantsResponse, error) {
				r := currentGrant(domain.ReadInternal, tc.currentScope, tc.currentID)
				r.PrincipalType = tc.principalType
				return r, nil
			})
			p, err := (Staff{Tokens: tokens, Client: client, ServiceToken: iamCredential}).Verify(context.Background(), raw)
			if tc.principalType == "CUSTOMER" {
				if !errors.Is(err, domain.ErrForbidden) {
					t.Fatal("customer accepted", err)
				}
				return
			}
			if err != nil || p.CanAt(domain.ReadInternal, a) != tc.allowed || p.Can(domain.ReadInternal) {
				t.Fatal("permission coverage expanded or lost", err)
			}
		})
	}
}
func TestStaffRejectsMalformedRPCResponse(t *testing.T) {
	for _, testCase := range []string{"id mismatch", "subject mismatch", "grant count", "message size"} {
		t.Run(testCase, func(t *testing.T) {
			raw, tokens := signedStaff(t, "omama-api", []string{domain.ReadInternal + "#GLOBAL"})
			client := iamClient(t, raw, func() (*identityv1.CurrentStaffGrantsResponse, error) {
				r := currentGrant(domain.ReadInternal, "GLOBAL", "")
				switch testCase {
				case "id mismatch":
					r.PrincipalId = "33333333-3333-4333-8333-333333333333"
				case "subject mismatch":
					r.IdpSubject = "other-subject"
				case "grant count":
					for len(r.PermissionGrants) < 129 {
						r.PermissionGrants = append(r.PermissionGrants, r.PermissionGrants[0])
					}
				case "message size":
					r.IdpSubject = strings.Repeat("x", 32769)
				}
				return r, nil
			})
			if _, err := (Staff{Tokens: tokens, Client: client, ServiceToken: iamCredential}).Verify(context.Background(), raw); !errors.Is(err, domain.ErrUnavailable) {
				t.Fatal("invalid authority response accepted", err)
			}
		})
	}
}
