package httpapi

import (
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/omamx/care-service/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestParseStaffTicketFilter(t *testing.T) {
	staffID := uuid.New()
	request := httptest.NewRequest("GET", "/?q=box&status=new&priority=high&category=shipping&assigned_queue=delivery&assigned_staff_id="+staffID.String()+"&limit=40", nil)

	filter, err := parseStaffTicketFilter(request)
	require.NoError(t, err)
	require.Equal(t, "box", filter.Search)
	require.Equal(t, domain.TicketStatusNew, filter.Status)
	require.Equal(t, domain.TicketPriorityHigh, filter.Priority)
	require.Equal(t, domain.TicketCategoryShipping, filter.Category)
	require.Equal(t, "delivery", filter.AssignedQueue)
	require.Equal(t, staffID, filter.AssignedStaffID)
	require.Equal(t, 40, filter.Limit)
}

func TestParseStaffTicketFilterRejectsInvalidValues(t *testing.T) {
	tests := []string{
		"/?limit=0",
		"/?limit=abc",
		"/?assigned_staff_id=not-a-uuid",
		"/?unassigned=perhaps",
	}
	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			request := httptest.NewRequest("GET", target, nil)
			_, err := parseStaffTicketFilter(request)
			require.Error(t, err)
		})
	}
}
