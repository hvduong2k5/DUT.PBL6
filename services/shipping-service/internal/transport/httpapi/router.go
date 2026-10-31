package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/omamx/shipping-service/internal/domain"
	"github.com/rs/zerolog"
)

type Router struct {
	service           *application.Service
	internalToken     string
	fakeWebhookSecret string
	ghnWebhookSecret  string
	logger            zerolog.Logger
}

func NewRouter(service *application.Service, internalToken, fakeWebhookSecret, ghnWebhookSecret string, logger zerolog.Logger) http.Handler {
	h := &Router{
		service: service, internalToken: internalToken,
		fakeWebhookSecret: fakeWebhookSecret, ghnWebhookSecret: ghnWebhookSecret, logger: logger,
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Use(h.requestLog)
	r.Get("/health/live", h.live)
	r.Get("/health/ready", h.ready)
	r.With(h.requireInternalToken).Post("/api/v1/internal/shipping/quotes", h.quote)
	r.With(h.requireInternalToken).Post("/api/v1/internal/shipments", h.createShipment)
	r.With(h.requireInternalToken).Post("/api/v1/internal/shipments/{shipmentID}/cancel", h.cancelShipment)
	r.With(h.requireInternalToken).Post("/api/v1/internal/shipments/{shipmentID}/handover", h.confirmHandover)
	r.With(h.requireInternalToken).Get("/api/v1/admin/shipping/shipments", h.listShipments)
	r.With(h.requireInternalToken).Get("/api/v1/admin/shipping/shipments/{shipmentID}", h.getShipment)
	r.With(h.requireInternalToken).Get("/api/v1/orders/{orderID}/tracking", h.trackOrder)
	r.Post("/api/v1/webhooks/logistics/fake", h.fakeWebhook)
	r.Post("/api/v1/webhooks/logistics/ghn", h.ghnWebhook)
	return r
}

func (h *Router) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Router) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 2*time.Second)
	defer cancel()
	if err := h.service.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "NOT_READY", "database is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

type quoteRequest struct {
	CarrierCode string           `json:"carrier_code"`
	Destination domain.Recipient `json:"destination"`
	Parcel      domain.Parcel    `json:"parcel"`
	ServiceCode string           `json:"service_code,omitempty"`
}

func (h *Router) quote(w http.ResponseWriter, r *http.Request) {
	var request quoteRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	quote, err := h.service.Quote(r.Context(), request.CarrierCode, carrier.QuoteRequest{
		Destination: request.Destination, Parcel: request.Parcel, ServiceCode: request.ServiceCode,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, quote)
}

func (h *Router) createShipment(w http.ResponseWriter, r *http.Request) {
	var command domain.CreateShipmentCommand
	if !decodeJSON(w, r, &command) {
		return
	}
	if headerKey := strings.TrimSpace(r.Header.Get("Idempotency-Key")); headerKey != "" {
		if command.IdempotencyKey != "" && command.IdempotencyKey != headerKey {
			writeError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_MISMATCH", "header and body idempotency keys differ")
			return
		}
		command.IdempotencyKey = headerKey
	}
	shipment, err := h.service.CreateShipment(r.Context(), command)
	if err != nil {
		var providerErr *application.ProviderError
		if errors.As(err, &providerErr) {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": errorBody{Code: "CARRIER_REJECTED", Message: providerErr.Cause.Error()}, "shipment": shipment})
			return
		}
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, shipment)
}

func (h *Router) confirmHandover(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "shipmentID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SHIPMENT_ID", "shipmentID must be a UUID")
		return
	}
	shipment, err := h.service.ConfirmHandover(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, shipment)
}

type cancelShipmentRequest struct {
	ReasonCode string `json:"reason_code"`
	Reason     string `json:"reason"`
}

func (h *Router) cancelShipment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "shipmentID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SHIPMENT_ID", "shipmentID must be a UUID")
		return
	}
	var request cancelShipmentRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	shipment, err := h.service.CancelShipment(r.Context(), id, request.ReasonCode, request.Reason)
	if err != nil {
		var providerErr *application.ProviderError
		if errors.As(err, &providerErr) {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error":    errorBody{Code: "CARRIER_CANCEL_REJECTED", Message: providerErr.Cause.Error()},
				"shipment": shipment,
			})
			return
		}
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, shipment)
}

func (h *Router) getShipment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "shipmentID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SHIPMENT_ID", "shipmentID must be a UUID")
		return
	}
	shipment, timeline, err := h.service.GetShipment(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shipment": shipment, "tracking_events": timeline})
}

func (h *Router) listShipments(w http.ResponseWriter, r *http.Request) {
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	pageSize := parsePositiveInt(r.URL.Query().Get("pageSize"), 20)
	if pageSize > 100 {
		pageSize = 100
	}
	items, err := h.service.ListShipments(r.Context(), application.ListFilter{
		Status:      domain.ShipmentStatus(r.URL.Query().Get("status")),
		CarrierCode: strings.ToUpper(r.URL.Query().Get("carrier")), OrderID: r.URL.Query().Get("orderId"),
		Limit: pageSize, Offset: (page - 1) * pageSize,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "page": page, "page_size": pageSize})
}

