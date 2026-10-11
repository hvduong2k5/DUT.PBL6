package domain

import (
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound        = errors.New("article not found")
	ErrConflict        = errors.New("article version conflict")
	ErrSlug            = errors.New("slug already reserved")
	ErrState           = errors.New("invalid status transition")
	ErrValidation      = errors.New("invalid article input")
	ErrUnauthenticated = errors.New("verified identity required")
	ErrForbidden       = errors.New("permission denied")
)

type Status string

const (
	Draft     Status = "DRAFT"
	Pending   Status = "PENDING"
	Approved  Status = "APPROVED"
	Rejected  Status = "REJECTED"
	Published Status = "PUBLISHED"
)

func ValidStatus(s Status) bool {
	return s == Draft || s == Pending || s == Approved || s == Rejected || s == Published
}

type Identity struct {
	ID   uuid.UUID
	Role string
}

func (i Identity) Authorize(approval bool) error {
	if i.ID == uuid.Nil {
		return ErrUnauthenticated
	}
	if i.Role == "ADMIN" || i.Role == "CONTENT_APPROVER" || (!approval && i.Role == "CONTENT_EDITOR") {
		return nil
	}
	return ErrForbidden
}

type SEO struct {
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	Keywords     []string        `json:"keywords"`
	CanonicalURL string          `json:"canonical_url"`
	Schema       json.RawMessage `json:"schema,omitempty"`
}
type MediaReference struct {
	ID   uuid.UUID `json:"id"`
	URL  string    `json:"url"`
	Alt  string    `json:"alt"`
	Kind string    `json:"kind"`
}
type Content struct {
	Title    string           `json:"title"`
	Slug     string           `json:"slug"`
	Body     string           `json:"body"`
	Category string           `json:"category"`
	SEO      SEO              `json:"seo"`
	Media    []MediaReference `json:"media"`
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func validURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && u.User == nil
}
func (c Content) Validate() error {
	if strings.TrimSpace(c.Title) == "" || len(c.Title) > 300 || !slugPattern.MatchString(c.Slug) || len(c.Slug) > 200 || strings.TrimSpace(c.Body) == "" || len(c.Body) > 500000 || len(c.Category) > 100 {
		return ErrValidation
	}
	if len(c.SEO.Title) > 300 || len(c.SEO.Description) > 1000 || len(c.SEO.Keywords) > 30 || len(c.SEO.Schema) > 16000 || (c.SEO.CanonicalURL != "" && !validURL(c.SEO.CanonicalURL)) || len(c.Media) > 50 {
		return ErrValidation
	}
	for _, k := range c.SEO.Keywords {
		if len(k) > 100 {
			return ErrValidation
		}
	}
	if len(c.SEO.Schema) > 0 {
		var object map[string]any
		if json.Unmarshal(c.SEO.Schema, &object) != nil || object == nil {
			return ErrValidation
		}
	}
	seen := make(map[uuid.UUID]bool)
	for _, m := range c.Media {
		if m.ID == uuid.Nil || seen[m.ID] || !validURL(m.URL) || len(m.URL) > 2048 || len(m.Alt) > 500 || (m.Kind != "image" && m.Kind != "video") {
			return ErrValidation
		}
		seen[m.ID] = true
	}
	return nil
}

type Revision struct {
	ID        uuid.UUID
	Number    int
	Status    Status
	Content   Content
	CreatedBy uuid.UUID
	CreatedAt time.Time
}
type Decision struct {
	ID         uuid.UUID
	RevisionID uuid.UUID
	Status     Status
	Reason     string
	ActorID    uuid.UUID
	CreatedAt  time.Time
}
type SlugHistory struct {
	Slug      string
	CreatedAt time.Time
}
type Article struct {
	ID                  uuid.UUID
	Version             int64
	CurrentRevisionID   uuid.UUID
	PublishedRevisionID *uuid.UUID
	Visible             bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
	Revisions           []Revision
	Decisions           []Decision
	Slugs               []SlugHistory
}

func (a *Article) Current() *Revision { return a.Revision(a.CurrentRevisionID) }
func (a *Article) Revision(id uuid.UUID) *Revision {
	for n := range a.Revisions {
		if a.Revisions[n].ID == id {
			return &a.Revisions[n]
		}
	}
	return nil
}

// Content snapshots are append-only. Only moderation status may change.
func (a *Article) Edit(c Content, actor Identity, now time.Time) error {
	if err := actor.Authorize(false); err != nil {
		return err
	}
	if err := c.Validate(); err != nil {
		return err
	}
	r := a.Current()
	if r == nil {
		return ErrNotFound
	}
	if r.Status == Pending {
		return ErrState
	}
	next := Revision{ID: uuid.New(), Number: r.Number + 1, Status: Draft, Content: c, CreatedBy: actor.ID, CreatedAt: now}
	a.Revisions = append(a.Revisions, next)
	a.CurrentRevisionID = next.ID
	return nil
}
func (a *Article) Transition(action, reason string, actor Identity, now time.Time) error {
	if err := actor.Authorize(action == "approve" || action == "reject"); err != nil {
		return err
	}
	r := a.Current()
	if r == nil {
		return ErrNotFound
	}
	switch action {
	case "submit":
		if r.Status != Draft && r.Status != Rejected {
			return ErrState
		}
		r.Status = Pending
	case "approve", "reject":
		if r.Status != Pending {
			return ErrState
		}
		if action == "reject" {
			reason = strings.TrimSpace(reason)
			if reason == "" || len(reason) > 2000 {
				return ErrValidation
			}
			r.Status = Rejected
		} else {
			r.Status = Approved
			reason = ""
		}
		a.Decisions = append(a.Decisions, Decision{ID: uuid.New(), RevisionID: r.ID, Status: r.Status, Reason: reason, ActorID: actor.ID, CreatedAt: now})
	case "publish":
		if r.Status != Approved {
			return ErrState
		}
		r.Status = Published
		id := r.ID
		a.PublishedRevisionID = &id
		a.Visible = true
	case "hide":
		if !a.Visible || a.PublishedRevisionID == nil {
			return ErrState
		}
		a.Visible = false
	default:
		return ErrValidation
	}
	return nil
}

type Filter struct {
	Page     int
	Limit    int
	Query    string
	Category string
	Status   Status
	Public   bool
}

func (f Filter) Validate() error {
	if f.Page < 1 || f.Page > 1000000 || f.Limit < 1 || f.Limit > 100 || len(f.Query) > 200 || len(f.Category) > 100 || (f.Status != "" && !ValidStatus(f.Status)) || (f.Public && f.Status != "") {
		return ErrValidation
	}
	return nil
}
