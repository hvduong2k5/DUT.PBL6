package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/domain"
	"github.com/omamx/care-service/internal/tickets"
)

type StaffTicketHandler struct {
	service *tickets.StaffService
}

func NewStaffTicketHandler(service *tickets.StaffService) *StaffTicketHandler {
	return &StaffTicketHandler{service: service}
}

func (h *StaffTicketHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, _ := AuthenticatedActor(r.Context())
	filter, err := parseStaffTicketFilter(r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	page, err := h.service.List(r.Context(), actor.ID, filter)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	items := make([]map[string]any, 0, len(page.Items))
	for _, ticket := range page.Items {
		items = append(items, staffTicketDTO(ticket))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"next_cursor": page.NextCursor},
	})
}

func (h *StaffTicketHandler) Get(w http.ResponseWriter, r *http.Request) {
	actor, _ := AuthenticatedActor(r.Context())
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_TICKET_ID", "The ticket ID is invalid", nil)
		return
	}
	detail, err := h.service.Get(r.Context(), actor.ID, ticketID, RequestID(r.Context()))
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": staffTicketDetailDTO(detail)})
}

type claimTicketRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	Reason          string `json:"reason"`
}

func (h *StaffTicketHandler) Claim(w http.ResponseWriter, r *http.Request) {
	actor, _ := AuthenticatedActor(r.Context())
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_TICKET_ID", "The ticket ID is invalid", nil)
		return
	}
	var request claimTicketRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "The request body is invalid", nil)
		return
	}
	ticket, _, err := h.service.Claim(r.Context(), tickets.ClaimInput{
		ActorID: actor.ID, TicketID: ticketID, ExpectedVersion: request.ExpectedVersion,
		Reason: request.Reason, RequestID: RequestID(r.Context()),
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": staffTicketDTO(ticket)})
}

type addInternalNoteRequest struct {
	Content string `json:"content"`
}

func (h *StaffTicketHandler) AddInternalNote(w http.ResponseWriter, r *http.Request) {
	actor, _ := AuthenticatedActor(r.Context())
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_TICKET_ID", "The ticket ID is invalid", nil)
		return
	}
	var request addInternalNoteRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "The request body is invalid", nil)
		return
	}
	note, created, err := h.service.AddInternalNote(r.Context(), tickets.AddInternalNoteInput{
		ActorID: actor.ID, TicketID: ticketID, Content: request.Content,
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
	writeJSON(w, status, map[string]any{"data": internalNoteDTO(note)})
}

type addStaffReplyRequest struct {
	Content string `json:"content"`
}

func (h *StaffTicketHandler) AddReply(w http.ResponseWriter, r *http.Request) {
	actor, _ := AuthenticatedActor(r.Context())
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_TICKET_ID", "The ticket ID is invalid", nil)
		return
	}
	var request addStaffReplyRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "The request body is invalid", nil)
		return
	}
	message, created, err := h.service.AddReply(r.Context(), tickets.AddStaffReplyInput{
		ActorID: actor.ID, TicketID: ticketID, Content: request.Content,
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
	writeJSON(w, status, map[string]any{"data": messageDTO(message)})
}

type transitionTicketRequest struct {
	ExpectedVersion int                 `json:"expected_version"`
	ToStatus        domain.TicketStatus `json:"to_status"`
	Reason          string              `json:"reason"`
}

func (h *StaffTicketHandler) Transition(w http.ResponseWriter, r *http.Request) {
	actor, _ := AuthenticatedActor(r.Context())
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_TICKET_ID", "The ticket ID is invalid", nil)
		return
	}
	var request transitionTicketRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "The request body is invalid", nil)
		return
	}
	ticket, _, err := h.service.Transition(r.Context(), tickets.TransitionInput{
		ActorID: actor.ID, TicketID: ticketID, ExpectedVersion: request.ExpectedVersion,
		ToStatus: request.ToStatus, Reason: request.Reason,
		RequestID: RequestID(r.Context()), IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": staffTicketDTO(ticket)})
}

func parseStaffTicketFilter(r *http.Request) (domain.StaffTicketFilter, error) {
	query := r.URL.Query()
	filter := domain.StaffTicketFilter{
		Search:        query.Get("q"),
		Status:        domain.TicketStatus(strings.ToUpper(query.Get("status"))),
		Priority:      domain.TicketPriority(strings.ToUpper(query.Get("priority"))),
		Category:      domain.TicketCategory(strings.ToUpper(query.Get("category"))),
		AssignedQueue: query.Get("assigned_queue"),
		Cursor:        query.Get("cursor"),
		Limit:         20,
	}
	if value := query.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			return domain.StaffTicketFilter{}, domain.ErrInvalidArgument
		}
		filter.Limit = parsed
	}
	if value := query.Get("assigned_staff_id"); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil || parsed == uuid.Nil {
			return domain.StaffTicketFilter{}, domain.ErrInvalidArgument
		}
		filter.AssignedStaffID = parsed
	}
	if value := query.Get("unassigned"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return domain.StaffTicketFilter{}, domain.ErrInvalidArgument
		}
		filter.Unassigned = parsed
	}
	return filter, nil
}

func staffTicketDTO(ticket domain.SupportTicket) map[string]any {
	result := ticketDTO(ticket)
	result["customer_id"] = ticket.CustomerID
	result["assigned_staff_id"] = nullableUUID(ticket.AssignedStaffID)
	result["assigned_queue"] = nullableString(ticket.AssignedQueue)
	result["first_responded_at"] = ticket.FirstRespondedAt
	result["resolved_at"] = ticket.ResolvedAt
	result["closed_at"] = ticket.ClosedAt
	return result
}

func staffTicketDetailDTO(detail domain.StaffTicketDetail) map[string]any {
	result := staffTicketDTO(detail.Ticket)
	messages := make([]map[string]any, 0, len(detail.Messages))
	for _, message := range detail.Messages {
		messages = append(messages, messageDTO(message))
	}
	notes := make([]map[string]any, 0, len(detail.InternalNotes))
	for _, note := range detail.InternalNotes {
		notes = append(notes, internalNoteDTO(note))
	}
	assignments := make([]map[string]any, 0, len(detail.Assignments))
	for _, assignment := range detail.Assignments {
		assignments = append(assignments, map[string]any{
			"id": assignment.ID, "staff_id": nullableUUID(assignment.StaffID),
			"queue": nullableString(assignment.Queue), "assigned_by": assignment.AssignedBy,
			"reason": nullableString(assignment.Reason), "assigned_at": assignment.AssignedAt,
			"ended_at": assignment.EndedAt,
		})
	}
	history := make([]map[string]any, 0, len(detail.StatusHistory))
	for _, change := range detail.StatusHistory {
		history = append(history, map[string]any{
			"id": change.ID, "from_status": change.FromStatus, "to_status": change.ToStatus,
			"actor_id": change.ActorID, "reason": nullableString(change.Reason),
			"created_at": change.CreatedAt,
		})
	}
	result["messages"] = messages
	result["internal_notes"] = notes
	result["assignments"] = assignments
	result["status_history"] = history
	return result
}

func internalNoteDTO(note domain.TicketInternalNote) map[string]any {
	return map[string]any{
		"id": note.ID, "author_id": note.AuthorID, "content": note.Content,
		"created_at": note.CreatedAt, "updated_at": note.UpdatedAt,
	}
}

func nullableUUID(value uuid.UUID) any {
	if value == uuid.Nil {
		return nil
	}
	return value
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
