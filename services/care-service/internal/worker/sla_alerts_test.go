package worker

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestClassifySLAAlert(t *testing.T) {
	now := time.Date(2026, time.October, 11, 8, 0, 0, 0, time.UTC)
	window := 30 * time.Minute

	require.Equal(t, slaBreachedEvent, classifySLAAlert(now.Add(-time.Second), now, window))
	require.Equal(t, slaBreachedEvent, classifySLAAlert(now, now, window))
	require.Equal(t, slaNearBreachEvent, classifySLAAlert(now.Add(time.Minute), now, window))
	require.Equal(t, slaNearBreachEvent, classifySLAAlert(now.Add(window), now, window))
	require.Empty(t, classifySLAAlert(now.Add(window+time.Second), now, window))
}

func TestSLAAlertPayloadContainsOnlyRoutingMetadata(t *testing.T) {
	now := time.Date(2026, time.October, 11, 8, 0, 0, 0, time.UTC)
	milestone := slaMilestone{
		ID: uuid.New(), TicketID: uuid.New(), MilestoneType: "RESOLUTION",
		DueAt: now.Add(20 * time.Minute), TicketCode: "TKT-TEST",
		Priority: "HIGH", AssignedStaffID: uuid.New(),
	}

	payload := slaAlertPayload(milestone, now)
	require.Equal(t, milestone.TicketID, payload["ticket_id"])
	require.Equal(t, milestone.AssignedStaffID, payload["assigned_staff_id"])
	require.NotContains(t, payload, "content")
	require.NotContains(t, payload, "customer_id")
}
