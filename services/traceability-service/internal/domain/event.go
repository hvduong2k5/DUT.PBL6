package domain

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/jsonutil"
	"regexp"
	"strings"
	"time"
)

type EventData struct {
	ProfileID        string `json:"profile_id"`
	AggregateVersion int64  `json:"aggregate_version"`
	Revision         int    `json:"revision"`
	Mode             string `json:"mode"`
	Status           string `json:"status"`
	Eligible         *bool  `json:"eligible,omitempty"`
	ReasonCode       string `json:"reason_code,omitempty"`
}
type CloudEvent struct {
	SpecVersion     string    `json:"specversion"`
	ID              string    `json:"id"`
	Source          string    `json:"source"`
	Type            string    `json:"type"`
	Subject         string    `json:"subject"`
	Time            time.Time `json:"time"`
	DataContentType string    `json:"datacontenttype"`
	TraceParent     string    `json:"traceparent"`
	Data            EventData `json:"data"`
}

var tracePattern = regexp.MustCompile(`^00-[a-f0-9]{32}-[a-f0-9]{16}-0[01]$`)

func ValidTraceParent(trace string) bool {
	return tracePattern.MatchString(trace) && trace[3:35] != strings.Repeat("0", 32) && trace[36:52] != strings.Repeat("0", 16)
}
func NewTraceParent() (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "00-" + hex.EncodeToString(b[:16]) + "-" + hex.EncodeToString(b[16:]) + "-01", nil
}

func ChildTraceParent(parent string) (string, error) {
	if !ValidTraceParent(parent) {
		return "", ErrInvalidCommand
	}
	child, err := NewTraceParent()
	if err != nil {
		return "", err
	}
	return "00-" + parent[3:35] + "-" + child[36:52] + "-" + parent[53:55], nil
}

func ValidateEvent(message OutboxMessage) error {
	var e CloudEvent
	if jsonutil.Decode(message.Payload, &e) != nil || !ValidUUID(message.ID) || e.ID != message.ID || e.SpecVersion != "1.0" || e.Source != "https://omama.vn/services/traceability-service" || e.Type != message.Type || e.DataContentType != "application/json" || !ValidTraceParent(e.TraceParent) || e.Time.IsZero() {
		return ErrInvalidCommand
	}
	switch e.Type {
	case "vn.omama.traceability.profile.published.v1", "vn.omama.traceability.profile.suspended.v1", "vn.omama.traceability.profile.archived.v1", "vn.omama.traceability.certification.status.changed.v1":
	default:
		return ErrInvalidCommand
	}
	if message.Topic != "traceability.events.v1" || !ValidUUID(message.ProfileID) || message.Key != message.ProfileID || e.Subject != "profile:"+message.ProfileID || e.Data.ProfileID != message.ProfileID || e.Data.AggregateVersion != message.Version || message.Version < 1 || e.Data.Revision < 1 {
		return ErrInvalidCommand
	}
	if e.Data.Mode != "sandbox" && e.Data.Mode != "integration" {
		return ErrInvalidCommand
	}
	switch e.Data.Status {
	case "ACTIVE", "SUSPENDED", "ARCHIVED":
	default:
		return ErrInvalidCommand
	}
	switch e.Type {
	case "vn.omama.traceability.profile.published.v1":
		if e.Data.Status != "ACTIVE" {
			return ErrInvalidCommand
		}
	case "vn.omama.traceability.profile.suspended.v1":
		if e.Data.Status != "SUSPENDED" {
			return ErrInvalidCommand
		}
	case "vn.omama.traceability.profile.archived.v1":
		if e.Data.Status != "ARCHIVED" {
			return ErrInvalidCommand
		}
	case "vn.omama.traceability.certification.status.changed.v1":
		if e.Data.Status != "ACTIVE" || e.Data.Eligible == nil || *e.Data.Eligible || (e.Data.ReasonCode != "CERTIFICATE_EXPIRED" && e.Data.ReasonCode != "CERTIFICATE_REVIEW_DUE") {
			return ErrInvalidCommand
		}
	}
	if e.Type != "vn.omama.traceability.certification.status.changed.v1" && (e.Data.Eligible != nil || e.Data.ReasonCode != "") {
		return ErrInvalidCommand
	}
	return nil
}
