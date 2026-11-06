package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/omamx/promotion-service/internal/domain"
	"github.com/omamx/promotion-service/internal/repository/postgres"
)

type AdminCouponHandler struct {
	repository *postgres.AdminCouponRepository
}

func RegisterAdminCouponRoutes(router chi.Router, repository *postgres.AdminCouponRepository, internalToken string) {
	handler := &AdminCouponHandler{repository: repository}
	auth := requireSalesManager(internalToken)
	router.With(auth).Get("/api/v1/admin/coupons", handler.List)
	router.With(auth).Post("/api/v1/admin/coupons", handler.Create)
	router.With(auth).Put("/api/v1/admin/coupons/{couponID}", handler.UpdateDraft)
	router.With(auth).Post("/api/v1/admin/coupons/{couponID}/publish", handler.Publish)
	router.With(auth).Post("/api/v1/admin/coupons/{couponID}/pause", handler.Pause)
}

type couponRequest struct {
	Code                      string              `json:"code"`
	DiscountType              domain.DiscountType `json:"discount_type"`
	DiscountValue             int64               `json:"discount_value"`
	MaxDiscountAmount         *int64              `json:"max_discount_amount"`
	MinimumOrderAmount        int64               `json:"minimum_order_amount"`
	MaxRedemptions            *int64              `json:"max_redemptions"`
	MaxRedemptionsPerCustomer *int64              `json:"max_redemptions_per_customer"`
	StartsAt                  time.Time           `json:"starts_at"`
	EndsAt                    time.Time           `json:"ends_at"`
}

type couponResponse struct {
	ID                        string              `json:"id"`
	Code                      string              `json:"code"`
	DiscountType              domain.DiscountType `json:"discount_type"`
	DiscountValue             int64               `json:"discount_value"`
	MaxDiscountAmount         *int64              `json:"max_discount_amount,omitempty"`
	MinimumOrderAmount        int64               `json:"minimum_order_amount"`
	MaxRedemptions            *int64              `json:"max_redemptions,omitempty"`
	MaxRedemptionsPerCustomer *int64              `json:"max_redemptions_per_customer,omitempty"`
	StartsAt                  time.Time           `json:"starts_at"`
	EndsAt                    time.Time           `json:"ends_at"`
	Status                    string              `json:"status"`
	CreatedAt                 time.Time           `json:"created_at"`
	UpdatedAt                 time.Time           `json:"updated_at"`
}

func (h *AdminCouponHandler) Create(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeCouponRequest(w, r)
	if !ok {
		return
	}
	coupon, err := h.repository.Create(r.Context(), request.input(), actorFromRequest(r))
	if err != nil {
		writeAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, couponResponseFrom(coupon))
}

func (h *AdminCouponHandler) UpdateDraft(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeCouponRequest(w, r)
	if !ok {
		return
	}
	coupon, err := h.repository.UpdateDraft(r.Context(), chi.URLParam(r, "couponID"), request.input(), actorFromRequest(r))
	if err != nil {
		writeAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, couponResponseFrom(coupon))
}

func (h *AdminCouponHandler) Publish(w http.ResponseWriter, r *http.Request) {
	coupon, err := h.repository.Publish(r.Context(), chi.URLParam(r, "couponID"), actorFromRequest(r))
	if err != nil {
		writeAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, couponResponseFrom(coupon))
}

func (h *AdminCouponHandler) Pause(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request struct {
		Reason string `json:"reason"`
	}
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "request body must contain a reason")
		return
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "request body must contain one JSON object")
		return
	}
	coupon, err := h.repository.Pause(r.Context(), chi.URLParam(r, "couponID"), request.Reason, actorFromRequest(r))
	if err != nil {
		writeAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, couponResponseFrom(coupon))
}

