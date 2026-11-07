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
		r.Get("/api/v1/admin/procurement/materials/{materialID}/offers", h.listMaterialOffers)
		r.Put("/api/v1/admin/procurement/materials/{materialID}", h.updateMaterial)
		r.Patch("/api/v1/admin/procurement/materials/{materialID}/status", h.setMaterialStatus)
		r.Put("/api/v1/admin/procurement/materials/{materialID}/preferred-supplier", h.setPreferredSupplier)
		r.Get("/api/v1/admin/procurement/suppliers/{supplierID}/offers", h.listSupplierOffers)
		r.Post("/api/v1/admin/procurement/suppliers/{supplierID}/offers", h.createSupplierOffer)
		r.Patch("/api/v1/admin/procurement/suppliers/{supplierID}/offers/{offerID}/status", h.setSupplierOfferStatus)
		r.Get("/api/v1/admin/procurement/purchase-orders", h.listPurchaseOrders)
		r.Get("/api/v1/admin/procurement/supply-plans", h.listSupplyPlans)
		r.Post("/api/v1/admin/procurement/supply-plans", h.createSupplyPlan)
		r.Post("/api/v1/admin/procurement/supply-plans/preview", h.previewSupplyPlan)
		r.Get("/api/v1/admin/procurement/supply-plans/{supplyPlanID}", h.getSupplyPlan)
		r.Get("/api/v1/admin/procurement/supply-plans/{supplyPlanID}/revisions", h.listSupplyPlanRevisions)
		r.Post("/api/v1/admin/procurement/supply-plans/{supplyPlanID}/revisions", h.reviseSupplyPlan)
		r.Post("/api/v1/admin/procurement/supply-plans/{supplyPlanID}/submit", h.submitSupplyPlan)
		r.Get("/api/v1/admin/procurement/supply-plans/{supplyPlanID}/status-history", h.listSupplyPlanStatusHistory)
		r.Post("/api/v1/admin/procurement/supply-plans/{supplyPlanID}/purchase-order", h.createPurchaseOrderFromSupplyPlan)
		r.Post("/api/v1/admin/procurement/purchase-orders", h.createPurchaseOrder)
		r.Get("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}", h.getPurchaseOrder)
		r.Put("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}", h.updatePurchaseOrderDraft)
		r.Post("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/submit", h.submitPurchaseOrder)
		r.Post("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/approve", h.approvePurchaseOrder)
		r.Post("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/reject", h.rejectPurchaseOrder)
		r.Post("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/send", h.markPurchaseOrderSent)
		r.Post("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/supplier-response", h.recordSupplierResponse)
		r.Get("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/revisions", h.listPurchaseOrderRevisions)
		r.Post("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/revisions", h.revisePurchaseOrder)
		r.Post("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/cancel", h.cancelPurchaseOrder)
		r.Post("/api/v1/admin/procurement/purchase-orders/{purchaseOrderID}/close", h.closePurchaseOrder)
	})
	r.Group(func(r chi.Router) {
		r.Use(h.requireReceivingStaff)
		r.Post("/api/v1/admin/procurement/goods-receipts", h.createGoodsReceipt)
		r.Get("/api/v1/admin/procurement/goods-receipts/{goodsReceiptID}", h.getGoodsReceipt)
		r.Post("/api/v1/admin/procurement/goods-receipts/{goodsReceiptID}/inspect", h.inspectGoodsReceipt)
	})
	r.Group(func(r chi.Router) {
		r.Use(h.requireSensitiveProcurementStaff)
		r.Get("/api/v1/admin/procurement/suppliers/{supplierID}/contracts", h.listSupplierContracts)
		r.Post("/api/v1/admin/procurement/suppliers/{supplierID}/contracts", h.createSupplierContract)
		r.Get("/api/v1/admin/procurement/contracts/{contractID}", h.getSupplierContract)
		r.Post("/api/v1/admin/procurement/contracts/{contractID}/activate", h.activateSupplierContract)
		r.Post("/api/v1/admin/procurement/contracts/{contractID}/terminate", h.terminateSupplierContract)
		r.Post("/api/v1/admin/procurement/goods-receipts/{goodsReceiptID}/variances/{varianceID}/resolve", h.resolveGoodsReceiptVariance)
		r.Post("/api/v1/admin/procurement/supply-plans/{supplyPlanID}/approve", h.approveSupplyPlan)
		r.Post("/api/v1/admin/procurement/supply-plans/{supplyPlanID}/reject", h.rejectSupplyPlan)
	})
	return r
}

