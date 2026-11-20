package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/omamx/shipping-service/internal/domain"
)

type permissionAuthenticator interface {
	AuthenticatePermission(context.Context, string, string) (StaffPrincipal, error)
}

func (h *Router) assignmentActor(w http.ResponseWriter, r *http.Request, mutation bool) (StaffPrincipal, bool) {
	return h.internalDeliveryActor(w, r, "SHIPMENT_ASSIGN", mutation)
}

func (h *Router) internalDeliveryActor(w http.ResponseWriter, r *http.Request, permission string, mutation bool) (StaffPrincipal, bool) {
	auth, ok := h.staffAuth.(permissionAuthenticator)
	if !ok {
		writeError(w, 401, "UNAUTHORIZED", "valid employee authentication is required")
		return StaffPrincipal{}, false
	}
	p, err := auth.AuthenticatePermission(r.Context(), r.Header.Get("Authorization"), permission)
	if err != nil {
		if errors.Is(err, errStaffForbidden) {
			writeError(w, 403, "FORBIDDEN", "required global shipment permission is missing")
		} else {
			writeError(w, 401, "UNAUTHORIZED", "valid employee authentication is required")
		}
		return StaffPrincipal{}, false
	}
	if mutation {
		found := false
		for _, g := range p.PermissionGrants {
			if g == "USER_VIEW#GLOBAL" {
				found = true
			}
		}
		if !found {
			writeError(w, 403, "FORBIDDEN", "global employee directory read permission is required")
			return StaffPrincipal{}, false
		}
	}
	id, err := uuid.Parse(p.UserID)
	if err != nil || id == uuid.Nil {
		writeError(w, 401, "UNAUTHORIZED", "valid employee authentication is required")
		return StaffPrincipal{}, false
	}
	if err := h.assignmentDirectory.ValidateActor(r.Context(), r.Header.Get("Authorization"), id, p.Subject); err != nil {
		h.assignmentError(w, err)
		return StaffPrincipal{}, false
	}
	return p, true
}

func (h *Router) assignmentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrDriverNotAssigned):
		writeError(w, 403, "FORBIDDEN", "delivery is not assigned to this employee")
	case errors.Is(err, application.ErrEmployeeUnauthenticated):
		writeError(w, 401, "UNAUTHORIZED", "employee authentication was rejected")
	case errors.Is(err, application.ErrEmployeeForbidden):
		writeError(w, 403, "FORBIDDEN", "active employee authorization is required")
	case errors.Is(err, application.ErrDriverIneligible):
		writeError(w, 422, "DRIVER_INELIGIBLE", "driver must be an active employee")
	case errors.Is(err, application.ErrEmployeeDirectoryUnavailable):
		writeError(w, 503, "IDENTITY_UNAVAILABLE", "employee directory is temporarily unavailable")
	case errors.Is(err, domain.ErrAssignmentNotFound):
		writeError(w, 404, "NOT_FOUND", "shipment has no internal delivery assignment")
	default:
		h.handleError(w, err)
	}
}

func (h *Router) acceptInternalDelivery(w http.ResponseWriter, r *http.Request) {
	p, ok := h.internalDeliveryActor(w, r, "SHIPMENT_EXECUTE", false)
	if !ok {
		return
	}
	key, ok := requireIdempotencyKey(w, r)
	if !ok {
		return
	}
	id, ok := h.internalAssignmentTarget(w, r)
	if !ok {
		return
	}
	var body struct {
		ExpectedVersion *int64 `json:"expected_version"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.ExpectedVersion == nil {
		writeError(w, 400, "INVALID_REQUEST", "expected_version is required")
		return
	}
	actor, _ := uuid.Parse(p.UserID)
	result, err := h.service.AcceptInternalDelivery(r.Context(), domain.AcceptInternalDelivery{ShipmentID: id, ActorID: actor, ExpectedVersion: *body.ExpectedVersion, IdempotencyKey: key})
	if err != nil {
		h.assignmentError(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (h *Router) internalAssignmentTarget(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "shipmentID"))
	if err != nil || id == uuid.Nil {
		writeError(w, 400, "INVALID_SHIPMENT_ID", "shipmentID must be a UUID")
		return uuid.Nil, false
	}
	s, _, err := h.service.GetShipment(r.Context(), id)
	if err != nil {
		h.assignmentError(w, err)
		return uuid.Nil, false
	}
	if s.CarrierCode != carrier.InternalCode {
		writeError(w, 422, "VALIDATION_FAILED", "assignment requires an INTERNAL shipment")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Router) assignInternalDelivery(w http.ResponseWriter, r *http.Request) {
	p, ok := h.assignmentActor(w, r, true)
	if !ok {
		return
	}
	key, ok := requireIdempotencyKey(w, r)
	if !ok {
		return
	}
	id, ok := h.internalAssignmentTarget(w, r)
	if !ok {
		return
	}
	var body struct {
		DriverID        uuid.UUID `json:"driver_id"`
		ExpectedVersion *int64    `json:"expected_version"`
		Reason          string    `json:"reason"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.ExpectedVersion == nil {
		writeError(w, 400, "INVALID_REQUEST", "expected_version is required")
		return
	}
	actor, _ := uuid.Parse(p.UserID)
	cmd := domain.AssignInternalDelivery{ShipmentID: id, DriverID: body.DriverID, ActorID: actor, ExpectedVersion: *body.ExpectedVersion, Reason: body.Reason, IdempotencyKey: key}
	if err := cmd.Validate(); err != nil {
		h.assignmentError(w, err)
		return
	}
	if err := h.assignmentDirectory.ValidateDriver(r.Context(), r.Header.Get("Authorization"), body.DriverID); err != nil {
		h.assignmentError(w, err)
		return
	}
	result, err := h.service.AssignInternalDelivery(r.Context(), cmd)
	if err != nil {
		h.assignmentError(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (h *Router) getInternalAssignment(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.assignmentActor(w, r, false); !ok {
		return
	}
	id, ok := h.internalAssignmentTarget(w, r)
	if !ok {
		return
	}
	result, err := h.service.GetInternalDeliveryAssignment(r.Context(), id)
	if err != nil {
		h.assignmentError(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (h *Router) internalAssignmentHistory(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.assignmentActor(w, r, false); !ok {
		return
	}
	id, ok := h.internalAssignmentTarget(w, r)
	if !ok {
		return
	}
	result, err := h.service.InternalDeliveryAssignmentHistory(r.Context(), id)
	if err != nil {
		h.assignmentError(w, err)
		return
	}
	if result == nil {
		result = []domain.InternalDeliveryAssignmentEvent{}
	}
	writeJSON(w, 200, map[string]any{"items": result, "total": len(result)})
}
