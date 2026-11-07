package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/domain"
	"github.com/omamx/care-service/internal/tickets"
)

type TicketHandler struct {
	service *tickets.Service
}

func NewTicketHandler(service *tickets.Service) *TicketHandler {
	return &TicketHandler{service: service}
}

type createTicketRequest struct {
	Subject  string                `json:"subject"`
	Category domain.TicketCategory `json:"category"`
	Message  string                `json:"message"`
	OrderID  string                `json:"order_id,omitempty"`
	FileIDs  []uuid.UUID           `json:"file_ids,omitempty"`
}

func (h *TicketHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, _ := AuthenticatedActor(r.Context())
	var request createTicketRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "The request body is invalid", nil)
		return
	}
	detail, created, err := h.service.Create(r.Context(), tickets.CreateInput{
		CustomerID: actor.ID, Subject: request.Subject, Category: request.Category,
		Message: request.Message, OrderID: request.OrderID, FileIDs: request.FileIDs,
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{"data": ticketDetailDTO(detail)})
}

func (h *TicketHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, _ := AuthenticatedActor(r.Context())
	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, r, http.StatusBadRequest, "INVALID_LIMIT", "limit must be between 1 and 100", nil)
			return
		}
		limit = parsed
	}
	page, err := h.service.List(r.Context(), actor.ID, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	items := make([]map[string]any, 0, len(page.Items))
	for _, ticket := range page.Items {
		items = append(items, ticketDTO(ticket))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items, "meta": map[string]any{"next_cursor": page.NextCursor}})
}

func (h *TicketHandler) Get(w http.ResponseWriter, r *http.Request) {
	actor, _ := AuthenticatedActor(r.Context())
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_TICKET_ID", "The ticket ID is invalid", nil)
		return
	}
	detail, err := h.service.Get(r.Context(), actor.ID, ticketID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": ticketDetailDTO(detail)})
}

type addMessageRequest struct {
	Content string      `json:"content"`
	FileIDs []uuid.UUID `json:"file_ids,omitempty"`
}

func (h *TicketHandler) AddMessage(w http.ResponseWriter, r *http.Request) {
	actor, _ := AuthenticatedActor(r.Context())
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_TICKET_ID", "The ticket ID is invalid", nil)
		return
	}
	var request addMessageRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "The request body is invalid", nil)
		return
	}
	message, created, err := h.service.AddMessage(r.Context(), tickets.AddMessageInput{
		CustomerID: actor.ID, TicketID: ticketID, Content: request.Content,
		FileIDs: request.FileIDs, IdempotencyKey: r.Header.Get("Idempotency-Key"),
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

func ticketDTO(ticket domain.SupportTicket) map[string]any {
	return map[string]any{
		"id": ticket.ID, "code": ticket.Code, "subject": ticket.Subject,
		"category": ticket.Category, "priority": ticket.Priority, "status": ticket.Status,
		"order_id": ticket.OrderID, "version": ticket.Version,
		"first_response_due_at": ticket.FirstResponseDueAt,
		"resolution_due_at":     ticket.ResolutionDueAt,
		"created_at":            ticket.CreatedAt, "updated_at": ticket.UpdatedAt,
	}
}

func messageDTO(message domain.TicketMessage) map[string]any {
	fileIDs := message.FileIDs
	if fileIDs == nil {
		fileIDs = []uuid.UUID{}
	}
	return map[string]any{
		"id": message.ID, "sender_type": message.SenderType, "direction": message.Direction,
		"delivery_status": message.DeliveryStatus, "content": message.Content,
		"file_ids": fileIDs, "created_at": message.CreatedAt,
	}
}

func ticketDetailDTO(detail domain.TicketDetail) map[string]any {
	messages := make([]map[string]any, 0, len(detail.Messages))
	for _, message := range detail.Messages {
		messages = append(messages, messageDTO(message))
	}
	result := ticketDTO(detail.Ticket)
	result["messages"] = messages
	return result
}
