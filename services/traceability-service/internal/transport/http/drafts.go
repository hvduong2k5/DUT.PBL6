package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/application"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/jsonutil"
)

type IdentityVerifier interface {
	Verify(context.Context, string) (domain.Principal, error)
}
type HandlerConfig struct {
	Readiness Readiness
	Logger    *slog.Logger
	Identity  IdentityVerifier
	Drafts    *application.Drafts
	Workflow  *application.Workflow
	Metrics   HTTPMetrics
}

func NewHandler(cfg HandlerConfig) http.Handler {
	mux := healthMux(cfg.Readiness)
	if cfg.Metrics != nil {
		mux.Handle("GET /metrics", cfg.Metrics)
	}
	mux.HandleFunc("GET /api/v1/admin/trace/records", func(w http.ResponseWriter, r *http.Request) {
		p, ok := authenticate(w, r, cfg.Identity)
		if !ok || !permitted(w, p, domain.ReadInternal) {
			return
		}
		query := domain.ListQuery{}
		for name, values := range r.URL.Query() {
			if len(values) != 1 || (name != "page_size" && name != "cursor") {
				respondError(w, domain.ErrInvalidCommand)
				return
			}
		}
		if raw, exists := r.URL.Query()["page_size"]; exists {
			n, err := strconv.Atoi(raw[0])
			if err != nil || n < 1 || n > 100 {
				respondError(w, domain.ErrInvalidCommand)
				return
			}
			query.PageSize = n
		}
		query.Cursor = r.URL.Query().Get("cursor")
		page, err := cfg.Drafts.List(r.Context(), p, query)
		if err != nil {
			respondError(w, err)
			return
		}
		writeJSON(w, 200, page)
	})
	mux.HandleFunc("POST /api/v1/admin/trace/records", func(w http.ResponseWriter, r *http.Request) {
		p, ok := authenticate(w, r, cfg.Identity)
		if !ok {
			return
		}
		if !permitted(w, p, domain.Create) {
			return
		}
		var request domain.CreateDraftRequest
		if !decodeRequest(w, r, &request, "product_id", "sku_code", "source_namespace", "batch_id", "content") {
			return
		}
		result, err := cfg.Drafts.Create(r.Context(), p, metadata(w, r), request)
		respondCommand(w, result, err)
	})
	mux.HandleFunc("GET /api/v1/admin/trace/records/{id}", func(w http.ResponseWriter, r *http.Request) {
		p, ok := authenticate(w, r, cfg.Identity)
		if !ok {
			return
		}
		if !permitted(w, p, domain.ReadInternal) {
			return
		}
		if !domain.ValidUUID(r.PathValue("id")) {
			respondError(w, domain.ErrInvalidCommand)
			return
		}
		profile, err := cfg.Drafts.Get(r.Context(), p, r.PathValue("id"))
		if err != nil {
			respondError(w, err)
			return
		}
		writeJSON(w, 200, profile)
	})
	mux.HandleFunc("PATCH /api/v1/admin/trace/records/{id}/revisions/{revision}", func(w http.ResponseWriter, r *http.Request) {
		p, ok := authenticate(w, r, cfg.Identity)
		if !ok {
			return
		}
		if !permitted(w, p, domain.Edit) {
			return
		}
		revision, err := strconv.Atoi(r.PathValue("revision"))
		if err != nil || revision < 1 || !domain.ValidUUID(r.PathValue("id")) {
			respondError(w, domain.ErrInvalidCommand)
			return
		}
		var request domain.EditDraftRequest
		if !decodeRequest(w, r, &request, "expected_version", "content") {
			return
		}
		if request.ExpectedVersion < 1 {
			respondError(w, domain.ErrInvalidCommand)
			return
		}
		result, err := cfg.Drafts.Edit(r.Context(), p, metadata(w, r), r.PathValue("id"), revision, request)
		respondCommand(w, result, err)
	})
	registerWorkflow(mux, cfg)
	return withMiddleware(mux, cfg.Logger, cfg.Metrics)
}

