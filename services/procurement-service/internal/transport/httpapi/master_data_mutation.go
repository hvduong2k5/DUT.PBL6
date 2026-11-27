package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/domain"
)

// Keep action-specific decoding: unrelated fields must not be silently accepted.
func (h *Router) masterMutation(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireMasterCreateKey(w, r) {
			return
		}
		parameter := "materialID"
		if action == "EDIT_SUPPLIER" || action == "SUPPLIER_STATUS" {
			parameter = "supplierID"
		}
		if action == "OFFER_STATUS" {
			parameter = "offerID"
		}
		id, ok := pathUUID(w, chi.URLParam(r, parameter))
		if !ok {
			return
		}
		cmd := domain.MasterDataMutation{Action: action, Actor: actorID(r), Key: r.Header.Get("Idempotency-Key"), ResourceID: id}
		var expected string
		switch action {
		case "EDIT_SUPPLIER":
			var input struct {
				supplierRequest
				Expected string `json:"expected_updated_at"`
			}
			if !decodeJSON(w, r, &input) {
				return
			}
			cmd.Supplier = &domain.Supplier{Code: input.Code, Name: input.Name, SupplierType: input.SupplierType, ContactName: input.ContactName, Phone: input.Phone, Email: input.Email, Address: input.Address, LegalIdentifier: input.LegalIdentifier}
			expected = input.Expected
		case "EDIT_MATERIAL":
			var input struct {
				materialRequest
				Expected string `json:"expected_updated_at"`
			}
			if !decodeJSON(w, r, &input) {
				return
			}
			cmd.Material = &domain.RawMaterial{Code: input.Code, Name: input.Name, BaseUnit: input.BaseUnit, Description: input.Description}
			expected = input.Expected
		case "SUPPLIER_STATUS", "MATERIAL_STATUS":
			var input struct {
				Status   string `json:"status"`
				Expected string `json:"expected_updated_at"`
			}
			if !decodeJSON(w, r, &input) {
				return
			}
			cmd.Status = input.Status
			expected = input.Expected
		case "PREFERRED_SUPPLIER":
			var input struct {
				SupplierID *string `json:"supplier_id"`
				Expected   string  `json:"expected_updated_at"`
			}
			if !decodeJSON(w, r, &input) {
				return
			}
			if input.SupplierID == nil {
				writeError(w, 400, "INVALID_ARGUMENT", "supplier_id is required; use an empty string to clear preference")
				return
			}
			if value := strings.TrimSpace(*input.SupplierID); value != "" {
				v, err := uuid.Parse(value)
				if err != nil || v == uuid.Nil {
					writeError(w, 400, "INVALID_ARGUMENT", "supplier_id must be a nonzero UUID or empty")
					return
				}
				cmd.PreferredSupplierID = v
			}
			expected = input.Expected
		case "OFFER_STATUS":
			parent, ok := pathUUID(w, chi.URLParam(r, "supplierID"))
			if !ok {
				return
			}
			cmd.SupplierID = parent
			var input struct {
				Active   *bool  `json:"active"`
				Expected string `json:"expected_updated_at"`
			}
			if !decodeJSON(w, r, &input) {
				return
			}
			cmd.Active = input.Active
			expected = input.Expected
		}
		stamp, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(expected))
		if err != nil || stamp.IsZero() {
			writeError(w, 400, "INVALID_ARGUMENT", "expected_updated_at must be the RFC3339 timestamp from the current resource")
			return
		}
		cmd.ExpectedUpdatedAt = stamp
		result, err := h.service.MutateMasterData(r.Context(), cmd)
		if err != nil {
			h.writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}
