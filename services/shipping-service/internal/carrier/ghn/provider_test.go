package ghn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/omamx/shipping-service/internal/domain"
)

func TestQuoteUsesPreviewWithoutCreatingOrder(t *testing.T) {
	client := testClientWithDoer(t, doerFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/shiip/public-api/v2/shipping-order/preview" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Token") != "staging-token" || r.Header.Get("ShopId") != "92837" {
			t.Fatal("GHN authentication headers are missing")
		}
		var request orderRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.ClientOrderCode != "" || request.PaymentTypeID != 1 || request.ServiceTypeID != lightServiceType {
			t.Fatalf("unexpected preview request: %+v", request)
		}
		if request.ToWardName != "Phường Thuận Hòa" || request.ToDistrictName != "Quận Phú Xuân" {
			t.Fatalf("recipient address was not mapped by name: %+v", request)
		}
		return response(t, http.StatusOK, map[string]any{
			"order_code": "", "total_fee": 32000,
			"expected_delivery_time": "2026-11-03T16:59:59+07:00",
		}), nil
	}))

	now := time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC)
	client.now = func() time.Time { return now }
	quote, err := client.Quote(context.Background(), carrier.QuoteRequest{
		Destination: testRecipient(), Parcel: testParcel(), ServiceCode: "STANDARD",
	})
	if err != nil {
		t.Fatal(err)
	}
	if quote.CarrierCode != Code || quote.ServiceCode != "GHN_LIGHT" || quote.FeeVND != 32000 {
		t.Fatalf("unexpected quote: %+v", quote)
	}
	if !quote.ExpiresAt.Equal(now.Add(15 * time.Minute)) {
		t.Fatalf("unexpected quote expiry %s", quote.ExpiresAt)
	}
}

