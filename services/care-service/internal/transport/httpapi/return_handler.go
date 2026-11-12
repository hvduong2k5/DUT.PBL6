package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/domain"
	careReturns "github.com/omamx/care-service/internal/returns"
)

type ReturnHandler struct{ service *careReturns.Service }

func NewReturnHandler(service *careReturns.Service) *ReturnHandler {
	return &ReturnHandler{service: service}
}

type createReturnRequest struct {
	OrderID         string                      `json:"order_id"`
	Resolution      string                      `json:"requested_resolution"`
	ReasonCode      string                      `json:"reason_code"`
	CustomerNote    string                      `json:"customer_note"`
	Lines           []careReturns.RequestedLine `json:"lines"`
	EvidenceFileIDs []uuid.UUID                 `json:"evidence_file_ids"`
}

func (h *ReturnHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := AuthenticatedActor(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required", nil)
		return
	}
	var request createReturnRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "The request body is invalid", nil)
		return
	}
	value, created, err := h.service.Create(r.Context(), careReturns.CreateInput{
		ActorID: actor.ID, OrderID: request.OrderID, Resolution: request.Resolution, ReasonCode: request.ReasonCode,
		CustomerNote: request.CustomerNote, Lines: request.Lines, EvidenceFileIDs: request.EvidenceFileIDs,
		RequestID: RequestID(r.Context()), IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	writeJSON(w, status, customerReturnDTO(value))
}

func (h *ReturnHandler) ListOwned(w http.ResponseWriter, r *http.Request) {
	actor, ok := AuthenticatedActor(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required", nil)
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	items, total, err := h.service.ListOwned(r.Context(), actor.ID, domain.ReturnCaseStatus(strings.TrimSpace(r.URL.Query().Get("status"))), page, pageSize)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	dto := make([]map[string]any, 0, len(items))
	for _, item := range items {
		dto = append(dto, customerReturnDTO(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": dto, "total": total, "page": max(page, 1), "pageSize": normalizedPageSize(pageSize)})
}

func (h *ReturnHandler) GetOwned(w http.ResponseWriter, r *http.Request) {
	actor, ok := AuthenticatedActor(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required", nil)
		return
	}
	caseID, err := uuid.Parse(chi.URLParam(r, "caseID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_CASE_ID", "The return case ID is invalid", nil)
		return
	}
	value, err := h.service.GetOwned(r.Context(), actor.ID, caseID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, customerReturnDTO(value))
}

func (h *ReturnHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	values, total, err := h.service.List(r.Context(), domain.ReturnCaseStatus(strings.TrimSpace(r.URL.Query().Get("status"))), page, pageSize)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	items := make([]map[string]any, 0, len(values))
	for _, value := range values {
		items = append(items, adminReturnDTO(value))
	}
	counts, err := h.service.StatusCounts(r.Context())
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	statusCounts := make([]map[string]any, 0, len(counts))
	for status, count := range counts {
		statusCounts = append(statusCounts, map[string]any{"status": status, "count": count})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": max(page, 1), "pageSize": normalizedPageSize(pageSize), "statusCounts": statusCounts, "caseTypes": []map[string]string{{"code": "RETURN_REFUND", "label": "Trả hàng và hoàn tiền"}}, "calculatedAt": time.Now().UTC()})
}

func (h *ReturnHandler) GetAdmin(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "caseID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_CASE_ID", "The return case ID is invalid", nil)
		return
	}
	value, err := h.service.Get(r.Context(), caseID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, adminReturnDTO(value))
}

type returnActionRequest struct {
	Action          string                          `json:"action"`
	Note            string                          `json:"note"`
	ReasonCode      string                          `json:"reason_code"`
	ExpectedVersion int                             `json:"expected_revision"`
	ReturnRequired  bool                            `json:"return_required"`
	Lines           []careReturns.DecisionLineInput `json:"lines"`
}

func (h *ReturnHandler) Action(w http.ResponseWriter, r *http.Request) {
	actor, ok := AuthenticatedActor(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required", nil)
		return
	}
	caseID, err := uuid.Parse(chi.URLParam(r, "caseID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_CASE_ID", "The return case ID is invalid", nil)
		return
	}
	var request returnActionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "The request body is invalid", nil)
		return
	}
	permission := "RETURN_CASE_REVIEW"
	switch strings.ToUpper(strings.TrimSpace(request.Action)) {
	case "DECIDE":
		permission = "RETURN_CASE_DECIDE"
	case "START_REVIEW", "REQUEST_INFORMATION":
		permission = "RETURN_CASE_REVIEW"
	default:
		writeError(w, r, http.StatusBadRequest, "INVALID_ACTION", "The return case action is invalid", nil)
		return
	}
	if !hasPermission(actor, permission) {
		writeError(w, r, http.StatusForbidden, "FORBIDDEN", "The operation is not permitted", nil)
		return
	}
	if strings.EqualFold(request.Action, "DECIDE") && !request.ReturnRequired && hasPositiveApproval(request.Lines) && !hasPermission(actor, "RETURN_CASE_REFUND_WITHOUT_RETURN") {
		writeError(w, r, http.StatusForbidden, "FORBIDDEN", "A separate exception permission is required for refund without returned goods", nil)
		return
	}
	value, _, err := h.service.Action(r.Context(), careReturns.ActionInput{
		ActorID: actor.ID, CaseID: caseID, Action: request.Action, ExpectedVersion: request.ExpectedVersion,
		ReasonCode: request.ReasonCode, Note: request.Note, ReturnRequired: request.ReturnRequired, Lines: request.Lines,
		RequestID: RequestID(r.Context()), IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, adminReturnDTO(value))
}

func hasPermission(actor Actor, permission string) bool {
	_, ok := actor.Permissions[permission]
	return ok
}
func hasPositiveApproval(lines []careReturns.DecisionLineInput) bool {
	for _, line := range lines {
		if line.ApprovedQuantity > 0 {
			return true
		}
	}
	return false
}

func customerReturnDTO(value careReturns.Case) map[string]any {
	lines := make([]map[string]any, 0, len(value.Lines))
	for _, line := range value.Lines {
		lines = append(lines, map[string]any{"case_line_id": line.ID, "order_line_id": line.OrderLineID, "product_name": line.ProductName, "requested_quantity": line.RequestedQuantity})
	}
	return map[string]any{"case_id": value.ID, "case_number": value.Code, "order_id": value.OrderID, "status": value.Status, "requested_resolution": value.Resolution, "reason_code": value.ReasonCode, "customer_note": value.CustomerNote, "revision": value.Version, "lines": lines, "evidence_file_ids": value.EvidenceFileIDs, "decision": customerDecisionDTO(value.Decision), "created_at": value.CreatedAt, "updated_at": value.UpdatedAt}
}

func adminReturnDTO(value careReturns.Case) map[string]any {
	return map[string]any{"case_id": value.ID, "case_number": value.Code, "order_id": value.OrderID, "order_number": value.OrderNumber, "status": value.Status, "requested_resolution": value.Resolution, "reason_code": value.ReasonCode, "customer_note": value.CustomerNote, "revision": value.Version, "lines": value.Lines, "evidence_file_ids": value.EvidenceFileIDs, "decision": customerDecisionDTO(value.Decision), "reverse_shipment_id": value.ReverseShipmentID, "updated_at": value.UpdatedAt}
}

func customerDecisionDTO(value *careReturns.Decision) any {
	if value == nil {
		return nil
	}
	return map[string]any{"decision_id": value.ID, "decision_version": value.Version, "outcome": value.Outcome, "reason_code": value.ReasonCode, "reason": value.Reason, "return_required": value.ReturnRequired, "approved_total_minor": value.ApprovedTotalMinor, "currency": value.Currency, "lines": value.Lines, "decided_at": value.DecidedAt}
}

func normalizedPageSize(value int) int {
	if value < 1 || value > 100 {
		return 20
	}
	return value
}
