package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/omamx/care-service/internal/domain"
)

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string, details map[string]any) {
	writeJSON(w, status, errorEnvelope{Error: apiError{
		Code: code, Message: message, RequestID: RequestID(r.Context()), Details: details,
	}})
}

func writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidArgument), errors.Is(err, domain.ErrMediaPolicy):
		writeError(w, r, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), nil)
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, r, http.StatusForbidden, "FORBIDDEN", "The operation is not permitted", nil)
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "NOT_FOUND", "The requested resource was not found", nil)
	case errors.Is(err, domain.ErrIdempotencyConflict):
		writeError(w, r, http.StatusConflict, "IDEMPOTENCY_CONFLICT", err.Error(), nil)
	case errors.Is(err, domain.ErrMediaNotReady):
		writeError(w, r, http.StatusConflict, "MEDIA_NOT_READY", err.Error(), nil)
	case errors.Is(err, domain.ErrConflict):
		writeError(w, r, http.StatusConflict, "CONFLICT", err.Error(), nil)
	case errors.Is(err, domain.ErrDependencyUnavailable):
		writeError(w, r, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "A required dependency is unavailable", nil)
	default:
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", nil)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain exactly one JSON object")
		}
		return err
	}
	return nil
}
