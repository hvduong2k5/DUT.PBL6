package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/omamx/order-service/internal/domain"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var errInactivePrincipal = errors.New("PRINCIPAL_FORBIDDEN")
var staffPermissionCode = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]{0,159}$`)

// Provisional staff JWT v1. This adapter is the only place where the token's
// authorization claim names are coupled to the service's internal grant model.
type staffAuthorizationClaims struct {
	PrincipalType        string            `json:"principal_type"`
	PrincipalStatus      string            `json:"principal_status"`
	AuthorizationVersion int               `json:"authz_version"`
	PermissionGrants     []staffTokenGrant `json:"permission_grants"`
	Service              string            `json:"service"`
}
type staffTokenGrant struct {
	PermissionCode string     `json:"permissionCode"`
	ScopeType      string     `json:"scopeType"`
	ScopeID        string     `json:"scopeId"`
	ValidFrom      *time.Time `json:"validFrom,omitempty"`
	ValidUntil     *time.Time `json:"validUntil,omitempty"`
}

func (g *staffTokenGrant) UnmarshalJSON(raw []byte) error {
	type grant staffTokenGrant
	var value grant
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*g = staffTokenGrant(value)
	return nil
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

func (c staffAuthorizationClaims) staffGrants(now time.Time) ([]domain.PermissionGrant, bool, error) {
	invalid := errors.New("INVALID_STAFF_AUTHORIZATION")
	if c.Service != "" {
		return nil, false, invalid
	}
	if c.PrincipalType == "" {
		return nil, false, nil
	}
	if c.PrincipalType != "EMPLOYEE" && c.PrincipalType != "CUSTOMER" {
		return nil, false, invalid
	}
	if c.PrincipalStatus != "ACTIVE" {
		return nil, false, errInactivePrincipal
	}
	if c.PrincipalType == "CUSTOMER" {
		return nil, false, nil
	}
	if c.AuthorizationVersion != 1 || c.PermissionGrants == nil || len(c.PermissionGrants) > 128 {
		return nil, true, invalid
	}
	grants := make([]domain.PermissionGrant, 0, len(c.PermissionGrants))
	for _, g := range c.PermissionGrants {
		if !staffPermissionCode.MatchString(g.PermissionCode) {
			return nil, true, invalid
		}
		switch g.ScopeType {
		case "GLOBAL":
			if g.ScopeID != "" {
				return nil, true, invalid
			}
		case "WAREHOUSE", "DEPARTMENT", "STORE", "REGION", "BRANCH", "BUSINESS_UNIT":
			if !authorizationValue(g.ScopeID) {
				return nil, true, invalid
			}
		default:
			return nil, true, invalid
		}
		if g.ValidFrom != nil && g.ValidUntil != nil && !g.ValidFrom.Before(*g.ValidUntil) {
			return nil, true, invalid
		}
		if g.ValidFrom != nil && now.Before(*g.ValidFrom) || g.ValidUntil != nil && !now.Before(*g.ValidUntil) {
			continue
		}
		grants = append(grants, domain.PermissionGrant{PermissionCode: g.PermissionCode, ScopeType: g.ScopeType, ScopeID: g.ScopeID})
	}
	return grants, true, nil
}
