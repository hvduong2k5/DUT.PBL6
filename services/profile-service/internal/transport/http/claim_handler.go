package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/transport/http/middleware"
	"github.com/omamx/profile-service/internal/usecase"
)

type ClaimHandler struct {
	claimUsecase *usecase.ClaimUsecase
}

func NewClaimHandler(claimUsecase *usecase.ClaimUsecase) *ClaimHandler {
	return &ClaimHandler{
		claimUsecase: claimUsecase,
	}
}

// CreateClaim handles POST /api/v1/profile/guest-claims
func (h *ClaimHandler) CreateClaim(w http.ResponseWriter, r *http.Request) {
	customerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		if paramID := r.URL.Query().Get("customer_id"); paramID != "" {
			if parsed, err := uuid.Parse(paramID); err == nil {
				customerID = parsed
				ok = true
			}
		}
	}
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing customer identity")
		return
	}

	var req usecase.ClaimOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.claimUsecase.ClaimGuestOrder(r.Context(), customerID, req)
	if err != nil {
		if errors.Is(err, domain.ErrOrderAlreadyClaimed) {
			writeJSONError(w, http.StatusConflict, "conflict: this guest order has already been claimed by another customer")
			return
		}
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, res)
}

// GetClaimStatus handles GET /api/v1/profile/guest-claims/{order_id}
func (h *ClaimHandler) GetClaimStatus(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "order_id")
	if orderID == "" {
		writeJSONError(w, http.StatusBadRequest, "order_id is required")
		return
	}

	claim, err := h.claimUsecase.GetActiveClaim(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, domain.ErrClaimNotFound) {
			writeJSONError(w, http.StatusNotFound, "no active claim found for this order")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, claim)
}