func authenticate(w http.ResponseWriter, r *http.Request, verifier IdentityVerifier) (domain.Principal, bool) {
	headers := r.Header.Values("Authorization")
	if verifier == nil || len(headers) != 1 {
		respondError(w, domain.ErrUnauthenticated)
		return domain.Principal{}, false
	}
	parts := strings.Fields(headers[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		respondError(w, domain.ErrUnauthenticated)
		return domain.Principal{}, false
	}
	p, err := verifier.Verify(r.Context(), parts[1])
	if err != nil || !domain.ValidUUID(p.ID) {
		respondError(w, domain.ErrUnauthenticated)
		return domain.Principal{}, false
	}
	return p, true
}
func permitted(w http.ResponseWriter, p domain.Principal, permission string) bool {
	if !p.Can(permission) || !p.Can(domain.ReadInternal) {
		respondError(w, domain.ErrForbidden)
		return false
	}
	return true
}
func metadata(w http.ResponseWriter, r *http.Request) domain.CommandMetadata {
	return domain.CommandMetadata{Key: r.Header.Get("X-Idempotency-Key"), RequestID: w.Header().Get("X-Request-Id"), TraceParent: w.Header().Get("Traceparent")}
}

func decodeRequest(w http.ResponseWriter, r *http.Request, target any, required ...string) bool {
	if len(r.Header.Values("X-Idempotency-Key")) != 1 {
		respondError(w, domain.ErrInvalidCommand)
		return false
	}
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || (r.Header.Get("Content-Encoding") != "" && r.Header.Get("Content-Encoding") != "identity") {
		respondError(w, domain.ErrInvalidCommand)
		return false
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 256*1024))
	if err != nil || jsonutil.Decode(data, target) != nil {
		respondError(w, domain.ErrInvalidCommand)
		return false
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil {
		respondError(w, domain.ErrInvalidCommand)
		return false
	}
	for _, name := range required {
		value, exists := fields[name]
		if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			respondError(w, domain.ErrInvalidCommand)
			return false
		}
	}
	if value, exists := fields["reason"]; exists {
		var reason string
		if json.Unmarshal(value, &reason) != nil || strings.TrimSpace(reason) == "" {
			respondError(w, domain.ErrInvalidCommand)
			return false
		}
	}
	return true
}
func respondCommand(w http.ResponseWriter, result domain.CommandResult, err error) {
	if err != nil {
		respondError(w, err)
		return
	}
	if result.Replayed {
		w.Header().Set("X-Idempotency-Replayed", "true")
	}
	writeJSON(w, result.Status, result.Body)
}
func respondError(w http.ResponseWriter, err error) {
	status, code, message := 500, "INTERNAL_ERROR", "Unable to process request"
	var validation *domain.ValidationError
	var gone *domain.GoneError
	switch {
	case errors.Is(err, domain.ErrUnauthenticated):
		status, code, message = 401, "UNAUTHENTICATED", "Verified identity required"
	case errors.Is(err, domain.ErrForbidden):
		status, code, message = 403, "FORBIDDEN", "Permission denied"
	case errors.Is(err, domain.ErrNotFound):
		status, code, message = 404, "NOT_FOUND", "Record not found"
	case errors.As(err, &gone):
		status, code, message = 410, gone.Reason, "Published record is no longer eligible"
	case errors.Is(err, domain.ErrConflict):
		status, code, message = 409, "CONFLICT", "Version, binding or state conflict"
	case errors.Is(err, domain.ErrIdempotencyConflict):
		status, code, message = 409, "IDEMPOTENCY_CONFLICT", "Idempotency key already used for a different request"
	case errors.Is(err, domain.ErrInvalidCommand) || errors.As(err, &validation):
		status, code, message = 400, "INVALID_INPUT", "Invalid request"
	case errors.Is(err, domain.ErrUnavailable) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
		status, code, message = 503, "DEPENDENCY_UNAVAILABLE", "Required dependency unavailable"
	}
	writeJSON(w, status, map[string]string{"error_code": code, "user_message": message, "request_id": w.Header().Get("X-Request-Id")})
}
