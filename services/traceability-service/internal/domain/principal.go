package domain

import "errors"

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