func (h *AdminCouponHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_PAGINATION", "limit must be an integer")
			return
		}
		limit = parsed
	}
	if limit < 1 || limit > 100 {
		writeError(w, http.StatusBadRequest, "INVALID_PAGINATION", "limit must be between 1 and 100")
		return
	}
	var offset int
	if rawOffset := r.URL.Query().Get("offset"); rawOffset != "" {
		parsed, err := strconv.Atoi(rawOffset)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_PAGINATION", "offset must be an integer")
			return
		}
		offset = parsed
	}
	if offset < 0 {
		writeError(w, http.StatusBadRequest, "INVALID_PAGINATION", "offset must be a non-negative integer")
		return
	}
	coupons, err := h.repository.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list coupons")
		return
	}
	items := make([]couponResponse, 0, len(coupons))
	for _, coupon := range coupons {
		items = append(items, couponResponseFrom(coupon))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (request couponRequest) input() postgres.CouponInput {
	return postgres.CouponInput{
		Code: request.Code, DiscountType: request.DiscountType, DiscountValue: request.DiscountValue,
		MaxDiscountAmount: request.MaxDiscountAmount, MinimumOrderAmount: request.MinimumOrderAmount,
		MaxRedemptions: request.MaxRedemptions, MaxRedemptionsPerCustomer: request.MaxRedemptionsPerCustomer,
		StartsAt: request.StartsAt, EndsAt: request.EndsAt,
	}
}

func couponResponseFrom(value postgres.AdminCoupon) couponResponse {
	coupon := value.Coupon
	return couponResponse{
		ID: coupon.ID, Code: coupon.Code, DiscountType: coupon.DiscountType, DiscountValue: coupon.DiscountValue,
		MaxDiscountAmount: coupon.MaxDiscountAmount, MinimumOrderAmount: coupon.MinimumOrderAmount,
		MaxRedemptions: coupon.MaxRedemptions, MaxRedemptionsPerCustomer: coupon.MaxRedemptionsPerCustomer,
		StartsAt: coupon.StartsAt, EndsAt: coupon.EndsAt, Status: coupon.Status,
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func decodeCouponRequest(w http.ResponseWriter, r *http.Request) (couponRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request couponRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid coupon JSON")
		return couponRequest{}, false
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "request body must contain one JSON object")
		return couponRequest{}, false
	}
	return request, true
}

func requireSalesManager(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			supplied := r.Header.Get("X-Internal-Token")
			if token == "" || len(supplied) != len(token) || subtle.ConstantTimeCompare([]byte(supplied), []byte(token)) != 1 {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "internal gateway authentication is required")
				return
			}
			if strings.TrimSpace(r.Header.Get("X-User-ID")) == "" {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authenticated actor identity is required")
				return
			}
			if !strings.EqualFold(strings.TrimSpace(r.Header.Get("X-User-Role")), "SALES_MANAGER") {
				writeError(w, http.StatusForbidden, "FORBIDDEN", "sales manager permission is required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func actorFromRequest(r *http.Request) postgres.CouponActor {
	return postgres.CouponActor{ID: strings.TrimSpace(r.Header.Get("X-User-ID")), Role: strings.ToUpper(strings.TrimSpace(r.Header.Get("X-User-Role")))}
}

func writeAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrCouponConfigInvalid):
		writeError(w, http.StatusBadRequest, "INVALID_COUPON", "coupon configuration is invalid")
	case errors.Is(err, domain.ErrCouponCodeConflict):
		writeError(w, http.StatusConflict, "COUPON_CODE_EXISTS", "a coupon with this code already exists")
	case errors.Is(err, domain.ErrCouponImmutable):
		writeError(w, http.StatusConflict, "COUPON_NOT_EDITABLE", "only draft coupons can be edited")
	case errors.Is(err, domain.ErrCouponStateConflict):
		writeError(w, http.StatusConflict, "INVALID_COUPON_STATE", "coupon cannot make this state transition")
	case errors.Is(err, domain.ErrVoucherExpired):
		writeError(w, http.StatusConflict, "COUPON_EXPIRED", "expired coupon cannot be published or paused")
	case errors.Is(err, domain.ErrVoucherNotFound):
		writeError(w, http.StatusNotFound, "COUPON_NOT_FOUND", "coupon was not found")
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "coupon operation failed")
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
