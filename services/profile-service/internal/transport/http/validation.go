package http

import (
	"encoding/json"
	"errors"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func decodeBody(w http.ResponseWriter, r *http.Request, target any) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("request must contain exactly one JSON object")
	}
	return nil
}
func etag(version int) string { return `"` + strconv.Itoa(version) + `"` }

var errPreconditionRequired = errors.New("If-Match is required")

func expectedVersion(r *http.Request) (int, error) {
	value := r.Header.Get("If-Match")
	if value == "" {
		return 0, errPreconditionRequired
	}
	if len(value) < 3 || value[0] != '"' || value[len(value)-1] != '"' {
		return 0, &domain.ValidationError{Field: "If-Match", Message: "must be a quoted positive integer ETag"}
	}
	n, err := strconv.Atoi(strings.Trim(value, `"`))
	if err != nil || n < 1 || value != etag(n) {
		return 0, &domain.ValidationError{Field: "If-Match", Message: "invalid ETag"}
	}
	return n, nil
}
func writeDomainError(w http.ResponseWriter, err error) {
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		writeJSON(w, 400, map[string]any{"code": "VALIDATION_ERROR", "error": validation.Message, "errors": []map[string]string{{"field": validation.Field, "message": validation.Message}}})
	case errors.Is(err, errPreconditionRequired):
		writeJSON(w, 428, map[string]string{"code": "PRECONDITION_REQUIRED", "error": "If-Match is required"})
	case errors.Is(err, domain.ErrOptimisticLockConflict):
		writeJSON(w, 409, map[string]string{"code": "VERSION_CONFLICT", "error": "refresh the resource and retry"})
	case errors.Is(err, domain.ErrAddressNotFound), errors.Is(err, domain.ErrAddressMatchNotFound), errors.Is(err, domain.ErrCustomerNotFound), errors.Is(err, domain.ErrClaimNotFound), errors.Is(err, domain.ErrEmployeeNotFound):
		writeJSON(w, 404, map[string]string{"code": "NOT_FOUND", "error": "resource not found"})
	case errors.Is(err, domain.ErrClaimVerificationUnavailable):
		writeJSON(w, 503, map[string]string{"code": "CLAIM_VERIFICATION_UNAVAILABLE", "error": "guest order claim requires a configured ownership verifier"})
	case errors.Is(err, domain.ErrVaultUnavailable):
		writeJSON(w, 503, map[string]string{"code": "DEPENDENCY_UNAVAILABLE", "error": "PII service unavailable"})
	default:
		log.Error().Err(err).Msg("profile request failed")
		writeJSON(w, 500, map[string]string{"code": "INTERNAL_ERROR", "error": "internal server error"})
	}
}
