package ghn

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/omamx/shipping-service/internal/domain"
)

const (
	lightServiceType = 2
	heavyServiceType = 5
	requiredNote     = "KHONGCHOXEMHANG"
)

func (c *Client) Code() string { return Code }

type orderRequest struct {
	PaymentTypeID   int         `json:"payment_type_id"`
	RequiredNote    string      `json:"required_note"`
	ClientOrderCode string      `json:"client_order_code,omitempty"`
	ToName          string      `json:"to_name"`
	ToPhone         string      `json:"to_phone"`
	ToAddress       string      `json:"to_address"`
	ToWardName      string      `json:"to_ward_name"`
	ToDistrictName  string      `json:"to_district_name"`
	ToProvinceName  string      `json:"to_province_name"`
	CODAmount       int64       `json:"cod_amount,omitempty"`
	ServiceTypeID   int         `json:"service_type_id"`
	Content         string      `json:"content"`
	Note            string      `json:"note,omitempty"`
	Weight          int64       `json:"weight"`
	Length          int32       `json:"length"`
	Width           int32       `json:"width"`
	Height          int32       `json:"height"`
	Items           []orderItem `json:"items"`
}

type orderItem struct {
	Name     string `json:"name"`
	Code     string `json:"code,omitempty"`
	Quantity int    `json:"quantity"`
	Price    int64  `json:"price,omitempty"`
	Length   int32  `json:"length"`
	Width    int32  `json:"width"`
	Height   int32  `json:"height"`
	Weight   int64  `json:"weight"`
}

type orderResponse struct {
	OrderCode            string    `json:"order_code"`
	TotalFee             int64     `json:"total_fee"`
	ExpectedDeliveryTime time.Time `json:"expected_delivery_time"`
}

type orderDetailResponse struct {
	OrderCode       string    `json:"order_code"`
	ClientOrderCode string    `json:"client_order_code"`
	Status          string    `json:"status"`
	Leadtime        time.Time `json:"leadtime"`
}

type cancelOrderRequest struct {
	OrderCodes []string `json:"order_codes"`
	ReasonCode string   `json:"reason_code,omitempty"`
	Reason     string   `json:"reason,omitempty"`
}

type cancelOrderResult struct {
	OrderCode string `json:"order_code"`
	Result    bool   `json:"result"`
	Message   string `json:"message"`
}

type printTokenRequest struct {
	OrderCodes []string `json:"order_codes"`
}

type printTokenResponse struct {
	Token string `json:"token"`
}

func (c *Client) Quote(ctx context.Context, request carrier.QuoteRequest) (carrier.Quote, error) {
	if err := request.Destination.Validate(); err != nil {
		return carrier.Quote{}, err
	}
	if err := request.Parcel.Validate(); err != nil {
		return carrier.Quote{}, err
	}
	serviceType, serviceCode, serviceName, err := resolveService(request.ServiceCode, request.Parcel.WeightGrams)
	if err != nil {
		return carrier.Quote{}, err
	}
	payload := newOrderRequest(request.Destination, request.Parcel, serviceType, "", 0, "")
	var result orderResponse
	if err := c.do(ctx, http.MethodPost, "/shiip/public-api/v2/shipping-order/preview", payload, &result); err != nil {
		return carrier.Quote{}, err
	}
	if result.TotalFee < 0 || result.ExpectedDeliveryTime.IsZero() {
		return carrier.Quote{}, fmt.Errorf("GHN preview returned invalid fee or delivery time")
	}
	now := c.now().UTC()
	return carrier.Quote{
		CarrierCode: Code, ServiceCode: serviceCode, ServiceName: serviceName,
		FeeVND: result.TotalFee, EstimatedDeliveryAt: result.ExpectedDeliveryTime.UTC(),
		ExpiresAt: now.Add(c.quoteTTL),
	}, nil
}

