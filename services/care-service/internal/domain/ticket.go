package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TicketStatus string

const (
	TicketStatusNew             TicketStatus = "NEW"
	TicketStatusTriaged         TicketStatus = "TRIAGED"
	TicketStatusAssigned        TicketStatus = "ASSIGNED"
	TicketStatusInProgress      TicketStatus = "IN_PROGRESS"
	TicketStatusWaitingCustomer TicketStatus = "WAITING_CUSTOMER"
	TicketStatusWaitingInternal TicketStatus = "WAITING_INTERNAL"
	TicketStatusResolved        TicketStatus = "RESOLVED"
	TicketStatusClosed          TicketStatus = "CLOSED"
)

type TicketPriority string

const (
	TicketPriorityLow    TicketPriority = "LOW"
	TicketPriorityNormal TicketPriority = "NORMAL"
	TicketPriorityHigh   TicketPriority = "HIGH"
	TicketPriorityUrgent TicketPriority = "URGENT"
)

type TicketCategory string

const (
	TicketCategoryOrder          TicketCategory = "ORDER"
	TicketCategoryPayment        TicketCategory = "PAYMENT"
	TicketCategoryShipping       TicketCategory = "SHIPPING"
	TicketCategoryProductQuality TicketCategory = "PRODUCT_QUALITY"
	TicketCategoryReturnRefund   TicketCategory = "RETURN_REFUND"
	TicketCategoryAccount        TicketCategory = "ACCOUNT"
	TicketCategoryOther          TicketCategory = "OTHER"
)

type SupportTicket struct {
	ID                 uuid.UUID
	Code               string
	CustomerID         uuid.UUID
	Subject            string
	Category           TicketCategory
	Priority           TicketPriority
	Status             TicketStatus
	OrderID            string
	Version            int
	FirstResponseDueAt time.Time
	ResolutionDueAt    time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type TicketMessage struct {
	ID         uuid.UUID
	TicketID   uuid.UUID
	SenderID   uuid.UUID
	SenderType string
	Direction  string
	Content    string
	FileIDs    []uuid.UUID
	CreatedAt  time.Time
}

type TicketDetail struct {
	Ticket   SupportTicket
	Messages []TicketMessage
}

func ValidateTicketInput(subject string, category TicketCategory, message string) error {
	subject = strings.TrimSpace(subject)
	message = strings.TrimSpace(message)
	if len(subject) < 3 || len(subject) > 200 {
		return fmt.Errorf("%w: subject must contain 3-200 characters", ErrInvalidArgument)
	}
	if len(message) < 1 || len(message) > 10_000 {
		return fmt.Errorf("%w: message must contain 1-10000 characters", ErrInvalidArgument)
	}
	if !category.Valid() {
		return fmt.Errorf("%w: unsupported ticket category", ErrInvalidArgument)
	}
	return nil
}

func (c TicketCategory) Valid() bool {
	switch c {
	case TicketCategoryOrder, TicketCategoryPayment, TicketCategoryShipping,
		TicketCategoryProductQuality, TicketCategoryReturnRefund, TicketCategoryAccount, TicketCategoryOther:
		return true
	default:
		return false
	}
}
