package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/transport/http/middleware"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/usecase"
	"github.com/rs/zerolog"
)

type Handler struct {
	service *usecase.Service
	log     zerolog.Logger
}

func New(s *usecase.Service, log zerolog.Logger) *Handler { return &Handler{service: s, log: log} }
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func (h *Handler) DomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthenticated):
		Error(w, 401, "UNAUTHENTICATED", "verified identity required")
	case errors.Is(err, domain.ErrForbidden):
		Error(w, 403, "FORBIDDEN", "permission denied")
	case errors.Is(err, domain.ErrNotFound):
		Error(w, 404, "NOT_FOUND", "article not found")
	case errors.Is(err, domain.ErrConflict):
		Error(w, 409, "VERSION_CONFLICT", "reload the article and retry with its current version")
	case errors.Is(err, domain.ErrSlug):
		Error(w, 409, "DUPLICATE_SLUG", "slug already reserved")
	case errors.Is(err, domain.ErrState):
		Error(w, 422, "INVALID_STATE", "invalid status transition")
	case errors.Is(err, domain.ErrValidation):
		Error(w, 422, "VALIDATION_ERROR", "invalid article input")
	default:
		h.log.Error().Err(err).Msg("article operation failed")
		Error(w, 500, "INTERNAL_ERROR", "internal server error")
	}
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
		Error(w, 400, "BAD_REQUEST", "Content-Type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		Error(w, 400, "BAD_REQUEST", "invalid JSON body")
		return false
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		Error(w, 400, "BAD_REQUEST", "exactly one JSON object is required")
		return false
	}
	return true
}
func id(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	v, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil || v == uuid.Nil {
		Error(w, 400, "BAD_REQUEST", "invalid article ID")
		return uuid.Nil, false
	}
	return v, true
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req ContentDTO
	if !decode(w, r, &req) {
		return
	}
	a, err := h.service.Create(r.Context(), req.domain(), middleware.Identity(r.Context()))
	if err != nil {
		h.DomainError(w, err)
		return
	}
	w.Header().Set("Location", "/api/v1/admin/articles/"+a.ID.String())
	respondAdmin(w, 201, a)
}
func respondAdmin(w http.ResponseWriter, status int, a *domain.Article) {
	w.Header().Set("ETag", `"`+strconv.FormatInt(a.Version, 10)+`"`)
	JSON(w, status, adminDTO(a, true))
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	v, ok := id(w, r)
	if !ok {
		return
	}
	a, err := h.service.Get(r.Context(), v, middleware.Identity(r.Context()))
	if err != nil {
		h.DomainError(w, err)
		return
	}
	respondAdmin(w, 200, a)
}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	v, ok := id(w, r)
	if !ok {
		return
	}
	var req PatchRequest
	if !decode(w, r, &req) {
		return
	}
	p := usecase.Patch{Title: req.Title, Slug: req.Slug, Body: req.Body, Category: req.Category}
	if req.SEO != nil {
		s := req.SEO.domain()
		p.SEO = &s
	}
	if req.Media != nil {
		m := mediaDomain(*req.Media)
		p.Media = &m
	}
	a, err := h.service.Update(r.Context(), v, req.Version, p, middleware.Identity(r.Context()))
	if err != nil {
		h.DomainError(w, err)
		return
	}
	respondAdmin(w, 200, a)
}
func (h *Handler) Action(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, ok := id(w, r)
		if !ok {
			return
		}
		var req ActionRequest
		if !decode(w, r, &req) {
			return
		}
		a, err := h.service.Transition(r.Context(), v, req.Version, action, req.Reason, middleware.Identity(r.Context()))
		if err != nil {
			if action == "publish" && errors.Is(err, domain.ErrState) {
				Error(w, http.StatusConflict, "INVALID_STATE", "invalid status transition")
			} else {
				h.DomainError(w, err)
			}
			return
		}
		respondAdmin(w, 200, a)
	}
}
func (h *Handler) PublicDetail(w http.ResponseWriter, r *http.Request) {
	a, err := h.service.GetPublished(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		h.DomainError(w, err)
		return
	}
	JSON(w, 200, publicDTO(a))
}
func (h *Handler) List(public bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := domain.Filter{Page: 1, Limit: 20, Public: public, Query: r.URL.Query().Get("q"), Category: r.URL.Query().Get("category"), Status: domain.Status(r.URL.Query().Get("status"))}
		for name, target := range map[string]*int{"page": &f.Page, "limit": &f.Limit} {
			if values, ok := r.URL.Query()[name]; ok {
				if len(values) != 1 {
					Error(w, 400, "BAD_REQUEST", "invalid pagination")
					return
				}
				v, e := strconv.Atoi(values[0])
				if e != nil {
					Error(w, 400, "BAD_REQUEST", "invalid pagination")
					return
				}
				*target = v
			}
		}
		items, total, err := h.service.List(r.Context(), f, middleware.Identity(r.Context()))
		if err != nil {
			h.DomainError(w, err)
			return
		}
		if public {
			out := make([]PublicResponse, 0, len(items))
			for n := range items {
				item := publicDTO(&items[n])
				item.Body = ""
				out = append(out, item)
			}
			JSON(w, 200, map[string]any{"items": out, "total": total, "page": f.Page, "limit": f.Limit})
		} else {
			out := make([]AdminResponse, 0, len(items))
			for n := range items {
				out = append(out, adminDTO(&items[n], false))
			}
			JSON(w, 200, map[string]any{"items": out, "total": total, "page": f.Page, "limit": f.Limit})
		}
	}
}
