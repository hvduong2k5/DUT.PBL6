package tickets

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/domain"
	"github.com/omamx/care-service/internal/idgen"
)

type StaffService struct {
	repository   domain.StaffTicketRepository
	now          func() time.Time
	reopenWindow time.Duration
}

func NewStaffService(repository domain.StaffTicketRepository, reopenWindow ...time.Duration) *StaffService {
	window := 168 * time.Hour
	if len(reopenWindow) > 0 && reopenWindow[0] > 0 {
		window = reopenWindow[0]
	}
	return &StaffService{repository: repository, now: time.Now, reopenWindow: window}
}

func (s *StaffService) List(ctx context.Context, actorID uuid.UUID, filter domain.StaffTicketFilter) (domain.TicketPage, error) {
	if actorID == uuid.Nil {
		return domain.TicketPage{}, domain.ErrForbidden
	}
	filter.Search = strings.TrimSpace(filter.Search)
	filter.AssignedQueue = strings.TrimSpace(filter.AssignedQueue)
	if len(filter.Search) > 200 {
		return domain.TicketPage{}, fmt.Errorf("%w: search must not exceed 200 characters", domain.ErrInvalidArgument)
	}
	if len(filter.AssignedQueue) > 100 {
		return domain.TicketPage{}, fmt.Errorf("%w: assigned queue must not exceed 100 characters", domain.ErrInvalidArgument)
	}
	if filter.Status != "" && !filter.Status.Valid() {
		return domain.TicketPage{}, fmt.Errorf("%w: unsupported ticket status", domain.ErrInvalidArgument)
	}
	if filter.Priority != "" && !filter.Priority.Valid() {
		return domain.TicketPage{}, fmt.Errorf("%w: unsupported ticket priority", domain.ErrInvalidArgument)
	}
	if filter.Category != "" && !filter.Category.Valid() {
		return domain.TicketPage{}, fmt.Errorf("%w: unsupported ticket category", domain.ErrInvalidArgument)
	}
	if filter.Unassigned && filter.AssignedStaffID != uuid.Nil {
		return domain.TicketPage{}, fmt.Errorf("%w: unassigned and assigned_staff_id cannot be combined", domain.ErrInvalidArgument)
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}
	return s.repository.ListForStaff(ctx, filter)
}

func (s *StaffService) Get(ctx context.Context, actorID, ticketID uuid.UUID, requestID string) (domain.StaffTicketDetail, error) {
	if actorID == uuid.Nil {
		return domain.StaffTicketDetail{}, domain.ErrForbidden
	}
	if ticketID == uuid.Nil {
		return domain.StaffTicketDetail{}, fmt.Errorf("%w: ticket ID is required", domain.ErrInvalidArgument)
	}
	return s.repository.GetForStaff(ctx, domain.StaffTicketAccess{
		TicketID:  ticketID,
		ActorID:   actorID,
		RequestID: strings.TrimSpace(requestID),
	})
}

type ClaimInput struct {
	ActorID         uuid.UUID
	TicketID        uuid.UUID
	ExpectedVersion int
	Reason          string
	RequestID       string
	IdempotencyKey  string
}

func (s *StaffService) Claim(ctx context.Context, input ClaimInput) (domain.SupportTicket, bool, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	input.RequestID = strings.TrimSpace(input.RequestID)
	if input.ActorID == uuid.Nil {
		return domain.SupportTicket{}, false, domain.ErrForbidden
	}
	if input.TicketID == uuid.Nil {
		return domain.SupportTicket{}, false, fmt.Errorf("%w: ticket ID is required", domain.ErrInvalidArgument)
	}
	if input.ExpectedVersion < 1 {
		return domain.SupportTicket{}, false, fmt.Errorf("%w: expected_version must be positive", domain.ErrInvalidArgument)
	}
	if len(input.Reason) < 1 || len(input.Reason) > 500 {
		return domain.SupportTicket{}, false, fmt.Errorf("%w: reason must contain 1-500 characters", domain.ErrInvalidArgument)
	}
	if err := validateKey(input.IdempotencyKey); err != nil {
		return domain.SupportTicket{}, false, err
	}

	now := s.now().UTC()
	return s.repository.ClaimForStaff(ctx, domain.ClaimTicketRecord{
		TicketID: input.TicketID, StaffID: input.ActorID, AssignmentID: idgen.New(),
		ExpectedVersion: input.ExpectedVersion, Reason: input.Reason,
		RequestID: input.RequestID, ClaimedAt: now,
		IdempotencyKey: input.IdempotencyKey, RequestHash: claimRequestDigest(input),
		IdempotencyExpiry: now.Add(24 * time.Hour),
	})
}

