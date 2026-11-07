package tickets

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/domain"
	"github.com/stretchr/testify/require"
)

type fakeStaffTicketRepository struct {
	filter           domain.StaffTicketFilter
	access           domain.StaffTicketAccess
	claimRecord      domain.ClaimTicketRecord
	noteRecord       domain.AddInternalNoteRecord
	transitionRecord domain.TransitionTicketRecord
	replyRecord      domain.AddStaffReplyRecord
}

func (f *fakeStaffTicketRepository) ListForStaff(_ context.Context, filter domain.StaffTicketFilter) (domain.TicketPage, error) {
	f.filter = filter
	return domain.TicketPage{}, nil
}

func (f *fakeStaffTicketRepository) GetForStaff(_ context.Context, access domain.StaffTicketAccess) (domain.StaffTicketDetail, error) {
	f.access = access
	return domain.StaffTicketDetail{Ticket: domain.SupportTicket{ID: access.TicketID}}, nil
}

func (f *fakeStaffTicketRepository) ClaimForStaff(_ context.Context, record domain.ClaimTicketRecord) (domain.SupportTicket, bool, error) {
	f.claimRecord = record
	return domain.SupportTicket{
		ID: record.TicketID, AssignedStaffID: record.StaffID,
		Status: domain.TicketStatusAssigned, Version: record.ExpectedVersion + 1,
	}, true, nil
}

func (f *fakeStaffTicketRepository) AddInternalNote(_ context.Context, record domain.AddInternalNoteRecord) (domain.TicketInternalNote, bool, error) {
	f.noteRecord = record
	return record.Note, true, nil
}

func (f *fakeStaffTicketRepository) TransitionForStaff(_ context.Context, record domain.TransitionTicketRecord) (domain.SupportTicket, bool, error) {
	f.transitionRecord = record
	return domain.SupportTicket{
		ID: record.TicketID, AssignedStaffID: record.ActorID,
		Status: record.ToStatus, Version: record.ExpectedVersion + 1,
	}, true, nil
}

func (f *fakeStaffTicketRepository) AddStaffReply(_ context.Context, record domain.AddStaffReplyRecord) (domain.TicketMessage, bool, error) {
	f.replyRecord = record
	return record.Message, true, nil
}

