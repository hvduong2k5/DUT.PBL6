package handlers

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
	"time"
)

type SEODTO struct {
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	Keywords     []string        `json:"keywords"`
	CanonicalURL string          `json:"canonical_url"`
	Schema       json.RawMessage `json:"schema,omitempty"`
}
type MediaDTO struct {
	ID   uuid.UUID `json:"id"`
	URL  string    `json:"url"`
	Alt  string    `json:"alt"`
	Kind string    `json:"kind"`
}
type ContentDTO struct {
	Title    string     `json:"title"`
	Slug     string     `json:"slug"`
	Body     string     `json:"body,omitempty"`
	Category string     `json:"category"`
	SEO      SEODTO     `json:"seo"`
	Media    []MediaDTO `json:"media"`
}

func (s SEODTO) domain() domain.SEO {
	return domain.SEO{Title: s.Title, Description: s.Description, Keywords: s.Keywords, CanonicalURL: s.CanonicalURL, Schema: s.Schema}
}
func mediaDomain(items []MediaDTO) []domain.MediaReference {
	out := make([]domain.MediaReference, 0, len(items))
	for _, m := range items {
		out = append(out, domain.MediaReference{ID: m.ID, URL: m.URL, Alt: m.Alt, Kind: m.Kind})
	}
	return out
}
func (c ContentDTO) domain() domain.Content {
	return domain.Content{Title: c.Title, Slug: c.Slug, Body: c.Body, Category: c.Category, SEO: c.SEO.domain(), Media: mediaDomain(c.Media)}
}
func contentDTO(c domain.Content) ContentDTO {
	keywords := c.SEO.Keywords
	if keywords == nil {
		keywords = []string{}
	}
	out := ContentDTO{Title: c.Title, Slug: c.Slug, Body: c.Body, Category: c.Category, SEO: SEODTO{Title: c.SEO.Title, Description: c.SEO.Description, Keywords: keywords, CanonicalURL: c.SEO.CanonicalURL, Schema: c.SEO.Schema}, Media: []MediaDTO{}}
	for _, m := range c.Media {
		out.Media = append(out.Media, MediaDTO{ID: m.ID, URL: m.URL, Alt: m.Alt, Kind: m.Kind})
	}
	return out
}

type PatchRequest struct {
	Version  int64       `json:"version"`
	Title    *string     `json:"title"`
	Slug     *string     `json:"slug"`
	Body     *string     `json:"body"`
	Category *string     `json:"category"`
	SEO      *SEODTO     `json:"seo"`
	Media    *[]MediaDTO `json:"media"`
}
type ActionRequest struct {
	Version int64  `json:"version"`
	Reason  string `json:"reason,omitempty"`
}
type RevisionResponse struct {
	ID        uuid.UUID     `json:"id"`
	Number    int           `json:"number"`
	Status    domain.Status `json:"status"`
	Content   ContentDTO    `json:"content"`
	CreatedBy uuid.UUID     `json:"created_by"`
	CreatedAt time.Time     `json:"created_at"`
}

func revisionDTO(r *domain.Revision) *RevisionResponse {
	if r == nil {
		return nil
	}
	return &RevisionResponse{ID: r.ID, Number: r.Number, Status: r.Status, Content: contentDTO(r.Content), CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt}
}

type DecisionResponse struct {
	ID         uuid.UUID     `json:"id"`
	RevisionID uuid.UUID     `json:"revision_id"`
	Status     domain.Status `json:"status"`
	Reason     string        `json:"reason"`
	ActorID    uuid.UUID     `json:"actor_id"`
	CreatedAt  time.Time     `json:"created_at"`
}
type SlugResponse struct {
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}
type AdminResponse struct {
	ID                  uuid.UUID          `json:"id"`
	Version             int64              `json:"version"`
	Visible             bool               `json:"visible"`
	Current             *RevisionResponse  `json:"current_revision"`
	PublishedRevisionID *uuid.UUID         `json:"published_revision_id"`
	Revisions           []RevisionResponse `json:"revisions,omitempty"`
	Decisions           []DecisionResponse `json:"decisions,omitempty"`
	Slugs               []SlugResponse     `json:"slug_history,omitempty"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
}

func adminDTO(a *domain.Article, history bool) AdminResponse {
	out := AdminResponse{ID: a.ID, Version: a.Version, Visible: a.Visible, Current: revisionDTO(a.Current()), PublishedRevisionID: a.PublishedRevisionID, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt}
	if !history && out.Current != nil {
		out.Current.Content.Body = ""
	}
	if history {
		for _, r := range a.Revisions {
			out.Revisions = append(out.Revisions, *revisionDTO(&r))
		}
		for _, d := range a.Decisions {
			out.Decisions = append(out.Decisions, DecisionResponse{ID: d.ID, RevisionID: d.RevisionID, Status: d.Status, Reason: d.Reason, ActorID: d.ActorID, CreatedAt: d.CreatedAt})
		}
		for _, h := range a.Slugs {
			out.Slugs = append(out.Slugs, SlugResponse{Slug: h.Slug, CreatedAt: h.CreatedAt})
		}
	}
	return out
}

type PublicResponse struct {
	ID uuid.UUID `json:"id"`
	ContentDTO
	CreatedAt time.Time `json:"created_at"`
}

func publicDTO(a *domain.Article) PublicResponse {
	r := a.Revision(*a.PublishedRevisionID)
	return PublicResponse{ID: a.ID, ContentDTO: contentDTO(r.Content), CreatedAt: r.CreatedAt}
}
