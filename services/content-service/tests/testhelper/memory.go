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
}

func NewMemory() *Memory {
	return &Memory{items: map[uuid.UUID]*domain.Article{}, slugs: map[string]uuid.UUID{}}
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
func Actor(role string) domain.Identity { return domain.Identity{ID: uuid.New(), Role: role} }
