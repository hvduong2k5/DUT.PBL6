package usecase

import (
	"context"
	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
	"time"
)

type Service struct{ repo domain.Repository }

func New(repo domain.Repository) *Service { return &Service{repo: repo} }
func (s *Service) Create(ctx context.Context, c domain.Content, actor domain.Identity) (*domain.Article, error) {
	if err := actor.Authorize(false); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	r := domain.Revision{ID: uuid.New(), Number: 1, Status: domain.Draft, Content: c, CreatedBy: actor.ID, CreatedAt: now}
	a := &domain.Article{ID: uuid.New(), Version: 1, CurrentRevisionID: r.ID, CreatedAt: now, UpdatedAt: now, Revisions: []domain.Revision{r}}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

type Patch struct {
	Title    *string
	Slug     *string
	Body     *string
	Category *string
	SEO      *domain.SEO
	Media    *[]domain.MediaReference
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, version int64, p Patch, actor domain.Identity) (*domain.Article, error) {
	if err := actor.Authorize(false); err != nil {
		return nil, err
	}
	if version < 1 || (p.Title == nil && p.Slug == nil && p.Body == nil && p.Category == nil && p.SEO == nil && p.Media == nil) {
		return nil, domain.ErrValidation
	}
	return s.repo.Mutate(ctx, id, version, func(a *domain.Article) error {
		c := a.Current().Content
		if p.Title != nil {
			c.Title = *p.Title
		}
		if p.Slug != nil {
			c.Slug = *p.Slug
		}
		if p.Body != nil {
			c.Body = *p.Body
		}
		if p.Category != nil {
			c.Category = *p.Category
		}
		if p.SEO != nil {
			c.SEO = *p.SEO
		}
		if p.Media != nil {
			c.Media = *p.Media
		}
		return a.Edit(c, actor, time.Now().UTC())
	})
}
func (s *Service) Transition(ctx context.Context, id uuid.UUID, version int64, action, reason string, actor domain.Identity) (*domain.Article, error) {
	if err := actor.Authorize(action == "approve" || action == "reject" || action == "publish"); err != nil {
		return nil, err
	}
	if version < 1 {
		return nil, domain.ErrValidation
	}
	return s.repo.Mutate(ctx, id, version, func(a *domain.Article) error { return a.Transition(action, reason, actor, time.Now().UTC()) })
}
func (s *Service) Get(ctx context.Context, id uuid.UUID, actor domain.Identity) (*domain.Article, error) {
	if err := actor.Authorize(false); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, id)
}
func (s *Service) GetPublished(ctx context.Context, slug string) (*domain.Article, error) {
	return s.repo.GetPublished(ctx, slug)
}
func (s *Service) List(ctx context.Context, f domain.Filter, actor domain.Identity) ([]domain.Article, int64, error) {
	if !f.Public {
		if err := actor.Authorize(false); err != nil {
			return nil, 0, err
		}
	}
	if err := f.Validate(); err != nil {
		return nil, 0, err
	}
	return s.repo.List(ctx, f)
}
