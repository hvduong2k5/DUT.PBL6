package domain

type PermissionGrant struct {
	PermissionCode string `json:"permissionCode"`
	ScopeType      string `json:"scopeType"`
	ScopeID        string `json:"scopeId"`
}

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

func (p Principal) PermissionScope(permission string) PermissionScope {
	s := PermissionScope{}
	if !p.IAM || p.Role != "EMPLOYEE" {
		return s
	}
	for _, g := range p.Grants {
		if g.PermissionCode != permission {
			continue
		}
		if g.ScopeType == "GLOBAL" && g.ScopeID == "" {
			s.Global = true
		}
		if g.ScopeType == "WAREHOUSE" && g.ScopeID != "" {
			s.IDs = append(s.IDs, g.ScopeID)
		}
	}
	return s
}

func (p Principal) HasPermission(permission string) bool {
	s := p.PermissionScope(permission)
	return s.Global || len(s.IDs) > 0
}

// Explicit sandbox mode retains its existing test role model. Production IAM
// always requires the exact permission/scope pair returned by Identity.
func (p Principal) AllowsAction(permission, warehouse string) bool {
	if p.IAM {
		return p.PermissionScope(permission).Allows(warehouse)
	}
	return p.Manager() && (p.Role == "ADMIN" || p.Warehouse == warehouse && warehouse != "")
}

func (p Principal) CanSeeRecipient(o Order) bool {
	if p.IAM {
		return p.PermissionScope("ORDER_SENSITIVE_VIEW").Allows(o.Warehouse)
	}
	return p.Scope() == o.Scope || p.Staff() && p.Role != "CUSTOMER_SERVICE"
}
