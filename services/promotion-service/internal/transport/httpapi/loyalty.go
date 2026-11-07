package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
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

type redemptionHoldRequest struct {
	CustomerID string `json:"customer_id"`
	Points     int64  `json:"points"`
}

type redemptionFinalizeRequest struct {
	ExpectedVersion int64 `json:"expected_version"`
}

type redemptionReleaseRequest struct {
	ExpectedHoldID  string `json:"expected_hold_id"`
	ExpectedVersion int64  `json:"expected_version"`
}

type redemptionHoldResponse struct {
	OrderID         string `json:"order_id"`
	HoldID          string `json:"hold_id"`
	CustomerID      string `json:"customer_id,omitempty"`
	Points          int64  `json:"points"`
	DiscountAmount  int64  `json:"discount_amount_vnd"`
	State           string `json:"state"`
	ResourceVersion int64  `json:"resource_version"`
	ExpiresAt       string `json:"expires_at"`
	PolicyVersion   string `json:"policy_version"`
	CommandResult   string `json:"command_result"`
	Tombstone       bool   `json:"cancellation_tombstone"`
	Replay          bool   `json:"replay,omitempty"`
}

type loyaltyPolicyResponse struct {
	PolicyVersion       string `json:"policy_version"`
	EarnRateBasisPoints int64  `json:"earn_rate_basis_points"`
	PointsToVNDRate     int64  `json:"points_to_vnd_rate"`
	BaseTierCode        string `json:"base_tier_code"`
	RoundingMode        string `json:"rounding_mode"`
	HoldTTLMinutes      int64  `json:"hold_ttl_minutes"`
	PointsExpireDays    *int64 `json:"points_expire_days,omitempty"`
}

func RegisterLoyaltyRoutes(router chi.Router, repository *postgres.LoyaltyRepository, internalToken, orderToken, refundToken string) {
	handler := &LoyaltyHandler{repository: repository}
	auth := requireInternalGateway(internalToken)
	router.With(requireAnyInternalToken(internalToken, orderToken, refundToken)).Get("/internal/v1/loyalty/policy/active", handler.GetActivePolicy)
	router.With(auth).Get("/internal/v1/loyalty/customers/{customerID}/points", handler.GetPoints)
	router.With(auth).Get("/internal/v1/loyalty/customers/{customerID}/ledger", handler.GetLedger)
	if orderToken != "" {
		router.With(requireInternalGateway(orderToken)).Post("/internal/v1/loyalty/order-completions/{orderID}/earn", handler.RecordPurchaseEarn)
	}
	if refundToken != "" {
		router.With(requireInternalGateway(refundToken)).Post("/internal/v1/loyalty/refund-decisions/{refundID}/conversion", handler.RecordRefundConversion)
	}
	if orderToken != "" {
		orderAuth := requireInternalGateway(orderToken)
		router.With(orderAuth).Post("/internal/v1/loyalty/orders/{orderID}/redemption-holds", handler.LockRedemptionPoints)
		router.With(orderAuth).Get("/internal/v1/loyalty/orders/{orderID}/redemption-holds", handler.GetRedemptionByOrder)
		router.With(orderAuth).Post("/internal/v1/loyalty/orders/{orderID}/redemption-holds/{holdID}/finalize", handler.FinalizeRedemptionHold)
		router.With(orderAuth).Post("/internal/v1/loyalty/orders/{orderID}/redemption-holds/release", handler.ReleaseRedemptionByOrder)
	}
}

type purchaseEarnRequest struct {
	CustomerID           string `json:"customer_id"`
	EligibleAmountVND    int64  `json:"eligible_amount_vnd"`
	CompletionDecisionID string `json:"completion_decision_id"`
	PolicyVersion        string `json:"policy_version"`
}

type refundConversionRequest struct {
	CustomerID         string `json:"customer_id"`
	RefundAmountVND    int64  `json:"refund_amount_vnd"`
	RefundDecisionID   string `json:"refund_decision_id"`
	ConsentReferenceID string `json:"consent_reference_id"`
	PolicyVersion      string `json:"policy_version"`
}