func TestStaffListNormalizesAndValidatesFilters(t *testing.T) {
	repository := &fakeStaffTicketRepository{}
	service := NewStaffService(repository)
	actorID := uuid.New()

	_, err := service.List(context.Background(), actorID, domain.StaffTicketFilter{
		Search: "  damaged box  ", Status: domain.TicketStatusNew,
		Priority: domain.TicketPriorityHigh, Category: domain.TicketCategoryShipping,
		AssignedQueue: "  delivery  ", Limit: 25,
	})
	require.NoError(t, err)
	require.Equal(t, "damaged box", repository.filter.Search)
	require.Equal(t, "delivery", repository.filter.AssignedQueue)
	require.Equal(t, 25, repository.filter.Limit)

	_, err = service.List(context.Background(), actorID, domain.StaffTicketFilter{
		Status: "UNKNOWN",
	})
	require.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestStaffListRejectsConflictingAssignmentFilters(t *testing.T) {
	service := NewStaffService(&fakeStaffTicketRepository{})
	_, err := service.List(context.Background(), uuid.New(), domain.StaffTicketFilter{
		AssignedStaffID: uuid.New(), Unassigned: true,
	})
	require.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestStaffGetCarriesAuditContext(t *testing.T) {
	repository := &fakeStaffTicketRepository{}
	service := NewStaffService(repository)
	actorID := uuid.New()
	ticketID := uuid.New()

	detail, err := service.Get(context.Background(), actorID, ticketID, "request-123")
	require.NoError(t, err)
	require.Equal(t, ticketID, detail.Ticket.ID)
	require.Equal(t, actorID, repository.access.ActorID)
	require.Equal(t, ticketID, repository.access.TicketID)
	require.Equal(t, "request-123", repository.access.RequestID)
}

func TestStaffClaimBuildsSelfAssignment(t *testing.T) {
	repository := &fakeStaffTicketRepository{}
	service := NewStaffService(repository)
	now := time.Date(2026, time.October, 10, 8, 30, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	actorID := uuid.New()
	ticketID := uuid.New()

	ticket, applied, err := service.Claim(context.Background(), ClaimInput{
		ActorID: actorID, TicketID: ticketID, ExpectedVersion: 2,
		Reason: "  Taking ownership  ", RequestID: " request-claim ",
		IdempotencyKey: "claim-key-1",
	})
	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, actorID, ticket.AssignedStaffID)
	require.Equal(t, ticketID, repository.claimRecord.TicketID)
	require.Equal(t, actorID, repository.claimRecord.StaffID)
	require.Equal(t, 2, repository.claimRecord.ExpectedVersion)
	require.Equal(t, "Taking ownership", repository.claimRecord.Reason)
	require.Equal(t, "request-claim", repository.claimRecord.RequestID)
	require.Equal(t, now, repository.claimRecord.ClaimedAt)
	require.Equal(t, now.Add(24*time.Hour), repository.claimRecord.IdempotencyExpiry)
	require.NotEqual(t, uuid.Nil, repository.claimRecord.AssignmentID)
	require.NotEmpty(t, repository.claimRecord.RequestHash)
}

func TestStaffClaimRejectsInvalidInput(t *testing.T) {
	service := NewStaffService(&fakeStaffTicketRepository{})
	base := ClaimInput{
		ActorID: uuid.New(), TicketID: uuid.New(), ExpectedVersion: 1,
		Reason: "Taking ownership", IdempotencyKey: "claim-key-2",
	}

	tests := []ClaimInput{
		func() ClaimInput { value := base; value.ActorID = uuid.Nil; return value }(),
		func() ClaimInput { value := base; value.TicketID = uuid.Nil; return value }(),
		func() ClaimInput { value := base; value.ExpectedVersion = 0; return value }(),
		func() ClaimInput { value := base; value.Reason = " "; return value }(),
		func() ClaimInput { value := base; value.IdempotencyKey = ""; return value }(),
	}
	for _, input := range tests {
		_, _, err := service.Claim(context.Background(), input)
		require.Error(t, err)
	}
}

func TestClaimRequestDigestIgnoresTransportMetadata(t *testing.T) {
	input := ClaimInput{
		ActorID: uuid.New(), TicketID: uuid.New(), ExpectedVersion: 3,
		Reason: "Take ownership", RequestID: "request-one", IdempotencyKey: "key-one",
	}
	other := input
	other.RequestID = "request-two"
	other.IdempotencyKey = "key-two"
	require.Equal(t, claimRequestDigest(input), claimRequestDigest(other))

	other.Reason = "Different business request"
	require.NotEqual(t, claimRequestDigest(input), claimRequestDigest(other))
}

func TestStaffAddInternalNoteBuildsPrivateNote(t *testing.T) {
	repository := &fakeStaffTicketRepository{}
	service := NewStaffService(repository)
	now := time.Date(2026, time.October, 10, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	actorID := uuid.New()
	ticketID := uuid.New()

	note, created, err := service.AddInternalNote(context.Background(), AddInternalNoteInput{
		ActorID: actorID, TicketID: ticketID, Content: "  Staff-only context  ",
		RequestID: " note-request ", IdempotencyKey: "note-key-1",
	})
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, "Staff-only context", note.Content)
	require.Equal(t, actorID, note.AuthorID)
	require.Equal(t, ticketID, note.TicketID)
	require.Equal(t, now, note.CreatedAt)
	require.Equal(t, "note-request", repository.noteRecord.RequestID)
	require.NotEmpty(t, repository.noteRecord.RequestHash)
}

func TestInternalNoteDigestIgnoresTransportMetadata(t *testing.T) {
	input := AddInternalNoteInput{
		ActorID: uuid.New(), TicketID: uuid.New(), Content: "Private note",
		RequestID: "request-one", IdempotencyKey: "key-one",
	}
	other := input
	other.RequestID = "request-two"
	other.IdempotencyKey = "key-two"
	require.Equal(t, internalNoteRequestDigest(input), internalNoteRequestDigest(other))

	other.Content = "Different private note"
	require.NotEqual(t, internalNoteRequestDigest(input), internalNoteRequestDigest(other))
}

func TestStaffAddInternalNoteRejectsInvalidInput(t *testing.T) {
	service := NewStaffService(&fakeStaffTicketRepository{})
	base := AddInternalNoteInput{
		ActorID: uuid.New(), TicketID: uuid.New(), Content: "Private note", IdempotencyKey: "note-key-2",
	}
	tests := []AddInternalNoteInput{
		func() AddInternalNoteInput { value := base; value.ActorID = uuid.Nil; return value }(),
		func() AddInternalNoteInput { value := base; value.TicketID = uuid.Nil; return value }(),
		func() AddInternalNoteInput { value := base; value.Content = " "; return value }(),
		func() AddInternalNoteInput { value := base; value.IdempotencyKey = ""; return value }(),
	}
	for _, input := range tests {
		_, _, err := service.AddInternalNote(context.Background(), input)
		require.Error(t, err)
	}
}

func TestStaffAddReplyBuildsPublicOutboundMessage(t *testing.T) {
	repository := &fakeStaffTicketRepository{}
	service := NewStaffService(repository)
	now := time.Date(2026, time.October, 10, 9, 30, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	actorID := uuid.New()
	ticketID := uuid.New()

	message, created, err := service.AddReply(context.Background(), AddStaffReplyInput{
		ActorID: actorID, TicketID: ticketID, Content: "  We are checking your order  ",
		RequestID: " reply-request ", IdempotencyKey: "reply-key-1",
	})
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, "We are checking your order", message.Content)
	require.Equal(t, "STAFF", message.SenderType)
	require.Equal(t, "OUTBOUND", message.Direction)
	require.Equal(t, "STORED", message.DeliveryStatus)
	require.Equal(t, actorID, message.SenderID)
	require.Equal(t, now, message.CreatedAt)
	require.Equal(t, "reply-request", repository.replyRecord.RequestID)
	require.NotEmpty(t, repository.replyRecord.RequestHash)
}

func TestStaffAddReplyRejectsInvalidInput(t *testing.T) {
	service := NewStaffService(&fakeStaffTicketRepository{})
	base := AddStaffReplyInput{
		ActorID: uuid.New(), TicketID: uuid.New(), Content: "Public reply", IdempotencyKey: "reply-key-2",
	}
	tests := []AddStaffReplyInput{
		func() AddStaffReplyInput { value := base; value.ActorID = uuid.Nil; return value }(),
		func() AddStaffReplyInput { value := base; value.TicketID = uuid.Nil; return value }(),
		func() AddStaffReplyInput { value := base; value.Content = " "; return value }(),
		func() AddStaffReplyInput { value := base; value.IdempotencyKey = ""; return value }(),
	}
	for _, input := range tests {
		_, _, err := service.AddReply(context.Background(), input)
		require.Error(t, err)
	}
}

func TestStaffReplyDigestIgnoresTransportMetadata(t *testing.T) {
	input := AddStaffReplyInput{
		ActorID: uuid.New(), TicketID: uuid.New(), Content: "Public reply",
		RequestID: "request-one", IdempotencyKey: "key-one",
	}
	other := input
	other.RequestID = "request-two"
	other.IdempotencyKey = "key-two"
	require.Equal(t, staffReplyRequestDigest(input), staffReplyRequestDigest(other))

	other.Content = "Changed public reply"
	require.NotEqual(t, staffReplyRequestDigest(input), staffReplyRequestDigest(other))
}

func TestStaffTransitionBuildsVersionedRecord(t *testing.T) {
	repository := &fakeStaffTicketRepository{}
	service := NewStaffService(repository, 72*time.Hour)
	now := time.Date(2026, time.October, 10, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	actorID := uuid.New()
	ticketID := uuid.New()

	ticket, applied, err := service.Transition(context.Background(), TransitionInput{
		ActorID: actorID, TicketID: ticketID, ExpectedVersion: 4,
		ToStatus: domain.TicketStatusWaitingCustomer, Reason: "  Need customer evidence  ",
		RequestID: " transition-request ", IdempotencyKey: "transition-key-1",
	})
	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, domain.TicketStatusWaitingCustomer, ticket.Status)
	require.Equal(t, actorID, repository.transitionRecord.ActorID)
	require.Equal(t, ticketID, repository.transitionRecord.TicketID)
	require.Equal(t, 4, repository.transitionRecord.ExpectedVersion)
	require.Equal(t, "Need customer evidence", repository.transitionRecord.Reason)
	require.Equal(t, now, repository.transitionRecord.TransitionedAt)
	require.Equal(t, 72*time.Hour, repository.transitionRecord.ReopenWindow)
	require.NotEmpty(t, repository.transitionRecord.RequestHash)
}

func TestStaffTransitionRejectsInvalidInput(t *testing.T) {
	service := NewStaffService(&fakeStaffTicketRepository{})
	base := TransitionInput{
		ActorID: uuid.New(), TicketID: uuid.New(), ExpectedVersion: 1,
		ToStatus: domain.TicketStatusInProgress, Reason: "Start work", IdempotencyKey: "transition-key-2",
	}
	tests := []TransitionInput{
		func() TransitionInput { value := base; value.ActorID = uuid.Nil; return value }(),
		func() TransitionInput { value := base; value.TicketID = uuid.Nil; return value }(),
		func() TransitionInput { value := base; value.ExpectedVersion = 0; return value }(),
		func() TransitionInput { value := base; value.ToStatus = domain.TicketStatusAssigned; return value }(),
		func() TransitionInput { value := base; value.Reason = " "; return value }(),
		func() TransitionInput { value := base; value.IdempotencyKey = ""; return value }(),
	}
	for _, input := range tests {
		_, _, err := service.Transition(context.Background(), input)
		require.Error(t, err)
	}
}

func TestTransitionDigestIgnoresTransportMetadata(t *testing.T) {
	input := TransitionInput{
		ActorID: uuid.New(), TicketID: uuid.New(), ExpectedVersion: 2,
		ToStatus: domain.TicketStatusInProgress, Reason: "Start work",
		RequestID: "request-one", IdempotencyKey: "key-one",
	}
	other := input
	other.RequestID = "request-two"
	other.IdempotencyKey = "key-two"
	require.Equal(t, transitionRequestDigest(input), transitionRequestDigest(other))

	other.ToStatus = domain.TicketStatusResolved
	require.NotEqual(t, transitionRequestDigest(input), transitionRequestDigest(other))
}
