// Package testhelper provides fixtures only; the executable uses PostgreSQL.
package testhelper

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
	"sort"
	"strings"
	"sync"
	"time"
)

type Memory struct {
	mu    sync.Mutex
	items map[uuid.UUID]*domain.Article
	slugs map[string]uuid.UUID
	keys  map[string]*domain.IdempotencyRecord
}

var _ domain.Repository = (*Memory)(nil)
var _ domain.IdempotencyStore = (*Memory)(nil)

func NewMemory() *Memory {
	return &Memory{items: map[uuid.UUID]*domain.Article{}, slugs: map[string]uuid.UUID{}, keys: map[string]*domain.IdempotencyRecord{}}
}

func keyID(principal uuid.UUID, key string) string { return principal.String() + "/" + key }
func (m *Memory) Begin(_ context.Context, principal uuid.UUID, key, hash string) (domain.IdempotencyRecord, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rec, ok := m.keys[keyID(principal, key)]; ok {
		return *rec, false, nil
	}
	m.keys[keyID(principal, key)] = &domain.IdempotencyRecord{RequestHash: hash}
	return domain.IdempotencyRecord{RequestHash: hash}, true, nil
}
func (m *Memory) Complete(_ context.Context, principal uuid.UUID, key string, status int, headers map[string]string, body []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec := m.keys[keyID(principal, key)]
	rec.Completed, rec.Status, rec.Headers, rec.Body = true, status, headers, append([]byte(nil), body...)
	return nil
}
func (m *Memory) Abandon(_ context.Context, principal uuid.UUID, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rec := m.keys[keyID(principal, key)]; rec != nil && !rec.Completed {
		delete(m.keys, keyID(principal, key))
	}
	return nil
}
func clone(a *domain.Article) *domain.Article {
	raw, _ := json.Marshal(a)
	var out domain.Article
	_ = json.Unmarshal(raw, &out)
	return &out
}
func (m *Memory) reserve(a *domain.Article) error {
	for _, r := range a.Revisions {
		if owner, ok := m.slugs[r.Content.Slug]; ok && owner != a.ID {
			return domain.ErrSlug
		}
	}
	for _, r := range a.Revisions {
		if _, ok := m.slugs[r.Content.Slug]; !ok {
			a.Slugs = append(a.Slugs, domain.SlugHistory{Slug: r.Content.Slug, CreatedAt: r.CreatedAt})
		}
		m.slugs[r.Content.Slug] = a.ID
	}
	return nil
}
func (m *Memory) Create(_ context.Context, a *domain.Article) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.reserve(a); err != nil {
		return err
	}
	m.items[a.ID] = clone(a)
	return nil
}
func (m *Memory) Get(_ context.Context, id uuid.UUID) (*domain.Article, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a := m.items[id]
	if a == nil {
		return nil, domain.ErrNotFound
	}
	return clone(a), nil
}
func (m *Memory) Mutate(_ context.Context, id uuid.UUID, version int64, apply func(*domain.Article) error) (*domain.Article, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a := m.items[id]
	if a == nil {
		return nil, domain.ErrNotFound
	}
	if a.Version != version {
		return nil, domain.ErrConflict
	}
	a = clone(a)
	if err := apply(a); err != nil {
		return nil, err
	}
	if err := m.reserve(a); err != nil {
		return nil, err
	}
	a.Version++
	a.UpdatedAt = time.Now().UTC()
	m.items[id] = clone(a)
	return a, nil
}
func (m *Memory) GetPublished(_ context.Context, slug string) (*domain.Article, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.items {
		if a.Visible && a.PublishedRevisionID != nil && a.Revision(*a.PublishedRevisionID).Content.Slug == slug {
			return clone(a), nil
		}
	}
	return nil, domain.ErrNotFound
}
func (m *Memory) List(_ context.Context, f domain.Filter) ([]domain.Article, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []domain.Article{}
	for _, a := range m.items {
		r := a.Current()
		if f.Public {
			if !a.Visible || a.PublishedRevisionID == nil {
				continue
			}
			r = a.Revision(*a.PublishedRevisionID)
		}
		if f.Status != "" && r.Status != f.Status {
			continue
		}
		if f.Category != "" && r.Content.Category != f.Category {
			continue
		}
		if f.Query != "" && !strings.Contains(strings.ToLower(r.Content.Title+" "+r.Content.Body), strings.ToLower(f.Query)) {
			continue
		}
		out = append(out, *clone(a))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID.String() > out[j].ID.String()
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	total := int64(len(out))
	start := (f.Page - 1) * f.Limit
	if start >= len(out) {
		return []domain.Article{}, total, nil
	}
	end := start + f.Limit
	if end > len(out) {
		end = len(out)
	}
	return out[start:end], total, nil
}
func Content(slug string) domain.Content {
	return domain.Content{Title: "Hue heritage", Slug: slug, Body: "Original story", Category: "heritage", SEO: domain.SEO{Title: "Hue", Keywords: []string{"Hue"}}, Media: []domain.MediaReference{}}
}

// Actor returns a verified test identity holding GLOBAL grants for the given
// permissions. Test identities never exist in the production executable.
func Actor(perms ...domain.Permission) domain.Identity {
	id := domain.Identity{ID: uuid.New()}
	for _, p := range perms {
		id.Grants = append(id.Grants, domain.Grant{Permission: p, ScopeType: domain.ScopeGlobal})
	}
	return id
}

// Editor can draft and submit but not review or publish.
func Editor() domain.Identity {
	return Actor(domain.PermView, domain.PermCreate, domain.PermEdit, domain.PermSubmit, domain.PermHide)
}

// Approver can review, publish and hide.
func Approver() domain.Identity {
	return Actor(domain.PermView, domain.PermApprove, domain.PermReject, domain.PermPublish, domain.PermHide)
}

// Full holds every content permission, the way a content administrator is granted them.
func Full() domain.Identity {
	return Actor(domain.PermView, domain.PermCreate, domain.PermEdit, domain.PermSubmit, domain.PermApprove, domain.PermReject, domain.PermPublish, domain.PermHide)
}

// Customer is a verified principal that holds no content grants.
func Customer() domain.Identity { return Actor() }
