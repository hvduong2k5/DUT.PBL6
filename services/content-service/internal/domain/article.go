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

// Permission is a content permission code issued by Identity. Role names never
// authorize anything here, including ADMIN: only a granted permission does.
type Permission string

const (
	PermView    Permission = "CONTENT_ARTICLE_VIEW"
	PermCreate  Permission = "CONTENT_ARTICLE_CREATE"
	PermEdit    Permission = "CONTENT_ARTICLE_EDIT"
	PermSubmit  Permission = "CONTENT_ARTICLE_SUBMIT"
	PermApprove Permission = "CONTENT_ARTICLE_APPROVE"
	PermReject  Permission = "CONTENT_ARTICLE_REJECT"
	PermPublish Permission = "CONTENT_ARTICLE_PUBLISH"
	PermHide    Permission = "CONTENT_ARTICLE_HIDE"
)

const ScopeGlobal = "GLOBAL"

// Grant binds one permission to the scope it was issued for. Only GLOBAL grants
// authorize content operations in this service version.
type Grant struct {
	Permission Permission
	ScopeType  string
	ScopeID    string
}

// Identity is a verified EMPLOYEE principal with its signed permission grants.
type Identity struct {
	ID     uuid.UUID
	Grants []Grant
}

// Authorize requires a verified identity and a GLOBAL grant for the permission.
func (i Identity) Authorize(p Permission) error {
	if i.ID == uuid.Nil {
		return ErrUnauthenticated
	}
	for _, g := range i.Grants {
		if g.Permission == p && g.ScopeType == ScopeGlobal {
			return nil
		}
	}
	return ErrForbidden
}

// Staff reports whether the identity holds any GLOBAL grant. The admin API is
// closed to every other principal, whatever role names it may carry.
func (i Identity) Staff() bool {
	if i.ID == uuid.Nil {
		return false
	}
	for _, g := range i.Grants {
		if g.ScopeType == ScopeGlobal {
			return true
		}
	}
	return false
}

// ActionPermission maps an editorial action to the permission that guards it.
// Restoring a hidden article re-publishes an already approved snapshot.
func ActionPermission(action string) (Permission, bool) {
	switch action {
	case "submit":
		return PermSubmit, true
	case "approve":
		return PermApprove, true
	case "reject":
		return PermReject, true
	case "publish", "restore":
		return PermPublish, true
	case "hide":
		return PermHide, true
	}
	return "", false
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

// AuditEvent is the append-only trail of every editorial action.
type AuditEvent struct {
	ID         uuid.UUID
	RevisionID uuid.UUID
	Action     string
	ActorID    uuid.UUID
	Reason     string
	CreatedAt  time.Time
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
	Events              []AuditEvent
}

func (a *Article) audit(action string, revisionID uuid.UUID, reason string, actor Identity, now time.Time) {
	a.Events = append(a.Events, AuditEvent{ID: uuid.New(), RevisionID: revisionID, Action: action, ActorID: actor.ID, Reason: reason, CreatedAt: now})
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

// NewArticle creates a version-1 article whose first revision is a DRAFT.
func NewArticle(c Content, actor Identity, now time.Time) (*Article, error) {
	if err := actor.Authorize(PermCreate); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	r := Revision{ID: uuid.New(), Number: 1, Status: Draft, Content: c, CreatedBy: actor.ID, CreatedAt: now}
	a := &Article{ID: uuid.New(), Version: 1, CurrentRevisionID: r.ID, CreatedAt: now, UpdatedAt: now, Revisions: []Revision{r}}
	a.audit("create", r.ID, "", actor, now)
	return a, nil
}

// Content snapshots are append-only. Only moderation status may change.
func (a *Article) Edit(c Content, actor Identity, now time.Time) error {
	if err := actor.Authorize(PermEdit); err != nil {
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
	a.audit("edit", next.ID, "", actor, now)
	return nil
}
func (a *Article) Transition(action, reason string, actor Identity, now time.Time) error {
	permission, known := ActionPermission(action)
	if !known {
		return ErrValidation
	}
	if err := actor.Authorize(permission); err != nil {
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
		reason = ""
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
		reason = ""
	case "hide":
		if !a.Visible || a.PublishedRevisionID == nil {
			return ErrState
		}
		reason = strings.TrimSpace(reason)
		if len(reason) > 2000 {
			return ErrValidation
		}
		a.Visible = false
	case "restore":
		// Make the previously published snapshot public again without a new review.
		if a.Visible || a.PublishedRevisionID == nil {
			return ErrState
		}
		a.Visible = true
		reason = ""
	}
	subject := r.ID
	if (action == "hide" || action == "restore") && a.PublishedRevisionID != nil {
		subject = *a.PublishedRevisionID
	}
	a.audit(action, subject, reason, actor, now)
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
