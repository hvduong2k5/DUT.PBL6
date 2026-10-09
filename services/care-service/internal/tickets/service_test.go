package tickets

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/domain"
	"github.com/stretchr/testify/require"
)

type fakeTicketRepository struct {
	created domain.CreateTicketRecord
}

func (f *fakeTicketRepository) Create(_ context.Context, record domain.CreateTicketRecord) (domain.TicketDetail, bool, error) {
	f.created = record
	return domain.TicketDetail{Ticket: record.Ticket, Messages: []domain.TicketMessage{record.FirstMessage}}, true, nil
}

func (f *fakeTicketRepository) ListOwned(context.Context, uuid.UUID, int, string) (domain.TicketPage, error) {
	return domain.TicketPage{}, nil
}

func (f *fakeTicketRepository) GetOwned(context.Context, uuid.UUID, uuid.UUID) (domain.TicketDetail, error) {
	return domain.TicketDetail{}, nil
}

func (f *fakeTicketRepository) AddCustomerMessage(_ context.Context, record domain.AddTicketMessageRecord) (domain.TicketMessage, bool, error) {
	return record.Message, true, nil
}

func TestCreateTicketBuildsSLAAndInitialMessage(t *testing.T) {
	repository := &fakeTicketRepository{}
	service := NewService(repository, nil)
	now := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	customerID := uuid.New()

	detail, created, err := service.Create(context.Background(), CreateInput{
		CustomerID: customerID, Subject: "Need product help", Category: domain.TicketCategoryProductQuality,
		Message: "The package arrived damaged", IdempotencyKey: "ticket-create-1",
	})
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, domain.TicketStatusNew, detail.Ticket.Status)
	require.Equal(t, now.Add(4*time.Hour), repository.created.Ticket.FirstResponseDueAt)
	require.Equal(t, now.Add(48*time.Hour), repository.created.Ticket.ResolutionDueAt)
	require.Equal(t, "CUSTOMER", repository.created.FirstMessage.SenderType)
}

func TestCreateTicketFailsClosedForUnverifiedOrder(t *testing.T) {
	service := NewService(&fakeTicketRepository{}, nil)
	_, _, err := service.Create(context.Background(), CreateInput{
		CustomerID: uuid.New(), Subject: "Need order help", Category: domain.TicketCategoryOrder,
		Message: "Please check this order", OrderID: "ORD-123", IdempotencyKey: "ticket-create-2",
	})
	require.ErrorIs(t, err, domain.ErrDependencyUnavailable)
}
