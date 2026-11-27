package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

const (
	ActionSubmit  = "revision.submit"
	ActionApprove = "revision.approve"
	ActionReject  = "revision.reject"
	ActionPublish = "revision.publish"
)

type RevisionCommand struct {
	ExpectedVersion int64  `json:"expected_version"`
	Reason          string `json:"reason,omitempty"`
}
type NewRevisionRequest struct {
	ExpectedVersion int64  `json:"expected_version"`
	FromRevision    int    `json:"from_revision"`
	Reason          string `json:"reason,omitempty"`
}

func ValidateReason(reason string, required bool) error {
	if !textWithin(reason, 1000, required) {
		return invalid("reason")
	}
	return nil
}
func (s DependencySnapshot) SourceHash() (string, error) {
	s.CheckedAt = time.Time{}
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