func TestCreateShipmentUsesInternalShipmentIDAsClientOrderCode(t *testing.T) {
	client := testClientWithDoer(t, doerFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/shiip/public-api/v2/shipping-order/create" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var request orderRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.ClientOrderCode != "c51be44f-e7a3-4e1f-9093-507948136699" {
			t.Fatalf("unexpected client_order_code %q", request.ClientOrderCode)
		}
		if request.CODAmount != 285000 || request.RequiredNote != requiredNote || request.Note != "Gọi trước khi giao" {
			t.Fatalf("unexpected create request: %+v", request)
		}
		return response(t, http.StatusOK, map[string]any{
			"order_code": "LADFYR", "total_fee": 35000,
			"expected_delivery_time": "2026-11-03T16:59:59+07:00",
		}), nil
	}))

	result, err := client.CreateShipment(context.Background(), carrier.CreateRequest{
		IdempotencyKey: "packing-event-1", ShipmentID: "c51be44f-e7a3-4e1f-9093-507948136699",
		OrderID: "order-1", PackageID: "package-1", ServiceCode: "GHN_LIGHT",
		Recipient: testRecipient(), Parcel: testParcel(), CODAmountVND: 285000,
		ShippingNotes: "Gọi trước khi giao",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TrackingCode != "LADFYR" || result.CarrierOrderID != "LADFYR" || result.LabelReference != "GHN:LADFYR" {
		t.Fatalf("unexpected create result: %+v", result)
	}
	if result.EstimatedFeeVND != 35000 || result.EstimatedDeliveryAt == nil {
		t.Fatalf("missing provider result data: %+v", result)
	}
}

func TestCreateShipmentRecoversAmbiguousResultByClientOrderCode(t *testing.T) {
	const shipmentID = "c51be44f-e7a3-4e1f-9093-507948136699"
	calls := 0
	client := testClientWithDoer(t, doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/shiip/public-api/v2/shipping-order/create":
			return nil, context.DeadlineExceeded
		case r.Method == http.MethodGet && r.URL.Path == "/shiip/public-api/v2/shipping-order/detail-by-client-code":
			if r.URL.Query().Get("client_order_code") != shipmentID {
				t.Fatalf("unexpected reconciliation query %q", r.URL.RawQuery)
			}
			return response(t, http.StatusOK, map[string]any{
				"order_code": "LADRECOVERED", "client_order_code": shipmentID,
				"status": "ready_to_pick", "leadtime": "2026-11-03T16:59:59+07:00",
			}), nil
		case r.Method == http.MethodPost && r.URL.Path == "/shiip/public-api/v2/shipping-order/preview":
			return response(t, http.StatusOK, map[string]any{
				"total_fee": 34000, "expected_delivery_time": "2026-11-03T16:59:59+07:00",
			}), nil
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
			return nil, nil
		}
	}))

	result, err := client.CreateShipment(context.Background(), carrier.CreateRequest{
		IdempotencyKey: "packing-event-1", ShipmentID: shipmentID,
		OrderID: "order-1", PackageID: "package-1", ServiceCode: "GHN_LIGHT",
		Recipient: testRecipient(), Parcel: testParcel(), CODAmountVND: 285000, CheckoutFeeVND: 53900,
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 || result.TrackingCode != "LADRECOVERED" || result.EstimatedFeeVND != 34000 {
		t.Fatalf("unexpected recovered result after %d calls: %+v", calls, result)
	}
}

func TestCreateShipmentUsesCheckoutFeeWhenRecoveryPreviewFails(t *testing.T) {
	const shipmentID = "c51be44f-e7a3-4e1f-9093-507948136699"
	calls := 0
	client := testClientWithDoer(t, doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/shiip/public-api/v2/shipping-order/create":
			return nil, context.DeadlineExceeded
		case r.Method == http.MethodGet && r.URL.Path == "/shiip/public-api/v2/shipping-order/detail-by-client-code":
			return response(t, http.StatusOK, map[string]any{
				"order_code": "LADFALLBACK", "client_order_code": shipmentID,
				"status": "ready_to_pick", "leadtime": "2026-11-03T16:59:59+07:00",
			}), nil
		case r.Method == http.MethodPost && r.URL.Path == "/shiip/public-api/v2/shipping-order/preview":
			return nil, context.DeadlineExceeded
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
			return nil, nil
		}
	}))

	result, err := client.CreateShipment(context.Background(), carrier.CreateRequest{
		IdempotencyKey: "packing-event-1", ShipmentID: shipmentID,
		OrderID: "order-1", PackageID: "package-1", ServiceCode: "GHN_LIGHT",
		Recipient: testRecipient(), Parcel: testParcel(), CheckoutFeeVND: 53900,
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 || result.TrackingCode != "LADFALLBACK" || result.EstimatedFeeVND != 53900 {
		t.Fatalf("unexpected fallback result after %d calls: %+v", calls, result)
	}
}

func TestCreateShipmentDoesNotReconcileDefinitiveProviderError(t *testing.T) {
	calls := 0
	client := testClientWithDoer(t, doerFunc(func(_ *http.Request) (*http.Response, error) {
		calls++
		return rawResponse(http.StatusBadRequest, `{"code":400,"message":"invalid address","code_message":"USER_ERR_COMMON","data":null}`), nil
	}))

	_, err := client.CreateShipment(context.Background(), carrier.CreateRequest{
		IdempotencyKey: "packing-event-1", ShipmentID: "c51be44f-e7a3-4e1f-9093-507948136699",
		OrderID: "order-1", PackageID: "package-1", ServiceCode: "GHN_LIGHT",
		Recipient: testRecipient(), Parcel: testParcel(),
	})
	if err == nil {
		t.Fatal("expected definitive GHN validation error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("expected GHN API error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("definitive 4xx must not trigger reconciliation, got %d calls", calls)
	}
}

func TestCancelShipmentMapsReasonAndAcceptsProviderResult(t *testing.T) {
	calls := 0
	client := testClientWithDoer(t, doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/shiip/public-api/v2/switch-status/cancel" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		var payload cancelOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.OrderCodes) != 1 || payload.OrderCodes[0] != "LADFYR" ||
			payload.ReasonCode != "GHN-CO003" || payload.Reason != "Khách đổi ý" {
			t.Fatalf("unexpected cancel payload: %+v", payload)
		}
		return response(t, http.StatusOK, []map[string]any{{
			"order_code": "LADFYR", "result": true, "message": "OK",
		}}), nil
	}))

	err := client.CancelShipment(context.Background(), carrier.CancelRequest{
		CarrierOrderID: "LADFYR", ReasonCode: carrier.CancelReasonRecipientChangedMind, Reason: "Khách đổi ý",
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected one cancel request, got %d", calls)
	}
}

func TestCancelShipmentConfirmsAmbiguousResultWithoutRetryingMutation(t *testing.T) {
	calls := 0
	client := testClientWithDoer(t, doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/shiip/public-api/v2/switch-status/cancel":
			return nil, context.DeadlineExceeded
		case r.Method == http.MethodGet && r.URL.Path == "/shiip/public-api/v2/shipping-order/detail":
			if r.URL.Query().Get("order_code") != "LADFYR" {
				t.Fatalf("unexpected detail query %q", r.URL.RawQuery)
			}
			return response(t, http.StatusOK, map[string]any{
				"order_code": "LADFYR", "status": "cancel",
			}), nil
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
			return nil, nil
		}
	}))

	if err := client.CancelShipment(context.Background(), carrier.CancelRequest{CarrierOrderID: "LADFYR"}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("ambiguous cancellation must use one status lookup, got %d calls", calls)
	}
}

