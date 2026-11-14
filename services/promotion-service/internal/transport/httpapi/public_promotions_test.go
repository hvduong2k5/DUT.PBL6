package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/omamx/promotion-service/internal/domain"
	"github.com/omamx/promotion-service/internal/repository/postgres"
)

type publicPromotionRepositoryStub struct {
	search string
	limit  int
	offset int
	items  []postgres.PublicCoupon
}

func (s *publicPromotionRepositoryStub) ListActive(context.Context, int, int) ([]postgres.AdminCoupon, error) {
	return nil, nil
}

func (s *publicPromotionRepositoryStub) ListPublicActive(_ context.Context, search string, limit, offset int) ([]postgres.PublicCoupon, error) {
	s.search, s.limit, s.offset = search, limit, offset
	return s.items, nil
}

func TestListVouchersMatchesCustomerContractAndPagination(t *testing.T) {
	maxDiscount := int64(30000)
	repository := &publicPromotionRepositoryStub{items: []postgres.PublicCoupon{{
		Code: "OMAMA_FREESHIP_30K", Title: "Miễn phí vận chuyển 30k cho đơn từ 200k",
		DiscountType: domain.DiscountFreeShipping, DiscountValue: 0, MaxDiscountAmount: &maxDiscount,
		MinimumOrderAmount: 200000, EndsAt: time.Date(2026, 11, 30, 23, 59, 59, 0, time.UTC),
	}}}
	router := chi.NewRouter()
	RegisterPublicPromotionRoutes(router, repository)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/promotions/vouchers?q=free&page=3&pageSize=7", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	if repository.search != "free" || repository.limit != 7 || repository.offset != 14 {
		t.Fatalf("repository query = (%q, %d, %d)", repository.search, repository.limit, repository.offset)
	}
	var payload struct {
		Vouchers []map[string]any `json:"vouchers"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Vouchers) != 1 {
		t.Fatalf("voucher count=%d want=1", len(payload.Vouchers))
	}
	voucher := payload.Vouchers[0]
	for key, want := range map[string]any{
		"voucher_code": "OMAMA_FREESHIP_30K", "title": "Miễn phí vận chuyển 30k cho đơn từ 200k",
		"discount_type": "FREESHIP", "discount_value": float64(0), "min_order_amount": float64(200000),
		"max_discount_amount": float64(30000), "expires_at": "2026-11-30T23:59:59Z",
	} {
		if voucher[key] != want {
			t.Errorf("%s = %#v, want %#v", key, voucher[key], want)
		}
	}
	if len(voucher) != 7 {
		t.Fatalf("response has %d fields, want exactly the 7 contract fields: %#v", len(voucher), voucher)
	}
}

func TestListVouchersPreservesUnboundedDiscountAsNull(t *testing.T) {
	voucher := publicVoucherFrom(postgres.PublicCoupon{
		Code: "SAVE10", Title: "SAVE10", DiscountType: domain.DiscountPercentage,
		DiscountValue: 10, MaxDiscountAmount: nil, MinimumOrderAmount: 1,
		EndsAt: time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC),
	})
	body, err := json.Marshal(voucher)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"max_discount_amount":null`) {
		t.Fatalf("unbounded maximum should be explicit null: %s", body)
	}
}

func TestListVouchersRejectsInvalidContractQueries(t *testing.T) {
	router := chi.NewRouter()
	RegisterPublicPromotionRoutes(router, nil)
	for _, path := range []string{
		"/api/v1/promotions/vouchers?page=0",
		"/api/v1/promotions/vouchers?page=nope",
		"/api/v1/promotions/vouchers?pageSize=0",
		"/api/v1/promotions/vouchers?pageSize=101",
		"/api/v1/promotions/vouchers?status=PAUSED",
		"/api/v1/promotions/vouchers?q=" + strings.Repeat("x", 201),
		"/api/v1/promotions/vouchers?page=999999999999999999999999999999&pageSize=100",
	} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want=%d body=%s", response.Code, http.StatusBadRequest, response.Body.String())
			}
		})
	}
}
