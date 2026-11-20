package httpapi

import (
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
	"strconv"
)

func (h *Router) listDriverShipments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	p, ok := h.internalDeliveryActor(w, r, "SHIPMENT_EXECUTE", false)
	if !ok {
		return
	}
	limit, offset := 50, 0
	for key, values := range r.URL.Query() {
		if (key != "limit" && key != "offset") || len(values) != 1 {
			writeError(w, 400, "INVALID_REQUEST", "only one limit and offset are supported")
			return
		}
		value, err := strconv.Atoi(values[0])
		if err != nil {
			writeError(w, 400, "INVALID_REQUEST", "pagination must be integers")
			return
		}
		if key == "limit" {
			limit = value
		} else {
			offset = value
		}
	}
	if limit < 1 || limit > 100 || offset < 0 || offset > 10000 {
		writeError(w, 400, "INVALID_REQUEST", "limit must be 1..100 and offset 0..10000")
		return
	}
	actor, _ := uuid.Parse(p.UserID)
	result, err := h.service.ListDriverShipments(r.Context(), actor, limit, offset)
	if err != nil {
		h.assignmentError(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (h *Router) getDriverShipment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	p, ok := h.internalDeliveryActor(w, r, "SHIPMENT_EXECUTE", false)
	if !ok {
		return
	}
	if len(r.URL.Query()) != 0 {
		writeError(w, 400, "INVALID_REQUEST", "query overrides are not supported")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "shipmentID"))
	if err != nil || id == uuid.Nil {
		writeError(w, 400, "INVALID_SHIPMENT_ID", "shipmentID must be a UUID")
		return
	}
	actor, _ := uuid.Parse(p.UserID)
	result, err := h.service.GetDriverShipment(r.Context(), actor, id)
	if err != nil {
		h.assignmentError(w, err)
		return
	}
	writeJSON(w, 200, result)
}
