package identity

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

type Staff struct {
	Tokens      *oidc.IDTokenVerifier
	IdentityURL string
	HTTP        *http.Client
}

func (s Staff) Verify(ctx context.Context, raw string) (domain.Principal, error) {
	if len(raw) > 16384 {
		return domain.Principal{}, domain.ErrUnauthenticated
	}
	token, err := s.Tokens.Verify(oidc.ClientContext(ctx, s.HTTP), raw)
	if err != nil {
		return domain.Principal{}, domain.ErrUnauthenticated
	}
	var claims struct {
		ID      string   `json:"user_id"`
		Type    string   `json:"typ"`
		Service string   `json:"service"`
		Grants  []string `json:"permission_grants"`
	}
	if token.Claims(&claims) != nil || !domain.ValidUUID(claims.ID) || token.Subject == "" ||
		token.IssuedAt.IsZero() || claims.Type != "Bearer" || claims.Service != "" || claims.Grants == nil || len(claims.Grants) > 128 {
		return domain.Principal{}, domain.ErrUnauthenticated
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.IdentityURL+"/api/v1/users/me", nil)
	if err != nil {
		return domain.Principal{}, domain.ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return domain.Principal{}, domain.ErrUnavailable
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return domain.Principal{}, domain.ErrUnauthenticated
	case http.StatusForbidden, http.StatusNotFound:
		return domain.Principal{}, domain.ErrForbidden
	case http.StatusOK:
	default:
		return domain.Principal{}, domain.ErrUnavailable
	}
	var profile struct {
		Success bool `json:"success"`
		Data    struct {
			ID      string `json:"principalId"`
			Subject string `json:"idpSubject"`
			Type    string `json:"principalType"`
			Status  string `json:"provisioningStatus"`
			Grants  []struct {
				Permission string `json:"permissionCode"`
				Scope      string `json:"scopeType"`
			} `json:"permissionGrants"`
		} `json:"data"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 32768)).Decode(&profile) != nil ||
		!profile.Success || profile.Data.ID != claims.ID || profile.Data.Subject != token.Subject {
		return domain.Principal{}, domain.ErrUnavailable
	}
	if profile.Data.Type != "EMPLOYEE" || profile.Data.Status != "ACTIVE" {
		return domain.Principal{}, domain.ErrForbidden
	}
	principal := domain.Principal{ID: claims.ID}
	current := make(map[string]bool)
	for _, grant := range profile.Data.Grants {
		if grant.Scope == "GLOBAL" {
			current[grant.Permission] = true
		}
	}
	for _, grant := range claims.Grants {
		if permission, ok := strings.CutSuffix(grant, "#GLOBAL"); ok && domain.KnownPermission(permission) && current[permission] {
			principal.Permissions = append(principal.Permissions, permission)
		}
	}
	return principal, nil
}