func (h *LoyaltyHandler) RecordPurchaseEarn(w http.ResponseWriter, r *http.Request) {
	var request purchaseEarnRequest
	if !decodeLoyaltyCommand(w, r, &request) {
		return
	}
	entry, err := h.repository.RecordPurchaseEarn(r.Context(), request.CustomerID, chi.URLParam(r, "orderID"), request.CompletionDecisionID, request.PolicyVersion,
		r.Header.Get("Idempotency-Key"), request.EligibleAmountVND)
	writeLoyaltyCommandResult(w, entry, err)
}

func (h *LoyaltyHandler) RecordRefundConversion(w http.ResponseWriter, r *http.Request) {
	var request refundConversionRequest
	if !decodeLoyaltyCommand(w, r, &request) {
		return
	}
	entry, err := h.repository.RecordRefundConversion(r.Context(), request.CustomerID, chi.URLParam(r, "refundID"), request.RefundDecisionID,
		request.ConsentReferenceID, request.PolicyVersion, r.Header.Get("Idempotency-Key"), request.RefundAmountVND)
	writeLoyaltyCommandResult(w, entry, err)
}

func decodeLoyaltyCommand(w http.ResponseWriter, r *http.Request, target any) bool {
	if len(r.Header.Get("Idempotency-Key")) < 16 || len(r.Header.Get("Idempotency-Key")) > 128 {
		writeError(w, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY", "Idempotency-Key must contain 16 to 128 visible ASCII characters")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid loyalty command JSON")
		return false
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "request body must contain one JSON object")
		return false
	}
	return true
}

func writeLoyaltyCommandResult(w http.ResponseWriter, entry postgres.LoyaltyEntry, err error) {
	if err != nil {
		switch {
		case errors.Is(err, postgres.ErrLoyaltyIdempotencyConflict), errors.Is(err, postgres.ErrLoyaltySourceConflict), errors.Is(err, postgres.ErrLoyaltyInsufficientBalance):
			writeError(w, http.StatusConflict, "LOYALTY_CONFLICT", "loyalty command conflicts with the current ledger state")
		case errors.Is(err, postgres.ErrNoLoyaltyPointsEarned):
			writeError(w, http.StatusUnprocessableEntity, "NO_POINTS_EARNED", "the active policy produces no points for this amount")
		default:
			writeError(w, http.StatusBadRequest, "INVALID_LOYALTY_COMMAND", "loyalty command is invalid or its policy is unavailable")
		}
		return
	}
	status := http.StatusCreated
	if entry.Replay {
		status = http.StatusOK
	}
	writeJSON(w, status, loyaltyEntryResponse{
		ID: entry.ID, EntryType: entry.EntryType, PointsDelta: entry.PointsDelta, BasisAmountVND: entry.BasisAmountVND,
		SourceType: entry.SourceType, SourceID: entry.SourceID, PolicyVersion: entry.PolicyVersion,
		ReferenceEntryID: entry.ReferenceEntryID, CreatedAt: entry.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	})
}

func (h *LoyaltyHandler) LockRedemptionPoints(w http.ResponseWriter, r *http.Request) {
	var request redemptionHoldRequest
	if !decodeLoyaltyCommand(w, r, &request) {
		return
	}
	hold, err := h.repository.LockRedemptionPoints(r.Context(), chi.URLParam(r, "orderID"), request.CustomerID, r.Header.Get("Idempotency-Key"), request.Points)
	writeRedemptionResult(w, hold, err)
}

func (h *LoyaltyHandler) FinalizeRedemptionHold(w http.ResponseWriter, r *http.Request) {
	var request redemptionFinalizeRequest
	if !decodeLoyaltyCommand(w, r, &request) {
		return
	}
	hold, err := h.repository.FinalizeRedemptionHold(r.Context(), chi.URLParam(r, "orderID"), chi.URLParam(r, "holdID"), r.Header.Get("Idempotency-Key"), request.ExpectedVersion)
	writeRedemptionResult(w, hold, err)
}

func (h *LoyaltyHandler) ReleaseRedemptionByOrder(w http.ResponseWriter, r *http.Request) {
	var request redemptionReleaseRequest
	if !decodeLoyaltyCommand(w, r, &request) {
		return
	}
	hold, err := h.repository.ReleaseRedemptionByOrder(r.Context(), chi.URLParam(r, "orderID"), request.ExpectedHoldID, r.Header.Get("Idempotency-Key"), request.ExpectedVersion)
	writeRedemptionResult(w, hold, err)
}