type publicTracking struct {
	ShipmentID     uuid.UUID              `json:"shipment_id"`
	PackageID      string                 `json:"package_id"`
	CarrierCode    string                 `json:"carrier_code"`
	ServiceCode    string                 `json:"service_code"`
	TrackingCode   string                 `json:"tracking_code,omitempty"`
	Status         domain.ShipmentStatus  `json:"status"`
	EstimatedAt    *time.Time             `json:"estimated_delivery_at,omitempty"`
	DeliveredAt    *time.Time             `json:"delivered_at,omitempty"`
	TrackingEvents []domain.TrackingEvent `json:"tracking_events"`
}

func (h *Router) trackOrder(w http.ResponseWriter, r *http.Request) {
	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ORDER_ID", "orderID is required")
		return
	}
	shipments, err := h.service.TrackOrder(r.Context(), orderID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	result := make([]publicTracking, 0, len(shipments))
	for i := range shipments {
		timeline, timelineErr := h.service.GetShipmentTimeline(r.Context(), shipments[i].ID)
		if timelineErr != nil {
			h.handleError(w, timelineErr)
			return
		}
		result = append(result, publicTracking{
			ShipmentID: shipments[i].ID, PackageID: shipments[i].PackageID,
			CarrierCode: shipments[i].CarrierCode, ServiceCode: shipments[i].ServiceCode,
			TrackingCode: shipments[i].TrackingCode, Status: shipments[i].Status,
			EstimatedAt: shipments[i].EstimatedDeliveryAt, DeliveredAt: shipments[i].DeliveredAt,
			TrackingEvents: publicEvents(timeline),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"order_id": orderID, "shipments": result})
}

func publicEvents(events []domain.TrackingEvent) []domain.TrackingEvent {
	result := make([]domain.TrackingEvent, 0, len(events))
	for _, event := range events {
		if event.MappingResult != "APPLIED" {
			continue
		}
		event.InternalNote = ""
		result = append(result, event)
	}
	return result
}

func (h *Router) fakeWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "request body is invalid or too large")
		return
	}
	if !verifySignature(body, r.Header.Get("X-Shipping-Signature"), h.fakeWebhookSecret) {
		writeError(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "webhook signature is invalid")
		return
	}
	shipment, result, err := h.service.ApplyWebhook(r.Context(), carrier.FakeCode, body)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"accepted": true, "result": result, "shipment_id": shipment.ID, "status": shipment.Status})
}

func (h *Router) ghnWebhook(w http.ResponseWriter, r *http.Request) {
	if h.ghnWebhookSecret == "" {
		writeError(w, http.StatusNotFound, "GHN_DISABLED", "GHN webhook is not enabled")
		return
	}
	provided := strings.TrimSpace(r.Header.Get("X-GHN-Webhook-Secret"))
	if len(provided) != len(h.ghnWebhookSecret) || subtle.ConstantTimeCompare([]byte(provided), []byte(h.ghnWebhookSecret)) != 1 {
		writeError(w, http.StatusUnauthorized, "INVALID_WEBHOOK_SECRET", "webhook authentication failed")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "request body is invalid or too large")
		return
	}
	shipment, result, err := h.service.ApplyWebhook(r.Context(), "GHN", body)
	if err != nil {
		switch {
		case errors.Is(err, carrier.ErrInvalidWebhook):
			writeError(w, http.StatusBadRequest, "INVALID_GHN_WEBHOOK", err.Error())
		case errors.Is(err, domain.ErrTrackingNotFound), errors.Is(err, domain.ErrShipmentNotFound):
			// GHN can deliver the create callback before the local create transaction commits.
			// A retryable response prevents that callback from being dropped permanently.
			writeError(w, http.StatusServiceUnavailable, "SHIPMENT_NOT_READY", "shipment is not ready for webhook processing")
		default:
			h.logger.Error().Err(err).Msg("process GHN webhook")
			writeError(w, http.StatusInternalServerError, "WEBHOOK_PROCESSING_FAILED", "webhook processing failed")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"accepted": true, "result": result, "shipment_id": shipment.ID, "status": shipment.Status,
	})
}

func verifySignature(body []byte, signature, secret string) bool {
	signature = strings.TrimPrefix(strings.TrimSpace(signature), "sha256=")
	provided, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	return len(provided) == len(expected) && subtle.ConstantTimeCompare(provided, expected) == 1
}

func (h *Router) requireInternalToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token == "" {
			token = strings.TrimSpace(r.Header.Get("X-Internal-Token"))
		}
		if len(token) != len(h.internalToken) || subtle.ConstantTimeCompare([]byte(token), []byte(h.internalToken)) != 1 {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "valid internal authentication is required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Router) requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		h.logger.Info().Str("method", r.Method).Str("path", r.URL.Path).
			Str("request_id", middleware.GetReqID(r.Context())).Dur("duration", time.Since(started)).Msg("http request")
	})
}

func (h *Router) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrShipmentNotFound), errors.Is(err, domain.ErrTrackingNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrInvalidShipment), errors.Is(err, domain.ErrUnsupportedCarrier):
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", err.Error())
	case errors.Is(err, domain.ErrInvalidTransition), errors.Is(err, domain.ErrConcurrentUpdate):
		writeError(w, http.StatusConflict, "CONFLICT", err.Error())
	default:
		h.logger.Error().Err(err).Msg("request failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred")
	}
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": errorBody{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", fmt.Sprintf("invalid request body: %v", err))
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "request body must contain one JSON value")
		return false
	}
	return true
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
