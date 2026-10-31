package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/transport/http/middleware"
	"github.com/omamx/profile-service/internal/usecase"
)

type ProfileHandler struct {
	customerUsecase *usecase.CustomerUsecase
}

func NewProfileHandler(customerUsecase *usecase.CustomerUsecase) *ProfileHandler {
	return &ProfileHandler{
		customerUsecase: customerUsecase,
	}
}

type UpdateProfilePayload struct {
	FullName        string `json:"full_name"`
	PhoneNumber     string `json:"phone_number"`
	Email           string `json:"email"`
	ExpectedVersion int    `json:"version"` // Optimistic locking CAS version
}

// GetProfile handles GET /api/v1/profile
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetCustomerIDFromContext(r.Context())

	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing customer identity")
		return
	}

	profile, err := h.customerUsecase.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			writeJSONError(w, http.StatusNotFound, "customer profile not found")
			return
		}
		writeDomainError(w, err)
		return
	}

	w.Header().Set("ETag", etag(profile.Version))
	writeJSON(w, http.StatusOK, profile)
}

// UpdateProfile handles PUT /api/v1/profile
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetCustomerIDFromContext(r.Context())

	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing customer identity")
		return
	}

	var req UpdateProfilePayload
	if err := decodeBody(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body payload")
		return
	}

	if r.Header.Get("If-Match") != "" {
		version, err := expectedVersion(r)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		req.ExpectedVersion = version
	}
	updated, err := h.customerUsecase.UpdateProfileWithOCC(
		r.Context(),
		userID,
		req.FullName,
		req.PhoneNumber,
		req.Email,
		req.ExpectedVersion,
	)

	if err != nil {
		writeDomainError(w, err)
		return
	}

	w.Header().Set("ETag", etag(updated.Version))
	writeJSON(w, http.StatusOK, updated)
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}