type supplyPlanPreviewRequest struct {
	ExpectedVersion   int64                          `json:"expected_version,omitempty"`
	HorizonStart      string                         `json:"horizon_start"`
	HorizonEnd        string                         `json:"horizon_end"`
	ForecastSource    string                         `json:"forecast_source"`
	ForecastAsOf      string                         `json:"forecast_as_of"`
	InventorySource   string                         `json:"inventory_source"`
	InventoryAsOf     string                         `json:"inventory_as_of"`
	StaleAfterMinutes int                            `json:"stale_after_minutes"`
	PolicyVersion     string                         `json:"policy_version"`
	Lines             []supplyPlanPreviewLineRequest `json:"lines"`
}

type supplyPlanPreviewLineRequest struct {
	MaterialID             string `json:"material_id"`
	ForecastQuantity       string `json:"forecast_quantity"`
	OnHandQuantity         string `json:"on_hand_quantity"`
	OpenPOQuantity         string `json:"open_po_quantity"`
	SafetyStock            string `json:"safety_stock"`
	MinimumOrderQty        string `json:"minimum_order_quantity"`
	PackSize               string `json:"pack_size,omitempty"`
	LeadTimeDays           int    `json:"lead_time_days"`
	ManualOverrideQuantity string `json:"manual_override_quantity,omitempty"`
	ManualOverrideReason   string `json:"manual_override_reason,omitempty"`
}

func (h *Router) previewSupplyPlan(w http.ResponseWriter, r *http.Request) {
	input, expectedVersion, ok := h.decodeSupplyPlanInput(w, r)
	if !ok {
		return
	}
	if expectedVersion != 0 {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "expected_version is only accepted when creating a new plan revision")
		return
	}
	preview, err := h.service.PreviewSupplyPlan(r.Context(), input, time.Now().UTC())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (h *Router) createSupplyPlan(w http.ResponseWriter, r *http.Request) {
	input, expectedVersion, ok := h.decodeSupplyPlanInput(w, r)
	if !ok {
		return
	}
	if expectedVersion != 0 {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "expected_version is only accepted when creating a new plan revision")
		return
	}
	plan, err := h.service.CreateSupplyPlan(r.Context(), input, strings.TrimSpace(r.Header.Get("X-User-ID")), time.Now().UTC())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, plan)
}

func (h *Router) getSupplyPlan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "supplyPlanID"))
	if !ok {
		return
	}
	plan, err := h.service.GetSupplyPlan(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (h *Router) listSupplyPlans(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListSupplyPlans(r.Context(), limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (h *Router) reviseSupplyPlan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "supplyPlanID"))
	if !ok {
		return
	}
	input, expectedVersion, ok := h.decodeSupplyPlanInput(w, r)
	if !ok {
		return
	}
	if expectedVersion <= 0 {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "expected_version must be a positive integer")
		return
	}
	plan, err := h.service.ReviseSupplyPlan(r.Context(), id, expectedVersion, input, strings.TrimSpace(r.Header.Get("X-User-ID")), time.Now().UTC())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (h *Router) listSupplyPlanRevisions(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "supplyPlanID"))
	if !ok {
		return
	}
	items, err := h.service.ListSupplyPlanRevisions(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Router) submitSupplyPlan(w http.ResponseWriter, r *http.Request) {
	h.transitionSupplyPlan(w, r, func(id uuid.UUID, actor string) (any, error) {
		return h.service.SubmitSupplyPlan(r.Context(), id, actor, time.Now().UTC())
	})
}

func (h *Router) approveSupplyPlan(w http.ResponseWriter, r *http.Request) {
	h.transitionSupplyPlan(w, r, func(id uuid.UUID, actor string) (any, error) {
		return h.service.ApproveSupplyPlan(r.Context(), id, actor, time.Now().UTC())
	})
}