func (c *Client) CreateShipment(ctx context.Context, request carrier.CreateRequest) (carrier.CreateResult, error) {
	if strings.TrimSpace(request.IdempotencyKey) == "" || strings.TrimSpace(request.ShipmentID) == "" {
		return carrier.CreateResult{}, fmt.Errorf("idempotency key and shipment ID are required")
	}
	if err := request.Recipient.Validate(); err != nil {
		return carrier.CreateResult{}, err
	}
	if err := request.Parcel.Validate(); err != nil {
		return carrier.CreateResult{}, err
	}
	if request.CODAmountVND < 0 || request.CODAmountVND > 50_000_000 {
		return carrier.CreateResult{}, fmt.Errorf("GHN COD amount must be between 0 and 50000000 VND")
	}
	serviceType, _, _, err := resolveService(request.ServiceCode, request.Parcel.WeightGrams)
	if err != nil {
		return carrier.CreateResult{}, err
	}
	payload := newOrderRequest(
		request.Recipient, request.Parcel, serviceType,
		request.ShipmentID, request.CODAmountVND, request.ShippingNotes,
	)
	var result orderResponse
	if createErr := c.do(ctx, http.MethodPost, "/shiip/public-api/v2/shipping-order/create", payload, &result); createErr != nil {
		if !isAmbiguousMutationError(createErr) {
			return carrier.CreateResult{}, createErr
		}
		reconcileCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), c.reconcileTimeout)
		defer cancel()
		detail, reconcileErr := c.getOrderByClientCode(reconcileCtx, request.ShipmentID)
		if reconcileErr != nil {
			return carrier.CreateResult{}, fmt.Errorf("create GHN order was ambiguous and reconciliation failed: %w", errors.Join(createErr, reconcileErr))
		}
		if detail.ClientOrderCode != request.ShipmentID || strings.TrimSpace(detail.OrderCode) == "" {
			return carrier.CreateResult{}, fmt.Errorf("GHN reconciliation returned a mismatched order")
		}
		recoveredFee := request.CheckoutFeeVND
		delivery := detail.Leadtime
		var preview orderResponse
		if previewErr := c.do(reconcileCtx, http.MethodPost, "/shiip/public-api/v2/shipping-order/preview", payload, &preview); previewErr == nil && preview.TotalFee > 0 {
			recoveredFee = preview.TotalFee
			if delivery.IsZero() {
				delivery = preview.ExpectedDeliveryTime
			}
		}
		// GHN detail-by-client-code does not expose total_fee. If preview is also
		// unavailable, preserve the checkout quote rather than orphaning an order
		// that GHN already created or recording an invented provider fee.
		return createResult(detail.OrderCode, recoveredFee, delivery), nil
	}
	if strings.TrimSpace(result.OrderCode) == "" {
		return carrier.CreateResult{}, fmt.Errorf("GHN create order returned an empty order code")
	}
	return createResult(result.OrderCode, result.TotalFee, result.ExpectedDeliveryTime), nil
}

func (c *Client) getOrderByClientCode(ctx context.Context, clientOrderCode string) (orderDetailResponse, error) {
	query := url.Values{"client_order_code": []string{clientOrderCode}}
	var detail orderDetailResponse
	err := c.do(ctx, http.MethodGet, "/shiip/public-api/v2/shipping-order/detail-by-client-code?"+query.Encode(), nil, &detail)
	return detail, err
}

func (c *Client) CancelShipment(ctx context.Context, request carrier.CancelRequest) error {
	orderCode := strings.TrimSpace(request.CarrierOrderID)
	if orderCode == "" {
		return fmt.Errorf("GHN carrier order id is required")
	}
	reasonCode, err := resolveCancelReason(request.ReasonCode)
	if err != nil {
		return err
	}
	payload := cancelOrderRequest{
		OrderCodes: []string{orderCode}, ReasonCode: reasonCode, Reason: strings.TrimSpace(request.Reason),
	}
	var results []cancelOrderResult
	if cancelErr := c.do(ctx, http.MethodPost, "/shiip/public-api/v2/switch-status/cancel", payload, &results); cancelErr != nil {
		if !isAmbiguousMutationError(cancelErr) {
			return cancelErr
		}
		return c.confirmCancellation(ctx, orderCode, cancelErr)
	}
	if len(results) != 1 || !strings.EqualFold(strings.TrimSpace(results[0].OrderCode), orderCode) {
		return fmt.Errorf("GHN cancel returned a mismatched result")
	}
	if results[0].Result {
		return nil
	}
	rejected := fmt.Errorf("GHN cancel rejected order %s: %s", orderCode, strings.TrimSpace(results[0].Message))
	return c.confirmCancellation(ctx, orderCode, rejected)
}

func (c *Client) confirmCancellation(ctx context.Context, orderCode string, operationErr error) error {
	reconcileCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), c.reconcileTimeout)
	defer cancel()
	detail, detailErr := c.getOrderByCode(reconcileCtx, orderCode)
	if detailErr == nil && strings.EqualFold(strings.TrimSpace(detail.Status), "cancel") {
		return nil
	}
	if detailErr != nil {
		return fmt.Errorf("GHN cancellation could not be confirmed: %w", errors.Join(operationErr, detailErr))
	}
	return operationErr
}

