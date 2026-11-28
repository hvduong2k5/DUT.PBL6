package identity

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	identityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/identity/v1"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Staff struct {
	Tokens       *oidc.IDTokenVerifier
	Client       identityv1.StaffAuthorizationClient
	ServiceToken string
	// HTTP is used exclusively by OIDC discovery/JWKS, not business REST.
	HTTP *http.Client
}

func (s Staff) Verify(ctx context.Context, raw string) (domain.Principal, error) {
	if len(raw) > 16384 || s.Tokens == nil {
		return domain.Principal{}, domain.ErrUnauthenticated
	}
	tokenContext := ctx
	if s.HTTP != nil {
		tokenContext = oidc.ClientContext(ctx, s.HTTP)
	}
	token, err := s.Tokens.Verify(tokenContext, raw)
	if err != nil {
		return domain.Principal{}, domain.ErrUnauthenticated
	}
	var claims struct {
		ID      string   `json:"user_id"`
		Type    string   `json:"typ"`
		Service string   `json:"service"`
		Grants  []string `json:"permission_grants"`
	}
	if token.Claims(&claims) != nil || !domain.ValidUUID(claims.ID) || token.Subject == "" || token.IssuedAt.IsZero() || claims.Type != "Bearer" || claims.Service != "" || claims.Grants == nil || len(claims.Grants) > 128 {
		return domain.Principal{}, domain.ErrUnauthenticated
	}
	if s.Client == nil || len(s.ServiceToken) < 32 {
		return domain.Principal{}, domain.ErrUnavailable
	}
	rpcContext, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	rpcContext = metadata.AppendToOutgoingContext(rpcContext, "authorization", "Bearer "+s.ServiceToken, "x-staff-authorization", "Bearer "+raw)
	profile, err := s.Client.GetCurrentStaffGrants(rpcContext, &identityv1.CurrentStaffGrantsRequest{}, grpc.MaxCallRecvMsgSize(32768))
	if err != nil {
		return domain.Principal{}, staffRPCError(err)
	}
	if profile == nil || profile.GetPrincipalId() != strings.ToLower(claims.ID) || profile.GetIdpSubject() != token.Subject || len(profile.GetPermissionGrants()) > 128 {
		return domain.Principal{}, domain.ErrUnavailable
	}
	if profile.GetPrincipalType() != "EMPLOYEE" || profile.GetProvisioningStatus() != "ACTIVE" {
		return domain.Principal{}, domain.ErrForbidden
	}
	principal := domain.Principal{ID: strings.ToLower(claims.ID)}
	tokenGrants := parseGrants(principal.ID, claims.Grants)
	current := domain.Principal{ID: principal.ID}
	for _, grant := range profile.GetPermissionGrants() {
		permission, scope, scopeID := grant.GetPermissionCode(), grant.GetScopeType(), grant.GetScopeId()
		if !domain.KnownPermission(permission) {
			continue
		}
		if scope == "GLOBAL" && scopeID == "" {
			current.Permissions = append(current.Permissions, permission)
		}
		if scope == "WAREHOUSE" && domain.ValidUUID(scopeID) {
			current.Grants = append(current.Grants, domain.PermissionGrant{Permission: permission, ScopeType: scope, ScopeID: strings.ToLower(scopeID)})
		}
	}
	for _, permission := range tokenGrants.Permissions {
		if current.Can(permission) {
			principal.Permissions = append(principal.Permissions, permission)
		}
	}
	seen := map[string]bool{}
	for _, grant := range append(tokenGrants.Grants, current.Grants...) {
		key := grant.Permission + "#" + grant.ScopeID
		if !seen[key] && tokenGrants.CanAt(grant.Permission, grant.ScopeID) && current.CanAt(grant.Permission, grant.ScopeID) {
			seen[key] = true
			principal.Grants = append(principal.Grants, grant)
		}
	}
	return principal, nil
}

func staffRPCError(err error) error {
	code := status.Code(err)
	for _, detail := range status.Convert(err).Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok || info.Domain != "identity.omama.internal" {
			continue
		}
		if code == codes.Unauthenticated && info.Reason == "STAFF_UNAUTHENTICATED" {
			return domain.ErrUnauthenticated
		}
		if code == codes.PermissionDenied && info.Reason == "STAFF_FORBIDDEN" {
			return domain.ErrForbidden
		}
	}
	// Service authentication/configuration and unknown failures are not a user
	// login error. Never fall back to stale token grants or the old /users/me API.
	return domain.ErrUnavailable
}

func parseGrants(id string, grants []string) domain.Principal {
	p := domain.Principal{ID: strings.ToLower(id)}
	for _, raw := range grants {
		parts := strings.Split(raw, "#")
		if len(parts) < 2 || !domain.KnownPermission(parts[0]) {
			continue
		}
		if len(parts) == 2 && parts[1] == "GLOBAL" {
			p.Permissions = append(p.Permissions, parts[0])
		}
		if len(parts) == 3 && parts[1] == "WAREHOUSE" && domain.ValidUUID(parts[2]) {
			p.Grants = append(p.Grants, domain.PermissionGrant{Permission: parts[0], ScopeType: parts[1], ScopeID: strings.ToLower(parts[2])})
		}
	}
	return p
}
