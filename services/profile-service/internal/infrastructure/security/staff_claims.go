package security

import (
	"errors"
	"github.com/omamx/profile-service/internal/domain"
	"regexp"
	"strings"
	"unicode"
)

var staffPermissionCode = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]{0,159}$`)

// Staff authorization claims, aligned with the Identity -> Keycloak sync
// contract implemented by the identity service: Identity recomputes each
// employee's effective grants from IAM assignments and writes them onto the
// Keycloak user as the multivalued attribute "permission_grants" — one string
// per grant, "PERMISSION#SCOPE_TYPE" for GLOBAL grants and
// "PERMISSION#SCOPE_TYPE#SCOPE_ID" otherwise. A realm protocol mapper copies
// the attribute into the access token under the same claim name.
//
// A token that carries the claim (even as an empty list) is a staff token and
// is authorized from this signed snapshot alone: Identity filtered the grants
// by assignment validity at sync time, and revocation takes effect at the
// next token issuance (bounded by the access-token lifespan), with lifecycle
// changes additionally forcing re-login. A token without the claim is not a
// staff token and follows the customer path. This adapter is the only place
// where the token's wire format is coupled to the internal grant model.
type staffAuthorizationClaims struct {
	PermissionGrants []string `json:"permission_grants"`
	Service          string   `json:"service"`
}

func authorizationValue(value string) bool {
	if value == "" || len(value) > 160 || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func (c staffAuthorizationClaims) staffGrants() ([]domain.PermissionGrant, bool, error) {
	invalid := errors.New("INVALID_STAFF_AUTHORIZATION")
	if c.Service != "" {
		return nil, false, invalid
	}
	if c.PermissionGrants == nil {
		return nil, false, nil
	}
	if len(c.PermissionGrants) > 128 {
		return nil, true, invalid
	}
	grants := make([]domain.PermissionGrant, 0, len(c.PermissionGrants))
	for _, raw := range c.PermissionGrants {
		parts := strings.Split(raw, "#")
		if len(parts) < 2 || len(parts) > 3 {
			return nil, true, invalid
		}
		code, scopeType := parts[0], parts[1]
		if !staffPermissionCode.MatchString(code) {
			return nil, true, invalid
		}
		var scopeID string
		switch scopeType {
		case "GLOBAL":
			if len(parts) != 2 {
				return nil, true, invalid
			}
		case "WAREHOUSE", "DEPARTMENT":
			if len(parts) != 3 || !authorizationValue(parts[2]) {
				return nil, true, invalid
			}
			scopeID = parts[2]
		default:
			return nil, true, invalid
		}
		grants = append(grants, domain.PermissionGrant{PermissionCode: code, ScopeType: scopeType, ScopeID: scopeID})
	}
	return grants, true, nil
}
