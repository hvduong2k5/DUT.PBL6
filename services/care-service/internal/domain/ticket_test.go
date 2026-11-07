package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTicketStatusTransitions(t *testing.T) {
	allowed := [][2]TicketStatus{
		{TicketStatusNew, TicketStatusTriaged},
		{TicketStatusAssigned, TicketStatusInProgress},
		{TicketStatusInProgress, TicketStatusWaitingCustomer},
		{TicketStatusInProgress, TicketStatusWaitingInternal},
		{TicketStatusInProgress, TicketStatusResolved},
		{TicketStatusWaitingCustomer, TicketStatusInProgress},
		{TicketStatusWaitingCustomer, TicketStatusResolved},
		{TicketStatusWaitingInternal, TicketStatusInProgress},
		{TicketStatusWaitingInternal, TicketStatusResolved},
		{TicketStatusResolved, TicketStatusInProgress},
		{TicketStatusResolved, TicketStatusClosed},
	}
	for _, transition := range allowed {
		require.True(t, transition[0].CanTransitionTo(transition[1]), "%s -> %s", transition[0], transition[1])
	}

	rejected := [][2]TicketStatus{
		{TicketStatusNew, TicketStatusAssigned},
		{TicketStatusTriaged, TicketStatusAssigned},
		{TicketStatusAssigned, TicketStatusResolved},
		{TicketStatusClosed, TicketStatusInProgress},
		{TicketStatusInProgress, TicketStatusInProgress},
	}
	for _, transition := range rejected {
		require.False(t, transition[0].CanTransitionTo(transition[1]), "%s -> %s", transition[0], transition[1])
	}
}
