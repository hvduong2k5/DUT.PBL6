package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
)

type Router struct {
	service       *application.Service
	internalToken string
	ping          func(context.Context) error
}

func NewRouter(service *application.Service, internalToken string, ping func(context.Context) error) http.Handler {
	h := &Router{service: service, internalToken: internalToken, ping: ping}
	r := chi.NewRouter()
	r.Get("/health/live", h.live)
	r.Get("/health/ready", h.ready)
	r.Group(func(r chi.Router) {
		r.Use(h.requireInternalStaff)
		r.Get("/api/v1/admin/procurement/suppliers", h.listSuppliers)
		r.Post("/api/v1/admin/procurement/suppliers", h.createSupplier)
		r.Get("/api/v1/admin/procurement/suppliers/{supplierID}", h.getSupplier)
		r.Put("/api/v1/admin/procurement/suppliers/{supplierID}", h.updateSupplier)
		r.Patch("/api/v1/admin/procurement/suppliers/{supplierID}/status", h.setSupplierStatus)
		r.Get("/api/v1/admin/procurement/materials", h.listMaterials)
		r.Post("/api/v1/admin/procurement/materials", h.createMaterial)
		r.Get("/api/v1/admin/procurement/materials/{materialID}", h.getMaterial)
		r.Patch("/api/v1/admin/procurement/materials/{materialID}/status", h.setMaterialStatus)
		r.Get("/api/v1/admin/procurement/suppliers/{supplierID}/offers", h.listSupplierOffers)
		r.Post("/api/v1/admin/procurement/suppliers/{supplierID}/offers", h.createSupplierOffer)
		r.Get("/api/v1/admin/procurement/purchase-orders", h.listPurchaseOrders)
		r.Post("/api/v1/admin/procurement/purchase-orders", h.createPurchaseOrder)
		r.Get("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}", h.getPurchaseOrder)
		r.Put("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}", h.updatePurchaseOrderDraft)
		r.Post("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/submit", h.submitPurchaseOrder)
		r.Post("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/approve", h.approvePurchaseOrder)
		r.Post("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/reject", h.rejectPurchaseOrder)
		r.Post("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/send", h.markPurchaseOrderSent)
		r.Post("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/cancel", h.cancelPurchaseOrder)
	})
	return r
}

func (h *Router) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Router) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if h.ping == nil || h.ping(ctx) != nil {
		writeError(w, http.StatusServiceUnavailable, "NOT_READY", "database is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Router) requireInternalStaff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(r.Header.Get("X-Internal-Token"))
		if token == "" || len(token) != len(h.internalToken) || subtle.ConstantTimeCompare([]byte(token), []byte(h.internalToken)) != 1 {
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "valid internal token is required")
			return
		}
		if strings.TrimSpace(r.Header.Get("X-User-ID")) == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authenticated user context is required")
			return
		}
		role := strings.ToUpper(strings.TrimSpace(r.Header.Get("X-User-Role")))
		switch role {
		case "ADMIN", "SUPPLY_MANAGER", "PROCUREMENT_MANAGER":
			next.ServeHTTP(w, r)
		default:
			writeError(w, http.StatusForbidden, "FORBIDDEN", "role cannot access procurement administration")
		}
	})
}

type purchaseOrderLineRequest struct {
	MaterialID string `json:"material_id"`
	Quantity   string `json:"quantity"`
}

type purchaseOrderRequest struct {
	SupplierID           string                     `json:"supplier_id"`
	ExpectedDeliveryDate string                     `json:"expected_delivery_date,omitempty"`
	Notes                string                     `json:"notes,omitempty"`
	Lines                []purchaseOrderLineRequest `json:"lines"`
}

func (h *Router) createPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	request, ok := decodePurchaseOrderRequest(w, r)
	if !ok {
		return
	}
	input, err := request.applicationInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error())
		return
	}
	order, err := h.service.CreatePurchaseOrder(r.Context(), input, strings.TrimSpace(r.Header.Get("X-User-ID")))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (h *Router) updatePurchaseOrderDraft(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "purchaseOrderID"))
	if !ok {
		return
	}
	request, ok := decodePurchaseOrderRequest(w, r)
	if !ok {
		return
	}
	input, err := request.applicationInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error())
		return
	}
	order, err := h.service.UpdatePurchaseOrderDraft(r.Context(), id, input, strings.TrimSpace(r.Header.Get("X-User-ID")))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func decodePurchaseOrderRequest(w http.ResponseWriter, r *http.Request) (purchaseOrderRequest, bool) {
	var request purchaseOrderRequest
	if !decodeJSON(w, r, &request) {
		return purchaseOrderRequest{}, false
	}
	return request, true
}

