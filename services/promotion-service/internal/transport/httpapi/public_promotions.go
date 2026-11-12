package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/omamx/promotion-service/internal/domain"
	"github.com/omamx/promotion-service/internal/repository/postgres"
)

type PublicPromotionHandler struct {
	repository publicPromotionRepository
}

type publicPromotionRepository interface {
	ListActive(context.Context, int, int) ([]postgres.AdminCoupon, error)
	ListPublicActive(context.Context, string, int, int) ([]postgres.PublicCoupon, error)
}

func RegisterPublicPromotionRoutes(router chi.Router, repository publicPromotionRepository) {
	handler := &PublicPromotionHandler{repository: repository}
	router.Get("/api/v1/promotions/vouchers", handler.ListVouchers)
	router.Get("/api/v1/promotions/active", handler.ListActive)
}

type publicVoucher struct {
	Code              string              `json:"voucher_code"`
	Title             string              `json:"title"`
	DiscountType      domain.DiscountType `json:"discount_type"`
	DiscountValue     int64               `json:"discount_value"`
	MinimumOrder      int64               `json:"min_order_amount"`
	MaxDiscountAmount *int64              `json:"max_discount_amount"`
	ExpiresAt         time.Time           `json:"expires_at"`
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

func (h *PublicPromotionHandler) ListVouchers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	search := strings.TrimSpace(query.Get("q"))
	if utf8.RuneCountInString(search) > 200 {
		writeError(w, http.StatusBadRequest, "INVALID_QUERY", "q must be at most 200 characters")
		return
	}
	if status := strings.TrimSpace(query.Get("status")); status != "" && status != "ACTIVE" {
		writeError(w, http.StatusBadRequest, "INVALID_STATUS", "public vouchers can only be listed with ACTIVE status")
		return
	}
	page, err := parsePositiveQuery(query.Get("page"), 1, 1, int(^uint(0)>>1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PAGINATION", "page must be a positive integer")
		return
	}
	pageSize, err := parsePositiveQuery(query.Get("pageSize"), 20, 1, 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PAGINATION", "pageSize must be between 1 and 100")
		return
	}
	maxInt := int(^uint(0) >> 1)
	if page-1 > maxInt/pageSize {
		writeError(w, http.StatusBadRequest, "INVALID_PAGINATION", "page is too large")
		return
	}
	vouchers, err := h.repository.ListPublicActive(r.Context(), search, pageSize, (page-1)*pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list public vouchers")
		return
	}
	items := make([]publicVoucher, 0, len(vouchers))
	for _, voucher := range vouchers {
		items = append(items, publicVoucherFrom(voucher))
	}
	writeJSON(w, http.StatusOK, map[string]any{"vouchers": items})
}

func parsePositiveQuery(raw string, fallback, minimum, maximum int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, strconv.ErrSyntax
	}
	return value, nil
}

func publicVoucherFrom(coupon postgres.PublicCoupon) publicVoucher {
	discountType := coupon.DiscountType
	if discountType == domain.DiscountFreeShipping {
		discountType = "FREESHIP"
	}
	return publicVoucher{
		Code: coupon.Code, Title: coupon.Title, DiscountType: discountType,
		DiscountValue: coupon.DiscountValue, MinimumOrder: coupon.MinimumOrderAmount,
		MaxDiscountAmount: coupon.MaxDiscountAmount, ExpiresAt: coupon.EndsAt,
	}
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