func (h *Router) rejectSupplyPlan(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Reason string `json:"reason"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	h.transitionSupplyPlan(w, r, func(id uuid.UUID, actor string) (any, error) {
		return h.service.RejectSupplyPlan(r.Context(), id, actor, request.Reason, time.Now().UTC())
	})
}

func (h *Router) transitionSupplyPlan(w http.ResponseWriter, r *http.Request, transition func(uuid.UUID, string) (any, error)) {
	id, ok := pathUUID(w, chi.URLParam(r, "supplyPlanID"))
	if !ok {
		return
	}
	result, err := transition(id, strings.TrimSpace(r.Header.Get("X-User-ID")))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Router) listSupplyPlanStatusHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "supplyPlanID"))
	if !ok {
		return
	}
	items, err := h.service.ListSupplyPlanStatusHistory(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Router) createPurchaseOrderFromSupplyPlan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "supplyPlanID"))
	if !ok {
		return
	}
	var request struct {
		SupplierID      string `json:"supplier_id"`
		ExpectedVersion int64  `json:"expected_version"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	supplierID, err := uuid.Parse(strings.TrimSpace(request.SupplierID))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "supplier_id must be a UUID")
		return
	}
	order, err := h.service.CreatePurchaseOrderFromSupplyPlan(r.Context(), id, supplierID, request.ExpectedVersion,
		strings.TrimSpace(r.Header.Get("X-User-ID")), time.Now().UTC())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (h *Router) decodeSupplyPlanInput(w http.ResponseWriter, r *http.Request) (domain.SupplyPlanInput, int64, bool) {
	var request supplyPlanPreviewRequest
	if !decodeJSON(w, r, &request) {
		return domain.SupplyPlanInput{}, 0, false
	}
	start, err := time.Parse("2006-01-02", request.HorizonStart)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "horizon_start must use YYYY-MM-DD")
		return domain.SupplyPlanInput{}, 0, false
	}
	end, err := time.Parse("2006-01-02", request.HorizonEnd)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "horizon_end must use YYYY-MM-DD")
		return domain.SupplyPlanInput{}, 0, false
	}
	forecastAsOf, err := time.Parse(time.RFC3339, request.ForecastAsOf)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "forecast_as_of must use RFC3339")
		return domain.SupplyPlanInput{}, 0, false
	}
	inventoryAsOf, err := time.Parse(time.RFC3339, request.InventoryAsOf)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "inventory_as_of must use RFC3339")
		return domain.SupplyPlanInput{}, 0, false
	}
	input := domain.SupplyPlanInput{HorizonStart: start, HorizonEnd: end, ForecastSource: request.ForecastSource, ForecastAsOf: forecastAsOf,
		InventorySource: request.InventorySource, InventoryAsOf: inventoryAsOf, StaleAfterMinutes: request.StaleAfterMinutes,
		PolicyVersion: request.PolicyVersion, Lines: make([]domain.SupplyPlanLineInput, 0, len(request.Lines))}
	for _, line := range request.Lines {
		materialID, parseErr := uuid.Parse(strings.TrimSpace(line.MaterialID))
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "material_id must be a UUID")
			return domain.SupplyPlanInput{}, 0, false
		}
		input.Lines = append(input.Lines, domain.SupplyPlanLineInput{MaterialID: materialID, ForecastQuantity: line.ForecastQuantity,
			OnHandQuantity: line.OnHandQuantity, OpenPOQuantity: line.OpenPOQuantity, SafetyStock: line.SafetyStock,
			MinimumOrderQty: line.MinimumOrderQty, PackSize: line.PackSize, LeadTimeDays: line.LeadTimeDays,
			ManualOverrideQuantity: line.ManualOverrideQuantity, ManualOverrideReason: line.ManualOverrideReason})
	}
	return input, request.ExpectedVersion, true
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
	return h.requireRoles(next, "ADMIN", "SUPPLY_MANAGER", "PROCUREMENT_MANAGER")
}

func (h *Router) requireReceivingStaff(next http.Handler) http.Handler {
	return h.requireRoles(next, "ADMIN", "SUPPLY_MANAGER", "PROCUREMENT_MANAGER", "WAREHOUSE_STAFF")
}

func (h *Router) requireSensitiveProcurementStaff(next http.Handler) http.Handler {
	return h.requireRoles(next, "ADMIN", "PROCUREMENT_MANAGER")
}

func (h *Router) requireRoles(next http.Handler, allowed ...string) http.Handler {
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
		for _, candidate := range allowed {
			if role == candidate {
				next.ServeHTTP(w, r)
				return
			}
		}
		writeError(w, http.StatusForbidden, "FORBIDDEN", "role cannot access this procurement operation")
	})
}

