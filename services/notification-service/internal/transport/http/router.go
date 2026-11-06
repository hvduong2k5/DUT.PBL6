package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omamx/notification-service/internal/domain"
	"github.com/omamx/notification-service/internal/ports"
	"github.com/omamx/notification-service/internal/usecase"
	"io"
	"mime"
	"net/http"
	"sync/atomic"
	"time"
)

type API struct {
	Service usecase.Service
	Auth    ports.Authenticator
	Ready   func(context.Context) error
}
type principalKey struct{}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func failure(w http.ResponseWriter, status int, code string) {
	write(w, status, map[string]any{"code": code, "message": "Request could not be processed", "request_id": uuid.NewString(), "retryable": status == 503})
}
func respondError(w http.ResponseWriter, e error) {
	code, status := "UNAVAILABLE", 503
	for _, v := range []struct {
		e      error
		status int
	}{{domain.ErrInvalid, 422}, {domain.ErrUnauthenticated, 401}, {domain.ErrForbidden, 403}, {domain.ErrConflict, 409}, {domain.ErrNotFound, 404}} {
		if errors.Is(e, v.e) {
			code = v.e.Error()
			status = v.status
			break
		}
	}
	failure(w, status, code)
}
func (a API) Router() http.Handler {
	r := chi.NewRouter()
	var authenticatedRequests atomic.Uint64
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = fmt.Fprintf(w, "# HELP notification_authenticated_requests_total Protected requests authenticated by this process.\n# TYPE notification_authenticated_requests_total counter\nnotification_authenticated_requests_total %d\n", authenticatedRequests.Load())
	})
	r.Get("/health/live", func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]string{"status": "ok"}) })
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if a.Ready == nil || a.Ready(ctx) != nil {
			failure(w, 503, "NOT_READY")
			return
		}
		write(w, 200, map[string]string{"status": "ready"})
	})
	r.Route("/api/v1/internal/notification-requests", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if a.Auth == nil || len(r.Header.Values("Authorization")) != 1 {
					failure(w, 401, "UNAUTHENTICATED")
					return
				}
				p, e := a.Auth.Authenticate(r.Context(), r.Header.Get("Authorization"))
				if e != nil {
					respondError(w, e)
					return
				}
				authenticatedRequests.Add(1)
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, p)))
			})
		})
		r.Post("/", a.accept)
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			p := r.Context().Value(principalKey{}).(ports.Principal)
			result, e := a.Service.Get(r.Context(), p, chi.URLParam(r, "id"))
			if e != nil {
				respondError(w, e)
				return
			}
			write(w, 200, result)
		})
	})
	return r
}
func (a API) accept(w http.ResponseWriter, r *http.Request) {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		failure(w, 415, "UNSUPPORTED_MEDIA_TYPE")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	var c domain.Command
	if err = d.Decode(&c); err != nil {
		decodeError(w, err)
		return
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		if err == nil {
			failure(w, 400, "MALFORMED_JSON")
		} else {
			decodeError(w, err)
		}
		return
	}
	if len(r.Header.Values("Idempotency-Key")) != 1 {
		failure(w, 422, "INVALID_REQUEST")
		return
	}
	p := r.Context().Value(principalKey{}).(ports.Principal)
	result, err := a.Service.Accept(r.Context(), p, r.Header.Get("Idempotency-Key"), c)
	if err != nil {
		respondError(w, err)
		return
	}
	status := 202
	if result.Duplicate {
		status = 200
	}
	w.Header().Set("Location", "/api/v1/internal/notification-requests/"+result.ID)
	write(w, status, result)
}
func decodeError(w http.ResponseWriter, e error) {
	var large *http.MaxBytesError
	if errors.As(e, &large) {
		failure(w, 413, "BODY_TOO_LARGE")
	} else {
		failure(w, 400, "MALFORMED_JSON")
	}
}