func claimRequestDigest(input ClaimInput) string {
	return requestDigest(struct {
		ActorID         uuid.UUID `json:"actor_id"`
		TicketID        uuid.UUID `json:"ticket_id"`
		ExpectedVersion int       `json:"expected_version"`
		Reason          string    `json:"reason"`
	}{
		ActorID: input.ActorID, TicketID: input.TicketID,
		ExpectedVersion: input.ExpectedVersion, Reason: input.Reason,
	})
}

type AddInternalNoteInput struct {
	ActorID        uuid.UUID
	TicketID       uuid.UUID
	Content        string
	RequestID      string
	IdempotencyKey string
}

func (s *StaffService) AddInternalNote(ctx context.Context, input AddInternalNoteInput) (domain.TicketInternalNote, bool, error) {
	input.Content = strings.TrimSpace(input.Content)
	input.RequestID = strings.TrimSpace(input.RequestID)
	if input.ActorID == uuid.Nil {
		return domain.TicketInternalNote{}, false, domain.ErrForbidden
	}
	if input.TicketID == uuid.Nil {
		return domain.TicketInternalNote{}, false, fmt.Errorf("%w: ticket ID is required", domain.ErrInvalidArgument)
	}
	if len(input.Content) < 1 || len(input.Content) > 10_000 {
		return domain.TicketInternalNote{}, false, fmt.Errorf("%w: content must contain 1-10000 characters", domain.ErrInvalidArgument)
	}
	if err := validateKey(input.IdempotencyKey); err != nil {
		return domain.TicketInternalNote{}, false, err
	}

	now := s.now().UTC()
	note := domain.TicketInternalNote{
		ID: idgen.New(), TicketID: input.TicketID, AuthorID: input.ActorID,
		Content: input.Content, CreatedAt: now, UpdatedAt: now,
	}
	return s.repository.AddInternalNote(ctx, domain.AddInternalNoteRecord{
		Note: note, ActorID: input.ActorID, RequestID: input.RequestID,
		IdempotencyKey: input.IdempotencyKey, RequestHash: internalNoteRequestDigest(input),
		IdempotencyExpiry: now.Add(24 * time.Hour),
	})
}

func internalNoteRequestDigest(input AddInternalNoteInput) string {
	return requestDigest(struct {
		ActorID  uuid.UUID `json:"actor_id"`
		TicketID uuid.UUID `json:"ticket_id"`
		Content  string    `json:"content"`
	}{ActorID: input.ActorID, TicketID: input.TicketID, Content: input.Content})
}

type AddStaffReplyInput struct {
	ActorID        uuid.UUID
	TicketID       uuid.UUID
	Content        string
	RequestID      string
	IdempotencyKey string
}