func (c *Client) getOrderByCode(ctx context.Context, orderCode string) (orderDetailResponse, error) {
	query := url.Values{"order_code": []string{orderCode}}
	var detail orderDetailResponse
	err := c.do(ctx, http.MethodGet, "/shiip/public-api/v2/shipping-order/detail?"+query.Encode(), nil, &detail)
	return detail, err
}

func resolveCancelReason(reasonCode string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(reasonCode)) {
	case "", carrier.CancelReasonOther:
		return "GHN-CANCEL-OTHER", nil
	case carrier.CancelReasonPickupDelay:
		return "GHN-CO001", nil
	case carrier.CancelReasonOutOfStock:
		return "GHN-CO002", nil
	case carrier.CancelReasonRecipientChangedMind:
		return "GHN-CO003", nil
	default:
		return "", fmt.Errorf("unsupported cancellation reason code %q", reasonCode)
	}
}

func (c *Client) GenerateLabelURL(ctx context.Context, request carrier.LabelRequest) (carrier.LabelResult, error) {
	orderCode := strings.TrimSpace(request.CarrierOrderID)
	if orderCode == "" {
		return carrier.LabelResult{}, fmt.Errorf("GHN carrier order id is required")
	}
	format, err := carrier.NormalizeLabelFormat(request.Format)
	if err != nil {
		return carrier.LabelResult{}, err
	}
	var result printTokenResponse
	if err := c.do(ctx, http.MethodPost, "/shiip/public-api/v2/a5/gen-token", printTokenRequest{
		OrderCodes: []string{orderCode},
	}, &result); err != nil {
		return carrier.LabelResult{}, err
	}
	if strings.TrimSpace(result.Token) == "" {
		return carrier.LabelResult{}, fmt.Errorf("GHN print token is empty")
	}

	printURL := *c.baseURL
	switch format {
	case carrier.LabelFormatA5:
		printURL.Path = "/a5/public-api/printA5"
	case carrier.LabelFormat80X80:
		printURL.Path = "/a5/public-api/print80x80"
	case carrier.LabelFormat52X70:
		printURL.Path = "/a5/public-api/print52x70"
	}
	printURL.RawQuery = url.Values{"token": []string{result.Token}}.Encode()
	return carrier.LabelResult{
		URL: printURL.String(), ExpiresAt: c.now().UTC().Add(25 * time.Minute),
	}, nil
}

func isAmbiguousMutationError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Temporary()
	}
	// Network failures, timeouts, truncated responses and malformed 2xx responses do not
	// prove whether GHN committed the order, so they must be reconciled before retrying.
	return true
}

func createResult(orderCode string, fee int64, estimatedDelivery time.Time) carrier.CreateResult {
	var delivery *time.Time
	if !estimatedDelivery.IsZero() {
		value := estimatedDelivery.UTC()
		delivery = &value
	}
	return carrier.CreateResult{
		CarrierOrderID: orderCode, TrackingCode: orderCode,
		LabelReference: "GHN:" + orderCode, EstimatedFeeVND: fee,
		EstimatedDeliveryAt: delivery,
	}
}

func newOrderRequest(recipient domain.Recipient, parcel domain.Parcel, serviceType int, clientOrderCode string, cod int64, note string) orderRequest {
	return orderRequest{
		PaymentTypeID: 1, RequiredNote: requiredNote, ClientOrderCode: clientOrderCode,
		ToName: strings.TrimSpace(recipient.Name), ToPhone: strings.TrimSpace(recipient.Phone),
		ToAddress: fullAddress(recipient), ToWardName: strings.TrimSpace(recipient.Ward),
		ToDistrictName: strings.TrimSpace(recipient.District), ToProvinceName: strings.TrimSpace(recipient.Province),
		CODAmount: cod, ServiceTypeID: serviceType, Content: "Đặc sản O Mạ",
		Note: strings.TrimSpace(note), Weight: parcel.WeightGrams,
		Length: parcel.LengthCM, Width: parcel.WidthCM, Height: parcel.HeightCM,
		Items: []orderItem{{
			Name: "Kiện hàng O Mạ", Code: clientOrderCode, Quantity: 1,
			Length: parcel.LengthCM, Width: parcel.WidthCM, Height: parcel.HeightCM, Weight: parcel.WeightGrams,
		}},
	}
}

func fullAddress(recipient domain.Recipient) string {
	parts := []string{recipient.Street, recipient.Ward, recipient.District, recipient.Province}
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return strings.Join(result, ", ")
}

