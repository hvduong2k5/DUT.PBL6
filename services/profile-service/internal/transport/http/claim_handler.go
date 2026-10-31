package http

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
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
	customerID, ok := middleware.GetCustomerIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing customer identity")
		return
	}

	var req usecase.ClaimOrderRequest
	if err := decodeBody(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.claimUsecase.ClaimGuestOrder(r.Context(), customerID, req)
	if err != nil {
		if errors.Is(err, domain.ErrOrderAlreadyClaimed) {
			writeJSONError(w, http.StatusConflict, "conflict: this guest order has already been claimed by another customer")
			return
		}
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, res)
}

// GetClaimStatus handles GET /api/v1/profile/guest-claims/{order_id}
func (h *ClaimHandler) GetClaimStatus(w http.ResponseWriter, r *http.Request) {
	customerID, ok := middleware.GetCustomerIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, 401, "unauthorized")
		return
	}
	orderID := chi.URLParam(r, "order_id")
	if orderID == "" {
		writeJSONError(w, http.StatusBadRequest, "order_id is required")
		return
	}

	claim, err := h.claimUsecase.GetOwnedClaim(r.Context(), customerID, orderID)
	if err != nil {
		if errors.Is(err, domain.ErrClaimNotFound) {
			writeJSONError(w, http.StatusNotFound, "no active claim found for this order")
			return
		}
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, claim)
}
