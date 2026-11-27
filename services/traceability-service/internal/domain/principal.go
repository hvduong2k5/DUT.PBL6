package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
)

var ErrUnauthenticated = errors.New("verified identity required")
var ErrUnavailable = errors.New("required dependency unavailable")

const (
	ReadInternal = "traceability.read_internal"
	ReadEvidence = "traceability.read_evidence"
	Create       = "traceability.create"
	Edit         = "traceability.edit"
	Submit       = "traceability.submit"
	Approve      = "traceability.approve"
	Publish      = "traceability.publish"
	Suspend      = "traceability.suspend"
	Archive      = "traceability.archive"
)

func KnownPermission(p string) bool {
	switch p {
	case ReadInternal, ReadEvidence, Create, Edit, Submit, Approve, Publish, Suspend, Archive:
		return true
	}
	return false
}

type Principal struct {
	ID          string
	Permissions []string
	Grants      []PermissionGrant
}

type PermissionGrant struct {
	Permission string `json:"permissionCode"`
	ScopeType  string `json:"scopeType"`
	ScopeID    string `json:"scopeId"`
}

// Can remains GLOBAL-only for the legacy API. At derives a request-local
// principal only after an authoritative resource warehouse has been resolved.
func (p Principal) CanAt(permission, warehouse string) bool {
	if p.Can(permission) {
		return true
	}
	if !ValidUUID(p.ID) || !KnownPermission(permission) || !ValidUUID(warehouse) {
		return false
	}
	for _, g := range p.Grants {
		if g.Permission == permission && g.ScopeType == "WAREHOUSE" && strings.EqualFold(g.ScopeID, warehouse) {
			return true
		}
	}
	return false
}

func (p Principal) At(warehouse string) Principal {
	result := Principal{ID: p.ID}
	for _, permission := range []string{ReadInternal, ReadEvidence, Create, Edit, Submit, Approve, Publish, Suspend, Archive} {
		if p.CanAt(permission, warehouse) {
			result.Permissions = append(result.Permissions, permission)
		}
	}
	return result
}

func (p Principal) Has(permission string) bool {
	if p.Can(permission) {
		return true
	}
	for _, g := range p.Grants {
		if g.Permission == permission && g.ScopeType == "WAREHOUSE" && ValidUUID(g.ScopeID) && ValidUUID(p.ID) {
			return true
		}
	}
	return false
}

func (p Principal) Warehouses() (bool, []string) {
	if p.Can(ReadInternal) {
		return true, []string{}
	}
	found := map[string]bool{}
	for _, g := range p.Grants {
		if g.Permission == ReadInternal && g.ScopeType == "WAREHOUSE" && ValidUUID(g.ScopeID) {
			found[strings.ToLower(g.ScopeID)] = true
		}
	}
	result := make([]string, 0, len(found))
	for id := range found {
		result = append(result, id)
	}
	sort.Strings(result)
	return false, result
}

func (p Principal) AuthorizationDigest() string {
	grants := map[string]bool{}
	for _, permission := range p.Permissions {
		if p.Can(permission) {
			grants[permission+"#GLOBAL"] = true
		}
	}
	for _, g := range p.Grants {
		if KnownPermission(g.Permission) && g.ScopeType == "WAREHOUSE" && ValidUUID(g.ScopeID) {
			grants[g.Permission+"#WAREHOUSE#"+strings.ToLower(g.ScopeID)] = true
		}
	}
	keys := make([]string, 0, len(grants))
	for g := range grants {
		keys = append(keys, g)
	}
	sort.Strings(keys)
	sum := sha256.Sum256([]byte(strings.Join(keys, "\n")))
	return hex.EncodeToString(sum[:])
}

func (p Principal) Can(permission string) bool {
	if !ValidUUID(p.ID) || !KnownPermission(permission) {
		return false
	}
	for _, allowed := range p.Permissions {
		if allowed == permission {
			return true
		}
	}
	return false
}
