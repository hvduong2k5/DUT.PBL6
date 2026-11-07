package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/omamx/promotion-service/internal/domain"
	"github.com/omamx/promotion-service/internal/repository/postgres"
)

type PublicPromotionHandler struct {
	repository *postgres.AdminCouponRepository
}

func RegisterPublicPromotionRoutes(router chi.Router, repository *postgres.AdminCouponRepository) {
	handler := &PublicPromotionHandler{repository: repository}
	router.Get("/api/v1/promotions/active", handler.ListActive)
}

type publicPromotion struct {
	Code              string              `json:"voucher_code"`
	DiscountType      domain.DiscountType `json:"discount_type"`
	DiscountValue     int64               `json:"discount_value"`
	MaxDiscountAmount *int64              `json:"max_discount_amount,omitempty"`
	MinimumOrder      int64               `json:"minimum_order_amount"`
	StartsAt          time.Time           `json:"starts_at"`
	EndsAt            time.Time           `json:"ends_at"`
}

func (h *PublicPromotionHandler) ListActive(w http.ResponseWriter, r *http.Request) {
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
	coupons, err := h.repository.ListActive(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list active promotions")
		return
	}
	items := make([]publicPromotion, 0, len(coupons))
	for _, coupon := range coupons {
		items = append(items, publicPromotion{
			Code: coupon.Code, DiscountType: coupon.DiscountType, DiscountValue: coupon.DiscountValue,
			MaxDiscountAmount: coupon.MaxDiscountAmount, MinimumOrder: coupon.MinimumOrderAmount,
			StartsAt: coupon.StartsAt, EndsAt: coupon.EndsAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}
