package domain

type PermissionGrant struct {
	PermissionCode string `json:"permissionCode"`
	ScopeType      string `json:"scopeType"`
	ScopeID        string `json:"scopeId"`
}

// PermissionScope never combines a permission with another role's scopes.
type PermissionScope struct {
	Global bool
	IDs    []string
}

func (s PermissionScope) Allows(id string) bool {
	if s.Global {
		return true
	}
	for _, allowed := range s.IDs {
		if id != "" && id == allowed {
			return true
		}
	}
	return false
}

func ScopeFor(grants []PermissionGrant, permission, kind string) PermissionScope {
	s := PermissionScope{}
	for _, g := range grants {
		if g.PermissionCode != permission {
			continue
		}
		if g.ScopeType == "GLOBAL" && g.ScopeID == "" {
			s.Global = true
		}
		if g.ScopeType == kind && g.ScopeID != "" {
			s.IDs = append(s.IDs, g.ScopeID)
		}
	}
	return s
}
