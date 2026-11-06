package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/omamx/promotion-service/internal/repository/postgres"
)

type LoyaltyHandler struct {
	repository *postgres.LoyaltyRepository
}

type loyaltyEntryResponse struct {
	ID               string `json:"id"`
	EntryType        string `json:"entry_type"`
	PointsDelta      int64  `json:"points_delta"`
	BasisAmountVND   *int64 `json:"basis_amount_vnd,omitempty"`
	SourceType       string `json:"source_type"`
	SourceID         string `json:"source_id"`
	PolicyVersion    string `json:"policy_version"`
	ReferenceEntryID string `json:"reference_entry_id,omitempty"`
	CreatedAt        string `json:"created_at"`
}

func RegisterLoyaltyRoutes(router chi.Router, repository *postgres.LoyaltyRepository, internalToken string) {
	handler := &LoyaltyHandler{repository: repository}
	router.With(requireInternalGateway(internalToken)).Get("/internal/v1/loyalty/customers/{customerID}/ledger", handler.GetLedger)
}

func (h *LoyaltyHandler) GetLedger(w http.ResponseWriter, r *http.Request) {
	customerID := strings.TrimSpace(chi.URLParam(r, "customerID"))
	if customerID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_CUSTOMER", "customer id is required")
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, "INVALID_PAGINATION", "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			writeError(w, http.StatusBadRequest, "INVALID_PAGINATION", "offset must be a non-negative integer")
			return
		}
		offset = parsed
	}
	page, err := h.repository.GetLedgerPage(r.Context(), customerID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read loyalty ledger")
		return
	}
	responseEntries := make([]loyaltyEntryResponse, 0, len(page.Entries))
	for _, entry := range page.Entries {
		responseEntries = append(responseEntries, loyaltyEntryResponse{
			ID: entry.ID, EntryType: entry.EntryType, PointsDelta: entry.PointsDelta,
			BasisAmountVND: entry.BasisAmountVND, SourceType: entry.SourceType, SourceID: entry.SourceID,
			PolicyVersion: entry.PolicyVersion, ReferenceEntryID: entry.ReferenceEntryID,
			CreatedAt: entry.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"customer_id":    customerID,
		"current_points": page.Balance,
		"entries":        responseEntries,
		"limit":          limit,
		"offset":         offset,
	})
}

func requireInternalGateway(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			supplied := r.Header.Get("X-Internal-Token")
			if token == "" || len(supplied) != len(token) || subtle.ConstantTimeCompare([]byte(supplied), []byte(token)) != 1 {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "internal gateway authentication is required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