func TestCancelShipmentRejectsProviderResultWhenOrderIsNotCancelled(t *testing.T) {
	calls := 0
	client := testClientWithDoer(t, doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/shiip/public-api/v2/switch-status/cancel":
			return response(t, http.StatusOK, []map[string]any{{
				"order_code": "LADFYR", "result": false, "message": "Trạng thái đơn hàng không hợp lệ",
			}}), nil
		case r.Method == http.MethodGet && r.URL.Path == "/shiip/public-api/v2/shipping-order/detail":
			return response(t, http.StatusOK, map[string]any{
				"order_code": "LADFYR", "status": "picked",
			}), nil
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
			return nil, nil
		}
	}))

	err := client.CancelShipment(context.Background(), carrier.CancelRequest{CarrierOrderID: "LADFYR"})
	if err == nil || !strings.Contains(err.Error(), "Trạng thái đơn hàng không hợp lệ") {
		t.Fatalf("expected provider rejection, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected cancel and status lookup, got %d calls", calls)
	}
}

func TestParseWebhookValidatesShopAndMapsStatus(t *testing.T) {
	client := testClient(t, "https://example.test")
	payload := []byte(`{
		"ShopID":92837,
		"Time":"2026-10-31T08:30:00Z",
		"OrderCode":"LADFYR",
		"ClientOrderCode":"c51be44f-e7a3-4e1f-9093-507948136699",
		"Type":"switch_status",
		"Description":"Giao hàng thành công",
		"Status":"delivered",
		"Reason":"",
		"ReasonCode":"",
		"Warehouse":"Bưu cục Huế"
	}`)
	first, err := client.ParseWebhook(payload)
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.ParseWebhook(payload)
	if err != nil {
		t.Fatal(err)
	}
	if first.ExternalEventID == "" || first.ExternalEventID != second.ExternalEventID {
		t.Fatal("webhook event identity must be deterministic")
	}
	if first.TrackingCode != "LADFYR" || first.MappedStatus != domain.StatusDelivered || first.Location != "Bưu cục Huế" {
		t.Fatalf("unexpected webhook mapping: %+v", first)
	}

	wrongShop := []byte(`{"ShopID":1,"Time":"2026-10-31T08:30:00Z","OrderCode":"LADFYR","Type":"switch_status","Status":"delivered"}`)
	if _, err := client.ParseWebhook(wrongShop); err == nil {
		t.Fatal("expected wrong shop webhook to be rejected")
	}
}

func TestAPIErrorDoesNotExposeToken(t *testing.T) {
	client := testClientWithDoer(t, doerFunc(func(_ *http.Request) (*http.Response, error) {
		return rawResponse(http.StatusBadRequest, `{"code":400,"message":"route not found","code_message":"USER_ERR_COMMON","data":null}`), nil
	}))
	_, err := client.Quote(context.Background(), carrier.QuoteRequest{Destination: testRecipient(), Parcel: testParcel()})
	if err == nil {
		t.Fatal("expected GHN API error")
	}
	if err.Error() != "GHN USER_ERR_COMMON: route not found" {
		t.Fatalf("unexpected safe error: %v", err)
	}
}

func testClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	client, err := New(Config{
		BaseURL: baseURL, Token: "staging-token", ShopID: 92837,
		FromDistrictID: 1452, FromWardCode: "21007", Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func testClientWithDoer(t *testing.T, doer HTTPDoer) *Client {
	t.Helper()
	client, err := NewWithHTTPClient(Config{
		BaseURL: "https://ghn.test", Token: "staging-token", ShopID: 92837,
		FromDistrictID: 1452, FromWardCode: "21007", Timeout: time.Second,
	}, doer)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func testRecipient() domain.Recipient {
	return domain.Recipient{
		Name: "Nguyễn Văn An", Phone: "0905000000", Street: "12 Lê Duẩn",
		Ward: "Phường Thuận Hòa", District: "Quận Phú Xuân", Province: "Thành phố Huế", CountryCode: "VN",
	}
}

func testParcel() domain.Parcel {
	return domain.Parcel{WeightGrams: 800, LengthCM: 25, WidthCM: 20, HeightCM: 10, Fragile: true}
}

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(request *http.Request) (*http.Response, error) { return f(request) }

func response(t *testing.T, status int, data any) *http.Response {
	t.Helper()
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(map[string]any{"code": status, "message": "Success", "data": data}); err != nil {
		t.Fatal(err)
	}
	return rawResponse(status, body.String())
}

func rawResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewBufferString(body)), Header: make(http.Header)}
}
