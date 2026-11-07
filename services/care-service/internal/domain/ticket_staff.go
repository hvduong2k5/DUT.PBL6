package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TicketInternalNote struct {
	ID        uuid.UUID
	TicketID  uuid.UUID
	AuthorID  uuid.UUID
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TicketAssignment struct {
	ID         uuid.UUID
	TicketID   uuid.UUID
	StaffID    uuid.UUID
	Queue      string
	AssignedBy uuid.UUID
	Reason     string
	AssignedAt time.Time
	EndedAt    *time.Time
}

type TicketStatusChange struct {
	ID         uuid.UUID
	TicketID   uuid.UUID
	FromStatus *TicketStatus
	ToStatus   TicketStatus
	ActorID    uuid.UUID
	Reason     string
	CreatedAt  time.Time
}

type StaffTicketDetail struct {
	Ticket        SupportTicket
	Messages      []TicketMessage
	InternalNotes []TicketInternalNote
	Assignments   []TicketAssignment
	StatusHistory []TicketStatusChange
}

type StaffTicketFilter struct {
	Search          string
	Status          TicketStatus
	Priority        TicketPriority
	Category        TicketCategory
	AssignedStaffID uuid.UUID
	AssignedQueue   string
	Unassigned      bool
	Limit           int
	Cursor          string
}

type StaffTicketAccess struct {
	TicketID  uuid.UUID
	ActorID   uuid.UUID
	RequestID string
}

type ClaimTicketRecord struct {
	TicketID          uuid.UUID
	StaffID           uuid.UUID
	AssignmentID      uuid.UUID
	ExpectedVersion   int
	Reason            string
	RequestID         string
	ClaimedAt         time.Time
	IdempotencyKey    string
	RequestHash       string
	IdempotencyExpiry time.Time
}

type AddInternalNoteRecord struct {
	Note              TicketInternalNote
	ActorID           uuid.UUID
	RequestID         string
	IdempotencyKey    string
	RequestHash       string
	IdempotencyExpiry time.Time
}

type TransitionTicketRecord struct {
	TicketID          uuid.UUID
	ActorID           uuid.UUID
	ExpectedVersion   int
	ToStatus          TicketStatus
	Reason            string
	RequestID         string
	TransitionedAt    time.Time
	ReopenWindow      time.Duration
	IdempotencyKey    string
	RequestHash       string
	IdempotencyExpiry time.Time
}

type AddStaffReplyRecord struct {
	Message           TicketMessage
	ActorID           uuid.UUID
	RequestID         string
	IdempotencyKey    string
	RequestHash       string
	IdempotencyExpiry time.Time
}

type StaffTicketRepository interface {
	ListForStaff(context.Context, StaffTicketFilter) (TicketPage, error)
	GetForStaff(context.Context, StaffTicketAccess) (StaffTicketDetail, error)
	ClaimForStaff(context.Context, ClaimTicketRecord) (SupportTicket, bool, error)
	AddInternalNote(context.Context, AddInternalNoteRecord) (TicketInternalNote, bool, error)
	TransitionForStaff(context.Context, TransitionTicketRecord) (SupportTicket, bool, error)
	AddStaffReply(context.Context, AddStaffReplyRecord) (TicketMessage, bool, error)
}
