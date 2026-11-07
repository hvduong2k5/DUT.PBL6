package tickets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/domain"
	"github.com/omamx/care-service/internal/idgen"
)

type OrderOwnershipVerifier interface {
	VerifyOrderOwnership(context.Context, string, uuid.UUID) error
}

type Service struct {
	repository    domain.TicketRepository
	orderVerifier OrderOwnershipVerifier
	now           func() time.Time
}

func NewService(repository domain.TicketRepository, orderVerifier OrderOwnershipVerifier) *Service {
	return &Service{repository: repository, orderVerifier: orderVerifier, now: time.Now}
}

type CreateInput struct {
	CustomerID     uuid.UUID
	Subject        string
	Category       domain.TicketCategory
	Message        string
	OrderID        string
	FileIDs        []uuid.UUID
	IdempotencyKey string
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.TicketDetail, bool, error) {
	input.Subject = strings.TrimSpace(input.Subject)
	input.Message = strings.TrimSpace(input.Message)
	input.OrderID = strings.TrimSpace(input.OrderID)
	if input.CustomerID == uuid.Nil {
		return domain.TicketDetail{}, false, domain.ErrForbidden
	}
	if err := validateKey(input.IdempotencyKey); err != nil {
		return domain.TicketDetail{}, false, err
	}
	if err := domain.ValidateTicketInput(input.Subject, input.Category, input.Message); err != nil {
		return domain.TicketDetail{}, false, err
	}
	if len(input.FileIDs) > 5 {
		return domain.TicketDetail{}, false, fmt.Errorf("%w: at most 5 attachments are allowed", domain.ErrInvalidArgument)
	}
	if input.OrderID != "" {
		if s.orderVerifier == nil {
			return domain.TicketDetail{}, false, fmt.Errorf("%w: order linking is unavailable until ownership can be verified", domain.ErrDependencyUnavailable)
		}
		if err := s.orderVerifier.VerifyOrderOwnership(ctx, input.OrderID, input.CustomerID); err != nil {
			return domain.TicketDetail{}, false, err
		}
	}

	now := s.now().UTC()
	ticketID := idgen.New()
	messageID := idgen.New()
	requestHash := requestDigest(input)
	detail, created, err := s.repository.Create(ctx, domain.CreateTicketRecord{
		Ticket: domain.SupportTicket{
			ID: ticketID, Code: idgen.TicketCode(ticketID), CustomerID: input.CustomerID,
			Subject: input.Subject, Category: input.Category, Priority: domain.TicketPriorityNormal,
			Status: domain.TicketStatusNew, OrderID: input.OrderID, Version: 1,
			FirstResponseDueAt: now.Add(4 * time.Hour), ResolutionDueAt: now.Add(48 * time.Hour),
			CreatedAt: now, UpdatedAt: now,
		},
		FirstMessage: domain.TicketMessage{
			ID: messageID, TicketID: ticketID, SenderID: input.CustomerID,
			SenderType: "CUSTOMER", Direction: "INBOUND", DeliveryStatus: "STORED", Content: input.Message,
			FileIDs: input.FileIDs, CreatedAt: now,
		},
		FileIDs: input.FileIDs, IdempotencyKey: input.IdempotencyKey,
		RequestHash: requestHash, IdempotencyExpiry: now.Add(24 * time.Hour),
	})
	return detail, created, err
}

func (s *Service) List(ctx context.Context, customerID uuid.UUID, limit int, cursor string) (domain.TicketPage, error) {
	if customerID == uuid.Nil {
		return domain.TicketPage{}, domain.ErrForbidden
	}
	return s.repository.ListOwned(ctx, customerID, limit, cursor)
}

func (s *Service) Get(ctx context.Context, customerID, ticketID uuid.UUID) (domain.TicketDetail, error) {
	if customerID == uuid.Nil || ticketID == uuid.Nil {
		return domain.TicketDetail{}, domain.ErrForbidden
	}
	return s.repository.GetOwned(ctx, ticketID, customerID)
}

type AddMessageInput struct {
	CustomerID     uuid.UUID
	TicketID       uuid.UUID
	Content        string
	FileIDs        []uuid.UUID
	IdempotencyKey string
}

func (s *Service) AddMessage(ctx context.Context, input AddMessageInput) (domain.TicketMessage, bool, error) {
	input.Content = strings.TrimSpace(input.Content)
	if input.CustomerID == uuid.Nil || input.TicketID == uuid.Nil {
		return domain.TicketMessage{}, false, domain.ErrForbidden
	}
	if err := validateKey(input.IdempotencyKey); err != nil {
		return domain.TicketMessage{}, false, err
	}
	if len(input.Content) < 1 || len(input.Content) > 10_000 {
		return domain.TicketMessage{}, false, fmt.Errorf("%w: content must contain 1-10000 characters", domain.ErrInvalidArgument)
	}
	if len(input.FileIDs) > 5 {
		return domain.TicketMessage{}, false, fmt.Errorf("%w: at most 5 attachments are allowed", domain.ErrInvalidArgument)
	}
	now := s.now().UTC()
	message := domain.TicketMessage{
		ID: idgen.New(), TicketID: input.TicketID, SenderID: input.CustomerID,
		SenderType: "CUSTOMER", Direction: "INBOUND", DeliveryStatus: "STORED", Content: input.Content,
		FileIDs: input.FileIDs, CreatedAt: now,
	}
	return s.repository.AddCustomerMessage(ctx, domain.AddTicketMessageRecord{
		Message: message, CustomerID: input.CustomerID, FileIDs: input.FileIDs,
		IdempotencyKey: input.IdempotencyKey, RequestHash: requestDigest(input),
		IdempotencyExpiry: now.Add(24 * time.Hour),
	})
}

func validateKey(key string) error {
	if strings.TrimSpace(key) == "" || len(key) > 200 {
		return fmt.Errorf("%w: Idempotency-Key is required and must not exceed 200 characters", domain.ErrInvalidArgument)
	}
	return nil
}

func requestDigest(value any) string {
	payload, _ := json.Marshal(value)
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}