func (s *StaffService) AddReply(ctx context.Context, input AddStaffReplyInput) (domain.TicketMessage, bool, error) {
	input.Content = strings.TrimSpace(input.Content)
	input.RequestID = strings.TrimSpace(input.RequestID)
	if input.ActorID == uuid.Nil {
		return domain.TicketMessage{}, false, domain.ErrForbidden
	}
	if input.TicketID == uuid.Nil {
		return domain.TicketMessage{}, false, fmt.Errorf("%w: ticket ID is required", domain.ErrInvalidArgument)
	}
	if len(input.Content) < 1 || len(input.Content) > 10_000 {
		return domain.TicketMessage{}, false, fmt.Errorf("%w: content must contain 1-10000 characters", domain.ErrInvalidArgument)
	}
	if err := validateKey(input.IdempotencyKey); err != nil {
		return domain.TicketMessage{}, false, err
	}

	now := s.now().UTC()
	message := domain.TicketMessage{
		ID: idgen.New(), TicketID: input.TicketID, SenderID: input.ActorID,
		SenderType: "STAFF", Direction: "OUTBOUND", DeliveryStatus: "STORED",
		Content: input.Content, CreatedAt: now,
	}
	return s.repository.AddStaffReply(ctx, domain.AddStaffReplyRecord{
		Message: message, ActorID: input.ActorID, RequestID: input.RequestID,
		IdempotencyKey: input.IdempotencyKey, RequestHash: staffReplyRequestDigest(input),
		IdempotencyExpiry: now.Add(24 * time.Hour),
	})
}

func staffReplyRequestDigest(input AddStaffReplyInput) string {
	return requestDigest(struct {
		ActorID  uuid.UUID `json:"actor_id"`
		TicketID uuid.UUID `json:"ticket_id"`
		Content  string    `json:"content"`
	}{ActorID: input.ActorID, TicketID: input.TicketID, Content: input.Content})
}

type TransitionInput struct {
	ActorID         uuid.UUID
	TicketID        uuid.UUID
	ExpectedVersion int
	ToStatus        domain.TicketStatus
	Reason          string
	RequestID       string
	IdempotencyKey  string
}

func (s *StaffService) Transition(ctx context.Context, input TransitionInput) (domain.SupportTicket, bool, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	input.RequestID = strings.TrimSpace(input.RequestID)
	if input.ActorID == uuid.Nil {
		return domain.SupportTicket{}, false, domain.ErrForbidden
	}
	if input.TicketID == uuid.Nil {
		return domain.SupportTicket{}, false, fmt.Errorf("%w: ticket ID is required", domain.ErrInvalidArgument)
	}
	if input.ExpectedVersion < 1 {
		return domain.SupportTicket{}, false, fmt.Errorf("%w: expected_version must be positive", domain.ErrInvalidArgument)
	}
	if !input.ToStatus.Valid() || input.ToStatus == domain.TicketStatusNew || input.ToStatus == domain.TicketStatusAssigned {
		return domain.SupportTicket{}, false, fmt.Errorf("%w: unsupported transition target", domain.ErrInvalidArgument)
	}
	if len(input.Reason) < 1 || len(input.Reason) > 500 {
		return domain.SupportTicket{}, false, fmt.Errorf("%w: reason must contain 1-500 characters", domain.ErrInvalidArgument)
	}
	if err := validateKey(input.IdempotencyKey); err != nil {
		return domain.SupportTicket{}, false, err
	}

	now := s.now().UTC()
	return s.repository.TransitionForStaff(ctx, domain.TransitionTicketRecord{
		TicketID: input.TicketID, ActorID: input.ActorID,
		ExpectedVersion: input.ExpectedVersion, ToStatus: input.ToStatus,
		Reason: input.Reason, RequestID: input.RequestID, TransitionedAt: now,
		ReopenWindow: s.reopenWindow, IdempotencyKey: input.IdempotencyKey,
		RequestHash: transitionRequestDigest(input), IdempotencyExpiry: now.Add(24 * time.Hour),
	})
}

func transitionRequestDigest(input TransitionInput) string {
	return requestDigest(struct {
		ActorID         uuid.UUID           `json:"actor_id"`
		TicketID        uuid.UUID           `json:"ticket_id"`
		ExpectedVersion int                 `json:"expected_version"`
		ToStatus        domain.TicketStatus `json:"to_status"`
		Reason          string              `json:"reason"`
	}{
		ActorID: input.ActorID, TicketID: input.TicketID,
		ExpectedVersion: input.ExpectedVersion, ToStatus: input.ToStatus, Reason: input.Reason,
	})
}