func resolveService(requested string, weight int64) (int, string, string, error) {
	value := strings.ToUpper(strings.TrimSpace(requested))
	if value == "" || value == "STANDARD" || value == "GHN_LIGHT" || value == "2" {
		if weight >= 20_000 && value != "2" && value != "GHN_LIGHT" {
			return heavyServiceType, "GHN_HEAVY", "GHN Hàng nặng", nil
		}
		return lightServiceType, "GHN_LIGHT", "GHN Hàng nhẹ", nil
	}
	if value == "HEAVY" || value == "GHN_HEAVY" || value == "5" {
		return heavyServiceType, "GHN_HEAVY", "GHN Hàng nặng", nil
	}
	return 0, "", "", fmt.Errorf("unsupported GHN service code %q", requested)
}

type webhookPayload struct {
	ShopID          int64     `json:"ShopID"`
	Time            time.Time `json:"Time"`
	OrderCode       string    `json:"OrderCode"`
	ClientOrderCode string    `json:"ClientOrderCode"`
	Type            string    `json:"Type"`
	Description     string    `json:"Description"`
	Status          string    `json:"Status"`
	Reason          string    `json:"Reason"`
	ReasonCode      string    `json:"ReasonCode"`
	Warehouse       string    `json:"Warehouse"`
}

func (c *Client) ParseWebhook(body []byte) (carrier.WebhookUpdate, error) {
	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return carrier.WebhookUpdate{}, fmt.Errorf("%w: decode GHN payload: %v", carrier.ErrInvalidWebhook, err)
	}
	if payload.ShopID != c.shopID {
		return carrier.WebhookUpdate{}, fmt.Errorf("%w: GHN shop ID does not match", carrier.ErrInvalidWebhook)
	}
	if strings.TrimSpace(payload.OrderCode) == "" || strings.TrimSpace(payload.Type) == "" ||
		strings.TrimSpace(payload.Status) == "" || payload.Time.IsZero() {
		return carrier.WebhookUpdate{}, fmt.Errorf("%w: OrderCode, Type, Status and Time are required", carrier.ErrInvalidWebhook)
	}
	mapped, ok := statusMap[strings.ToLower(strings.TrimSpace(payload.Status))]
	if !ok {
		return carrier.WebhookUpdate{}, fmt.Errorf("%w: unmapped GHN status %q", carrier.ErrInvalidWebhook, payload.Status)
	}
	eventKey := strings.Join([]string{
		payload.OrderCode, payload.Type, payload.Time.UTC().Format(time.RFC3339Nano), payload.Status,
	}, "|")
	sum := sha256.Sum256([]byte(eventKey))
	internalNote := strings.TrimSpace(strings.Join([]string{payload.ReasonCode, payload.Reason}, " "))
	publicDescription := strings.TrimSpace(payload.Description)
	if publicDescription == "" {
		publicDescription = strings.TrimSpace(payload.Reason)
	}
	return carrier.WebhookUpdate{
		ExternalEventID: "GHN-" + hex.EncodeToString(sum[:]), TrackingCode: payload.OrderCode,
		ProviderStatusCode:  strings.ToLower(strings.TrimSpace(payload.Status)),
		ProviderStatusLabel: strings.TrimSpace(payload.Description), MappedStatus: mapped,
		Location: strings.TrimSpace(payload.Warehouse), PublicDescription: publicDescription,
		InternalNote: internalNote, EventAt: payload.Time.UTC(),
	}, nil
}

var statusMap = map[string]domain.ShipmentStatus{
	"ready_to_pick":            domain.StatusReadyForHandover,
	"picking":                  domain.StatusReadyForHandover,
	"money_collect_picking":    domain.StatusReadyForHandover,
	"picked":                   domain.StatusHandedOver,
	"storing":                  domain.StatusInTransit,
	"transporting":             domain.StatusInTransit,
	"sorting":                  domain.StatusInTransit,
	"delivering":               domain.StatusOutForDelivery,
	"money_collect_delivering": domain.StatusOutForDelivery,
	"delivered":                domain.StatusDelivered,
	"delivery_fail":            domain.StatusDeliveryFailed,
	"waiting_to_return":        domain.StatusReturning,
	"return":                   domain.StatusReturning,
	"return_transporting":      domain.StatusReturning,
	"return_sorting":           domain.StatusReturning,
	"returning":                domain.StatusReturning,
	"returned":                 domain.StatusReturned,
	"return_fail":              domain.StatusDeliveryFailed,
	"cancel":                   domain.StatusCancelled,
	"exception":                domain.StatusDeliveryFailed,
	"lost":                     domain.StatusDeliveryFailed,
	"damage":                   domain.StatusDeliveryFailed,
}