type goodsReceiptLineRequest struct {
	PurchaseOrderLineID string `json:"purchase_order_line_id"`
	QuantityReceived    string `json:"quantity_received"`
	SupplierLotCode     string `json:"supplier_lot_code,omitempty"`
	ManufacturedOn      string `json:"manufactured_on,omitempty"`
	ExpiresOn           string `json:"expires_on,omitempty"`
}

type goodsReceiptRequest struct {
	PurchaseOrderID string                    `json:"purchase_order_id"`
	WarehouseCode   string                    `json:"warehouse_code"`
	Notes           string                    `json:"notes,omitempty"`
	Lines           []goodsReceiptLineRequest `json:"lines"`
}

func (h *Router) createGoodsReceipt(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if !validIdempotencyKey(idempotencyKey) {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Idempotency-Key must contain 16-128 visible ASCII characters")
		return
	}
	var request goodsReceiptRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	poID, err := uuid.Parse(strings.TrimSpace(request.PurchaseOrderID))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "purchase_order_id must be a UUID")
		return
	}
	input := application.GoodsReceiptRequest{PurchaseOrderID: poID, WarehouseCode: request.WarehouseCode, Notes: request.Notes, Lines: make([]application.GoodsReceiptLineRequest, 0, len(request.Lines))}
	for _, line := range request.Lines {
		lineID, parseErr := uuid.Parse(strings.TrimSpace(line.PurchaseOrderLineID))
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "purchase_order_line_id must be a UUID")
			return
		}
		item := application.GoodsReceiptLineRequest{PurchaseOrderLineID: lineID, QuantityReceived: line.QuantityReceived, SupplierLotCode: line.SupplierLotCode}
		if line.ManufacturedOn != "" {
			date, parseErr := time.Parse("2006-01-02", line.ManufacturedOn)
			if parseErr != nil {
				writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "manufactured_on must use YYYY-MM-DD")
				return
			}
			item.ManufacturedOn = &date
		}
		if line.ExpiresOn != "" {
			date, parseErr := time.Parse("2006-01-02", line.ExpiresOn)
			if parseErr != nil {
				writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "expires_on must use YYYY-MM-DD")
				return
			}
			item.ExpiresOn = &date
		}
		input.Lines = append(input.Lines, item)
	}
	created, err := h.service.CreateGoodsReceipt(r.Context(), input, strings.TrimSpace(r.Header.Get("X-User-ID")), idempotencyKey)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func validIdempotencyKey(value string) bool {
	if len(value) < 16 || len(value) > 128 {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < 0x21 || value[i] > 0x7e {
			return false
		}
	}
	return true
}

