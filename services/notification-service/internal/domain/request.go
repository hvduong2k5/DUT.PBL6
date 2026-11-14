package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUnauthenticated = errors.New("UNAUTHENTICATED")
	ErrForbidden       = errors.New("FORBIDDEN")
	ErrInvalid         = errors.New("INVALID_REQUEST")
	ErrConflict        = errors.New("IDEMPOTENCY_CONFLICT")
	ErrNotFound        = errors.New("NOT_FOUND")
)

var opaque = regexp.MustCompile(`^[A-Za-z0-9:_./-]{1,256}$`)
var keyPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,128}$`)

func ValidKey(s string) bool { return keyPattern.MatchString(s) }

type SourceEvent struct {
	ID               string `json:"id"`
	Source           string `json:"source"`
	Type             string `json:"type"`
	AggregateID      string `json:"aggregate_id"`
	AggregateVersion int64  `json:"aggregate_version"`
	Cohort           string `json:"cohort"`
}
type Command struct {
	SourceEvent  SourceEvent `json:"source_event"`
	Purpose      string      `json:"purpose"`
	Audience     string      `json:"audience"`
	RecipientRef string      `json:"recipient_ref"`
	Locale       string      `json:"locale"`
	ContextRef   string      `json:"context_ref"`
}

// Validate deliberately enables only the local Order confirmation contract.
// Other purposes require owner contracts before onboarding.
func (c Command) Validate() error {
	e := c.SourceEvent
	id, err := uuid.Parse(e.ID)
	if err != nil || id == uuid.Nil || id.String() != e.ID {
		return ErrInvalid
	}
	aid, err := uuid.Parse(e.AggregateID)
	if err != nil || aid == uuid.Nil || aid.String() != e.AggregateID {
		return ErrInvalid
	}
	if e.Source != "order-service" || e.Type != "vn.omama.order.placed.v3" || e.Cohort != "order-core-1" || e.AggregateVersion < 1 || c.Purpose != "ORDER_CONFIRMATION" || c.Audience != "BUYER" || c.Locale != "vi-VN" || c.RecipientRef != "order:"+e.AggregateID+":buyer" || !opaque.MatchString(c.ContextRef) {
		return ErrInvalid
	}
	return nil
}
func hash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (c Command) Fingerprint() string { return hash(c) }
func (c Command) Intent() string {
	return hash([]string{c.SourceEvent.Source, c.SourceEvent.ID, c.Purpose, c.Audience, c.RecipientRef})
}

type Request struct {
	ID        string    `json:"request_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
type Result struct {
	Request
	Duplicate bool `json:"duplicate"`
}