func (h *LoyaltyHandler) GetRedemptionByOrder(w http.ResponseWriter, r *http.Request) {
	hold, err := h.repository.GetRedemptionByOrder(r.Context(), chi.URLParam(r, "orderID"), r.URL.Query().Get("hold_id"))
	if err != nil {
		writeRedemptionResult(w, postgres.LoyaltyHold{}, err)
		return
	}
	writeJSON(w, http.StatusOK, redemptionHoldFrom(hold))
}

func writeRedemptionResult(w http.ResponseWriter, hold postgres.LoyaltyHold, err error) {
	if err != nil {
		switch {
		case errors.Is(err, postgres.ErrLoyaltyHoldNotFound):
			writeError(w, http.StatusNotFound, "LOYALTY_HOLD_NOT_FOUND", "redemption hold was not found")
		case errors.Is(err, postgres.ErrLoyaltyHoldConflict), errors.Is(err, postgres.ErrLoyaltyIdempotencyConflict), errors.Is(err, postgres.ErrLoyaltyInsufficientBalance):
			writeError(w, http.StatusConflict, "LOYALTY_HOLD_CONFLICT", "redemption hold conflicts with the current order or point balance")
		default:
			writeError(w, http.StatusBadRequest, "INVALID_LOYALTY_HOLD", "redemption hold command is invalid")
		}
		return
	}
	status := http.StatusCreated
	if hold.Replay || hold.CommandResult == "ALREADY_APPLIED" {
		status = http.StatusOK
	}
	writeJSON(w, status, redemptionHoldFrom(hold))
}

func redemptionHoldFrom(hold postgres.LoyaltyHold) redemptionHoldResponse {
	return redemptionHoldResponse{
		OrderID: hold.OrderID, HoldID: hold.ID, CustomerID: hold.CustomerID, Points: hold.Points,
		DiscountAmount: hold.DiscountAmount, State: hold.State, ResourceVersion: hold.Version,
		ExpiresAt:     hold.ExpiresAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		PolicyVersion: hold.PolicyVersion, CommandResult: hold.CommandResult,
		Tombstone: hold.Tombstone, Replay: hold.Replay,
	}
}

func (h *LoyaltyHandler) GetPoints(w http.ResponseWriter, r *http.Request) {
	customerID := strings.TrimSpace(chi.URLParam(r, "customerID"))
	if customerID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_CUSTOMER", "customer id is required")
		return
	}
	page, err := h.repository.GetLedgerPage(r.Context(), customerID, 1, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not read loyalty balance")
		return
	}
	policy, err := h.repository.GetActivePolicy(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "LOYALTY_POLICY_UNAVAILABLE", "loyalty policy is unavailable")
		return
	}
	rate, err := policy.PointsToVNDIntegerRate()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "LOYALTY_POLICY_UNSUPPORTED", "active points-to-VND rate is not supported by the customer contract")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"customer_id": customerID, "current_points": page.Balance,
		"tier": policy.BaseTierCode, "points_to_vnd_rate": rate,
	})
}

func (h *LoyaltyHandler) GetActivePolicy(w http.ResponseWriter, r *http.Request) {
	policy, err := h.repository.GetActivePolicy(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "LOYALTY_POLICY_UNAVAILABLE", "loyalty policy is unavailable")
		return
	}
	rate, err := policy.PointsToVNDIntegerRate()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "LOYALTY_POLICY_UNSUPPORTED", "active points-to-VND rate is not supported by the customer contract")
		return
	}
	writeJSON(w, http.StatusOK, loyaltyPolicyResponse{
		PolicyVersion: policy.Version, EarnRateBasisPoints: policy.EarnRateBasisPoints,
		PointsToVNDRate: rate, BaseTierCode: policy.BaseTierCode, RoundingMode: policy.RoundingMode,
		HoldTTLMinutes: policy.HoldTTLMinutes, PointsExpireDays: policy.PointsExpireDays,
	})
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

func requireAnyInternalToken(tokens ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			supplied := r.Header.Get("X-Internal-Token")
			for _, token := range tokens {
				if token != "" && len(supplied) == len(token) && subtle.ConstantTimeCompare([]byte(supplied), []byte(token)) == 1 {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "trusted internal service authentication is required")
		})
	}
}