func (request purchaseOrderRequest) applicationInput() (application.PurchaseOrderRequest, error) {
	supplierID, err := uuid.Parse(strings.TrimSpace(request.SupplierID))
	if err != nil {
		return application.PurchaseOrderRequest{}, errors.New("supplier_id must be a UUID")
	}
	input := application.PurchaseOrderRequest{SupplierID: supplierID, Notes: request.Notes}
	if value := strings.TrimSpace(request.ExpectedDeliveryDate); value != "" {
		date, err := time.Parse("2006-01-02", value)
		if err != nil {
			return application.PurchaseOrderRequest{}, errors.New("expected_delivery_date must use YYYY-MM-DD")
		}
		input.ExpectedDeliveryDate = &date
	}
	input.Lines = make([]application.PurchaseOrderLineRequest, 0, len(request.Lines))
	for _, line := range request.Lines {
		materialID, err := uuid.Parse(strings.TrimSpace(line.MaterialID))
		if err != nil {
			return application.PurchaseOrderRequest{}, errors.New("each material_id must be a UUID")
		}
		input.Lines = append(input.Lines, application.PurchaseOrderLineRequest{MaterialID: materialID, Quantity: strings.TrimSpace(line.Quantity)})
	}
	return input, nil
}

func (h *Router) listPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	var supplierID uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("supplier_id")); raw != "" {
		var err error
		supplierID, err = uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "supplier_id must be a UUID")
			return
		}
	}
	status := domain.PurchaseOrderStatus(strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status"))))
	items, err := h.service.ListPurchaseOrders(r.Context(), supplierID, status, limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (h *Router) getPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "purchaseOrderID"))
	if !ok {
		return
	}
	order, err := h.service.GetPurchaseOrder(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *Router) submitPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	h.transitionPurchaseOrder(w, r, func(id uuid.UUID, actor, _ string) (any, error) {
		return h.service.SubmitPurchaseOrder(r.Context(), id, actor)
	})
}

func (h *Router) approvePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	h.transitionPurchaseOrder(w, r, func(id uuid.UUID, actor, _ string) (any, error) {
		return h.service.ApprovePurchaseOrder(r.Context(), id, actor)
	})
}

func (h *Router) rejectPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Reason string `json:"reason"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	h.transitionPurchaseOrder(w, r, func(id uuid.UUID, actor, _ string) (any, error) {
		return h.service.RejectPurchaseOrder(r.Context(), id, actor, request.Reason)
	})
}

func (h *Router) markPurchaseOrderSent(w http.ResponseWriter, r *http.Request) {
	h.transitionPurchaseOrder(w, r, func(id uuid.UUID, actor, _ string) (any, error) {
		return h.service.MarkPurchaseOrderSent(r.Context(), id, actor)
	})
}

func (h *Router) cancelPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Reason string `json:"reason"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	h.transitionPurchaseOrder(w, r, func(id uuid.UUID, actor, reason string) (any, error) {
		return h.service.CancelPurchaseOrder(r.Context(), id, actor, reason)
	}, request.Reason)
}

type purchaseOrderTransition func(uuid.UUID, string, string) (any, error)