func (h *Router) getGoodsReceipt(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "goodsReceiptID"))
	if !ok {
		return
	}
	receipt, err := h.service.GetGoodsReceipt(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

func (h *Router) resolveGoodsReceiptVariance(w http.ResponseWriter, r *http.Request) {
	receiptID, ok := pathUUID(w, chi.URLParam(r, "goodsReceiptID"))
	if !ok {
		return
	}
	varianceID, ok := pathUUID(w, chi.URLParam(r, "varianceID"))
	if !ok {
		return
	}
	var request struct {
		Decision string `json:"decision"`
		Note     string `json:"note"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	receipt, err := h.service.ResolveGoodsReceiptVariance(r.Context(), receiptID, varianceID, request.Decision, strings.TrimSpace(r.Header.Get("X-User-ID")), request.Note)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

func (h *Router) inspectGoodsReceipt(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "goodsReceiptID"))
	if !ok {
		return
	}
	var request struct {
		Lines []struct {
			GoodsReceiptLineID string `json:"goods_receipt_line_id"`
			QuantityAccepted   string `json:"quantity_accepted"`
			QuantityRejected   string `json:"quantity_rejected"`
			HumidityPercent    string `json:"humidity_percent,omitempty"`
			InspectionNotes    string `json:"inspection_notes,omitempty"`
		} `json:"lines"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	lines := make([]domain.GoodsReceiptInspectionLine, 0, len(request.Lines))
	for _, line := range request.Lines {
		lineID, err := uuid.Parse(strings.TrimSpace(line.GoodsReceiptLineID))
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "goods_receipt_line_id must be a UUID")
			return
		}
		lines = append(lines, domain.GoodsReceiptInspectionLine{GoodsReceiptLineID: lineID, QuantityAccepted: line.QuantityAccepted, QuantityRejected: line.QuantityRejected, HumidityPercent: line.HumidityPercent, InspectionNotes: line.InspectionNotes})
	}
	result, err := h.service.InspectGoodsReceipt(r.Context(), id, strings.TrimSpace(r.Header.Get("X-User-ID")), lines)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
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
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if !validIdempotencyKey(idempotencyKey) {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Idempotency-Key must contain 16-128 visible ASCII characters")
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
	order, err := h.service.CreatePurchaseOrderWithIdempotency(r.Context(), input, strings.TrimSpace(r.Header.Get("X-User-ID")), idempotencyKey)
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

func (h *Router) recordSupplierResponse(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Response domain.SupplierPOResponse `json:"response"`
		Note     string                    `json:"note,omitempty"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	h.transitionPurchaseOrder(w, r, func(id uuid.UUID, actor, _ string) (any, error) {
		return h.service.RecordSupplierResponse(r.Context(), id, actor, request.Response, request.Note)
	})
}

type purchaseOrderRevisionRequest struct {
	SupplierID           string                     `json:"supplier_id"`
	ExpectedDeliveryDate string                     `json:"expected_delivery_date,omitempty"`
	Notes                string                     `json:"notes,omitempty"`
	Lines                []purchaseOrderLineRequest `json:"lines"`
	Reason               string                     `json:"reason"`
}

func (h *Router) revisePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "purchaseOrderID"))
	if !ok {
		return
	}
	var request purchaseOrderRevisionRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	input, err := (purchaseOrderRequest{SupplierID: request.SupplierID, ExpectedDeliveryDate: request.ExpectedDeliveryDate, Notes: request.Notes, Lines: request.Lines}).applicationInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error())
		return
	}
	updated, err := h.service.RevisePurchaseOrder(r.Context(), id, input, strings.TrimSpace(r.Header.Get("X-User-ID")), request.Reason)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Router) listPurchaseOrderRevisions(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "purchaseOrderID"))
	if !ok {
		return
	}
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListPurchaseOrderRevisions(r.Context(), id, limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
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

func (h *Router) closePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Reason string `json:"reason"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	h.transitionPurchaseOrder(w, r, func(id uuid.UUID, actor, reason string) (any, error) {
		return h.service.CloseRemainingPurchaseOrder(r.Context(), id, actor, reason)
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
	Code            string `json:"code"`
	Name            string `json:"name"`
	SupplierType    string `json:"supplier_type"`
	ContactName     string `json:"contact_name"`
	Phone           string `json:"phone"`
	Email           string `json:"email"`
	Address         string `json:"address"`
	LegalIdentifier string `json:"legal_identifier,omitempty"`
}

func (h *Router) createSupplier(w http.ResponseWriter, r *http.Request) {
	var request supplierRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	supplier, err := domain.NewSupplierWithLegalIdentifier(request.Code, request.Name, request.SupplierType, request.ContactName, request.Phone, request.Email, request.Address, request.LegalIdentifier)
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
	supplier, err := domain.NewSupplierWithLegalIdentifier(request.Code, request.Name, request.SupplierType, request.ContactName, request.Phone, request.Email, request.Address, request.LegalIdentifier)
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

func (h *Router) updateMaterial(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "materialID"))
	if !ok {
		return
	}
	var request materialRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	material, err := domain.NewRawMaterial(request.Code, request.Name, request.BaseUnit, request.Description)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "code, name and base_unit are required")
		return
	}
	updated, err := h.service.UpdateMaterial(r.Context(), id, material)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Router) setPreferredSupplier(w http.ResponseWriter, r *http.Request) {
	materialID, ok := pathUUID(w, chi.URLParam(r, "materialID"))
	if !ok {
		return
	}
	var request struct {
		SupplierID string `json:"supplier_id"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	var supplierID uuid.UUID
	if value := strings.TrimSpace(request.SupplierID); value != "" {
		var err error
		supplierID, err = uuid.Parse(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "supplier_id must be a UUID or empty to clear preference")
			return
		}
	}
	material, err := h.service.SetPreferredSupplier(r.Context(), materialID, supplierID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, material)
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
	EffectiveFrom   string `json:"effective_from,omitempty"`
	EffectiveTo     string `json:"effective_to,omitempty"`
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
	effectiveFrom := time.Now().UTC()
	if value := strings.TrimSpace(request.EffectiveFrom); value != "" {
		date, parseErr := time.Parse("2006-01-02", value)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "effective_from must use YYYY-MM-DD")
			return
		}
		effectiveFrom = date.UTC()
	}
	var effectiveTo *time.Time
	if value := strings.TrimSpace(request.EffectiveTo); value != "" {
		date, parseErr := time.Parse("2006-01-02", value)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "effective_to must use YYYY-MM-DD")
			return
		}
		end := date.AddDate(0, 0, 1).UTC()
		effectiveTo = &end
	}
	offer, err := domain.NewSupplierMaterialOfferVersion(supplierID, materialID, request.SupplierSKU, request.Unit, request.UnitPriceMinor, request.Currency, request.MinimumQuantity, request.LeadTimeDays, effectiveFrom, effectiveTo)
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

func (h *Router) setSupplierOfferStatus(w http.ResponseWriter, r *http.Request) {
	supplierID, ok := pathUUID(w, chi.URLParam(r, "supplierID"))
	if !ok {
		return
	}
	offerID, ok := pathUUID(w, chi.URLParam(r, "offerID"))
	if !ok {
		return
	}
	var request struct {
		Active *bool `json:"active"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.Active == nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "active is required")
		return
	}
	item, err := h.service.SetOfferActive(r.Context(), supplierID, offerID, *request.Active)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
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

func (h *Router) listMaterialOffers(w http.ResponseWriter, r *http.Request) {
	materialID, ok := pathUUID(w, chi.URLParam(r, "materialID"))
	if !ok {
		return
	}
	items, err := h.service.ListMaterialOffers(r.Context(), materialID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type supplierContractRequest struct {
	ContractNumber string `json:"contract_number"`
	Scope          string `json:"scope"`
	PaymentTerms   string `json:"payment_terms"`
	DocumentFileID string `json:"document_file_id,omitempty"`
	EffectiveFrom  string `json:"effective_from"`
	EffectiveTo    string `json:"effective_to,omitempty"`
}

func (h *Router) createSupplierContract(w http.ResponseWriter, r *http.Request) {
	supplierID, ok := pathUUID(w, chi.URLParam(r, "supplierID"))
	if !ok {
		return
	}
	var request supplierContractRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	from, err := time.Parse("2006-01-02", strings.TrimSpace(request.EffectiveFrom))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "effective_from must use YYYY-MM-DD")
		return
	}
	var to *time.Time
	if value := strings.TrimSpace(request.EffectiveTo); value != "" {
		date, parseErr := time.Parse("2006-01-02", value)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "effective_to must use YYYY-MM-DD")
			return
		}
		end := date.AddDate(0, 0, 1).UTC()
		to = &end
	}
	contract, err := domain.NewSupplierContract(supplierID, request.ContractNumber, request.Scope, request.PaymentTerms, request.DocumentFileID, from, to, strings.TrimSpace(r.Header.Get("X-User-ID")))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	created, err := h.service.CreateSupplierContract(r.Context(), contract)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Router) listSupplierContracts(w http.ResponseWriter, r *http.Request) {
	supplierID, ok := pathUUID(w, chi.URLParam(r, "supplierID"))
	if !ok {
		return
	}
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListSupplierContracts(r.Context(), supplierID, limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (h *Router) getSupplierContract(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, chi.URLParam(r, "contractID"))
	if !ok {
		return
	}
	item, err := h.service.GetSupplierContract(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Router) activateSupplierContract(w http.ResponseWriter, r *http.Request) {
	h.changeSupplierContractStatus(w, r, domain.SupplierContractActive)
}
func (h *Router) terminateSupplierContract(w http.ResponseWriter, r *http.Request) {
	h.changeSupplierContractStatus(w, r, domain.SupplierContractTerminated)
}
func (h *Router) changeSupplierContractStatus(w http.ResponseWriter, r *http.Request, status domain.SupplierContractStatus) {
	id, ok := pathUUID(w, chi.URLParam(r, "contractID"))
	if !ok {
		return
	}
	item, err := h.service.SetSupplierContractStatus(r.Context(), id, status, strings.TrimSpace(r.Header.Get("X-User-ID")))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
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
