package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/care-service/internal/domain"
	"github.com/omamx/care-service/internal/requestcontext"
	"github.com/stretchr/testify/require"
)

func TestIntegrationMigrationsAndTicketCreateReplay(t *testing.T) {
	databaseURL := os.Getenv("CARE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CARE_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminPool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(adminPool.Close)
	require.NoError(t, adminPool.Ping(ctx))

	schema := "care_it_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = adminPool.Exec(ctx, `CREATE SCHEMA `+schema)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = adminPool.Exec(cleanupCtx, `DROP SCHEMA IF EXISTS `+schema+` CASCADE`)
	})

	config, err := pgxpool.ParseConfig(databaseURL)
	require.NoError(t, err)
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	runMigration(t, ctx, pool, "000001_platform_media_ticket.up.sql")
	runMigration(t, ctx, pool, "000002_sla_alerts.up.sql")
	runMigration(t, ctx, pool, "000003_return_foundation.up.sql")
	runMigration(t, ctx, pool, "000004_correlation_ids.up.sql")

	var alertColumnCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = $1
		  AND table_name = 'ticket_sla_milestones'
		  AND column_name IN ('near_breach_alerted_at', 'breach_alerted_at')
	`, schema).Scan(&alertColumnCount)
	require.NoError(t, err)
	require.Equal(t, 2, alertColumnCount)

	repository := NewTicketRepository(pool, "care.events.v1")
	now := time.Now().UTC().Truncate(time.Microsecond)
	customerID := uuid.New()
	ticketID := uuid.New()
	messageID := uuid.New()
	record := domain.CreateTicketRecord{
		Ticket: domain.SupportTicket{
			ID:                 ticketID,
			Code:               "CS-IT-" + strings.ToUpper(ticketID.String()[:8]),
			CustomerID:         customerID,
			Subject:            "Integration test ticket",
			Category:           domain.TicketCategoryOther,
			Priority:           domain.TicketPriorityNormal,
			Status:             domain.TicketStatusNew,
			Version:            1,
			FirstResponseDueAt: now.Add(4 * time.Hour),
			ResolutionDueAt:    now.Add(48 * time.Hour),
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		FirstMessage: domain.TicketMessage{
			ID:             messageID,
			TicketID:       ticketID,
			SenderID:       customerID,
			SenderType:     "CUSTOMER",
			Direction:      "INBOUND",
			DeliveryStatus: "STORED",
			Content:        "The integration test verifies one atomic write.",
			CreatedAt:      now,
		},
		IdempotencyKey:    "integration-create-ticket",
		RequestHash:       strings.Repeat("a", 64),
		IdempotencyExpiry: now.Add(24 * time.Hour),
	}

	operationCtx := requestcontext.WithIDs(ctx, "request-integration", "correlation-integration")
	createdDetail, created, err := repository.Create(operationCtx, record)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, ticketID, createdDetail.Ticket.ID)
	require.Len(t, createdDetail.Messages, 1)

	replayedDetail, created, err := repository.Create(operationCtx, record)
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, ticketID, replayedDetail.Ticket.ID)
	require.Len(t, replayedDetail.Messages, 1)

	conflicting := record
	conflicting.RequestHash = strings.Repeat("b", 64)
	_, _, err = repository.Create(operationCtx, conflicting)
	require.ErrorIs(t, err, domain.ErrIdempotencyConflict)

	assertRowCount(t, ctx, pool, "support_tickets", 1)
	assertRowCount(t, ctx, pool, "ticket_messages", 1)
	assertRowCount(t, ctx, pool, "ticket_status_history", 1)
	assertRowCount(t, ctx, pool, "ticket_sla_milestones", 2)
	assertRowCount(t, ctx, pool, "outbox_events", 1)
	assertRowCount(t, ctx, pool, "idempotency_records", 1)
	var correlationID string
	err = pool.QueryRow(ctx, `SELECT correlation_id FROM outbox_events LIMIT 1`).Scan(&correlationID)
	require.NoError(t, err)
	require.Equal(t, "correlation-integration", correlationID)

	staffID := uuid.New()
	claimedTicket, applied, err := repository.ClaimForStaff(operationCtx, domain.ClaimTicketRecord{
		TicketID: ticketID, StaffID: staffID, AssignmentID: uuid.New(),
		ExpectedVersion: 1, Reason: "Integration correlation verification",
		RequestID: "request-integration", ClaimedAt: now.Add(time.Minute),
		IdempotencyKey: "integration-claim-ticket", RequestHash: strings.Repeat("c", 64),
		IdempotencyExpiry: now.Add(24 * time.Hour),
	})
	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, domain.TicketStatusAssigned, claimedTicket.Status)
	require.Equal(t, staffID, claimedTicket.AssignedStaffID)

	var auditCorrelation string
	err = pool.QueryRow(ctx, `
		SELECT correlation_id FROM audit_logs
		WHERE action = 'TICKET_CLAIMED' AND resource_id = $1
	`, ticketID.String()).Scan(&auditCorrelation)
	require.NoError(t, err)
	require.Equal(t, "correlation-integration", auditCorrelation)
	var correlatedOutboxCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events WHERE correlation_id = 'correlation-integration'
	`).Scan(&correlatedOutboxCount)
	require.NoError(t, err)
	require.Equal(t, 2, correlatedOutboxCount)

	var returnTable *string
	err = pool.QueryRow(ctx, `SELECT to_regclass('return_cases')::text`).Scan(&returnTable)
	require.NoError(t, err)
	require.NotNil(t, returnTable)

	runMigration(t, ctx, pool, "000004_correlation_ids.down.sql")
	runMigration(t, ctx, pool, "000003_return_foundation.down.sql")
	err = pool.QueryRow(ctx, `SELECT to_regclass('return_cases')::text`).Scan(&returnTable)
	require.NoError(t, err)
	require.Nil(t, returnTable)

	runMigration(t, ctx, pool, "000002_sla_alerts.down.sql")
	runMigration(t, ctx, pool, "000001_platform_media_ticket.down.sql")

	var remainingTable *string
	err = pool.QueryRow(ctx, `SELECT to_regclass('support_tickets')::text`).Scan(&remainingTable)
	require.NoError(t, err)
	require.Nil(t, remainingTable)
}

func runMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool, filename string) {
	t.Helper()
	path := filepath.Join("..", "..", "..", "migrations", filename)
	contents, err := os.ReadFile(path)
	require.NoError(t, err, "read migration %s", filename)
	_, err = pool.Exec(ctx, string(contents))
	require.NoError(t, err, "execute migration %s", filename)
}

func assertRowCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string, expected int) {
	t.Helper()
	allowed := map[string]bool{
		"support_tickets":       true,
		"ticket_messages":       true,
		"ticket_status_history": true,
		"ticket_sla_milestones": true,
		"outbox_events":         true,
		"idempotency_records":   true,
	}
	if !allowed[table] {
		t.Fatalf("table %q is not allowed", table)
	}
	var actual int
	err := pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual, table)
}