func (h *Router) transitionPurchaseOrder(w http.ResponseWriter, r *http.Request, transition purchaseOrderTransition, reasons ...string) {
	id, ok := pathUUID(w, chi.URLParam(r, "purchaseOrderID"))
	if !ok {
		return
	}
	reason := ""
	if len(reasons) > 0 {
		reason = reasons[0]
	}
	order, err := transition(id, strings.TrimSpace(r.Header.Get("X-User-ID")), reason)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

type supplierRequest struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	SupplierType string `json:"supplier_type"`
	ContactName  string `json:"contact_name"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Address      string `json:"address"`
}

func (h *Router) createSupplier(w http.ResponseWriter, r *http.Request) {
	var request supplierRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	supplier, err := domain.NewSupplier(request.Code, request.Name, request.SupplierType, request.ContactName, request.Phone, request.Email, request.Address)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "code, name and a supported supplier_type are required")
		return
	}
	created, err := h.service.CreateSupplier(r.Context(), supplier)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Router) listSuppliers(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListSuppliers(r.Context(), limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (h *Router) getSupplier(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "supplierID"))
	if !ok {
		return
	}
	item, err := h.service.GetSupplier(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Router) updateSupplier(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "supplierID"))
	if !ok {
		return
	}
	var request supplierRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	supplier, err := domain.NewSupplier(request.Code, request.Name, request.SupplierType, request.ContactName, request.Phone, request.Email, request.Address)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "code, name and a supported supplier_type are required")
		return
	}
	updated, err := h.service.UpdateSupplier(r.Context(), id, supplier)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Router) setSupplierStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "supplierID"))
	if !ok {
		return
	}
	var request struct {
		Status domain.SupplierStatus `json:"status"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.Status != domain.SupplierActive && request.Status != domain.SupplierSuspended {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "status must be ACTIVE or SUSPENDED")
		return
	}
	item, err := h.service.SetSupplierStatus(r.Context(), id, request.Status)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

type materialRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	BaseUnit    string `json:"base_unit"`
	Description string `json:"description"`
}

func (h *Router) createMaterial(w http.ResponseWriter, r *http.Request) {
	var request materialRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	material, err := domain.NewRawMaterial(request.Code, request.Name, request.BaseUnit, request.Description)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "code, name and base_unit are required")
		return
	}
	created, err := h.service.CreateMaterial(r.Context(), material)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Router) listMaterials(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListMaterials(r.Context(), limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (h *Router) getMaterial(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "materialID"))
	if !ok {
		return
	}
	item, err := h.service.GetMaterial(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Router) setMaterialStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "materialID"))
	if !ok {
		return
	}
	var request struct {
		Status domain.MaterialStatus `json:"status"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.Status != domain.MaterialActive && request.Status != domain.MaterialInactive {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "status must be ACTIVE or INACTIVE")
		return
	}
	item, err := h.service.SetMaterialStatus(r.Context(), id, request.Status)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

type offerRequest struct {
	MaterialID      string `json:"material_id"`
	SupplierSKU     string `json:"supplier_sku"`
	Unit            string `json:"unit"`
	UnitPriceMinor  int64  `json:"unit_price_minor"`
	Currency        string `json:"currency"`
	MinimumQuantity string `json:"minimum_quantity"`
	LeadTimeDays    int    `json:"lead_time_days"`
}

func (h *Router) createSupplierOffer(w http.ResponseWriter, r *http.Request) {
	supplierID, ok := pathUUID(w, chi.URLParam(r, "supplierID"))
	if !ok {
		return
	}
	var request offerRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	materialID, err := uuid.Parse(strings.TrimSpace(request.MaterialID))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "material_id must be a UUID")
		return
	}
	offer, err := domain.NewSupplierMaterialOffer(supplierID, materialID, request.SupplierSKU, request.Unit, request.UnitPriceMinor, request.Currency, request.MinimumQuantity, request.LeadTimeDays)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "offer fields are invalid")
		return
	}
	created, err := h.service.CreateOffer(r.Context(), offer)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Router) listSupplierOffers(w http.ResponseWriter, r *http.Request) {
	supplierID, ok := pathUUID(w, chi.URLParam(r, "supplierID"))
	if !ok {
		return
	}
	items, err := h.service.ListSupplierOffers(r.Context(), supplierID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Router) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "resource was not found")
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, "CONFLICT", "resource already exists or the action conflicts with current data")
	case errors.Is(err, domain.ErrInvalidState):
		writeError(w, http.StatusConflict, "INVALID_STATE", "the requested operation is not allowed in the current state")
	case errors.Is(err, domain.ErrInvalidArgument):
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "request contains invalid or missing fields")
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL", "request could not be completed")
	}
}

func pagination(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	limit, offset := 50, 0
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
	}
	if err == nil {
		if raw := r.URL.Query().Get("offset"); raw != "" {
			offset, err = strconv.Atoi(raw)
		}
	}
	if err != nil || limit < 1 || limit > 100 || offset < 0 {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "limit must be 1..100 and offset must be non-negative")
		return 0, 0, false
	}
	return limit, offset, true
}

func pathUUID(w http.ResponseWriter, raw string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "resource ID must be a UUID")
		return uuid.Nil, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "request body must contain a valid JSON object")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "request body must contain one JSON value")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
