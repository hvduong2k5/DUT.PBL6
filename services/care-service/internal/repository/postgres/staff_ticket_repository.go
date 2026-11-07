package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/care-service/internal/domain"
	"github.com/omamx/care-service/internal/idgen"
	"github.com/omamx/care-service/internal/requestcontext"
	"go.opentelemetry.io/otel/trace"
)

func (r *TicketRepository) ListForStaff(ctx context.Context, filter domain.StaffTicketFilter) (domain.TicketPage, error) {
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}

	where := make([]string, 0, 8)
	args := make([]any, 0, 10)
	addArgument := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if filter.Search != "" {
		placeholder := addArgument("%" + filter.Search + "%")
		where = append(where, fmt.Sprintf(
			"(code ILIKE %s OR subject ILIKE %s OR COALESCE(order_id, '') ILIKE %s)",
			placeholder, placeholder, placeholder,
		))
	}
	if filter.Status != "" {
		where = append(where, "status = "+addArgument(filter.Status))
	}
	if filter.Priority != "" {
		where = append(where, "priority = "+addArgument(filter.Priority))
	}
	if filter.Category != "" {
		where = append(where, "category = "+addArgument(filter.Category))
	}
	if filter.AssignedStaffID != uuid.Nil {
		where = append(where, "assigned_staff_id = "+addArgument(filter.AssignedStaffID))
	}
	if filter.AssignedQueue != "" {
		where = append(where, "assigned_queue = "+addArgument(filter.AssignedQueue))
	}
	if filter.Unassigned {
		where = append(where, "assigned_staff_id IS NULL")
	}
	if filter.Cursor != "" {
		updatedAt, id, err := decodeCursor(filter.Cursor)
		if err != nil {
			return domain.TicketPage{}, fmt.Errorf("%w: invalid cursor", domain.ErrInvalidArgument)
		}
		updatedPlaceholder := addArgument(updatedAt)
		idPlaceholder := addArgument(id)
		where = append(where, fmt.Sprintf("(updated_at, id) < (%s, %s)", updatedPlaceholder, idPlaceholder))
	}

	query := `SELECT ` + ticketColumns + ` FROM support_tickets`
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, ` AND `)
	}
	query += ` ORDER BY updated_at DESC, id DESC LIMIT ` + addArgument(limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return domain.TicketPage{}, fmt.Errorf("list staff tickets: %w", err)
	}
	defer rows.Close()

	items := make([]domain.SupportTicket, 0, limit+1)
	for rows.Next() {
		ticket, scanErr := scanTicket(rows)
		if scanErr != nil {
			return domain.TicketPage{}, scanErr
		}
		items = append(items, ticket)
	}
	if err := rows.Err(); err != nil {
		return domain.TicketPage{}, err
	}

	nextCursor := ""
	if len(items) > limit {
		last := items[limit-1]
		nextCursor = encodeCursor(last.UpdatedAt, last.ID)
		items = items[:limit]
	}
	return domain.TicketPage{Items: items, NextCursor: nextCursor}, nil
}

func (r *TicketRepository) GetForStaff(ctx context.Context, access domain.StaffTicketAccess) (domain.StaffTicketDetail, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.StaffTicketDetail{}, err
	}
	defer tx.Rollback(ctx)

	ticket, err := scanTicket(tx.QueryRow(ctx, `SELECT `+ticketColumns+` FROM support_tickets WHERE id = $1`, access.TicketID))
	if err == pgx.ErrNoRows {
		return domain.StaffTicketDetail{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.StaffTicketDetail{}, fmt.Errorf("get staff ticket: %w", err)
	}

	messages, err := listTicketMessages(ctx, tx, ticket.ID)
	if err != nil {
		return domain.StaffTicketDetail{}, err
	}
	notes, err := listInternalNotes(ctx, tx, ticket.ID)
	if err != nil {
		return domain.StaffTicketDetail{}, err
	}
	assignments, err := listAssignments(ctx, tx, ticket.ID)
	if err != nil {
		return domain.StaffTicketDetail{}, err
	}
	history, err := listStatusHistory(ctx, tx, ticket.ID)
	if err != nil {
		return domain.StaffTicketDetail{}, err
	}
	if err := insertTicketAccessAudit(ctx, tx, access); err != nil {
		return domain.StaffTicketDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.StaffTicketDetail{}, err
	}
	return domain.StaffTicketDetail{
		Ticket: ticket, Messages: messages, InternalNotes: notes,
		Assignments: assignments, StatusHistory: history,
	}, nil
}

func (r *TicketRepository) ClaimForStaff(ctx context.Context, record domain.ClaimTicketRecord) (domain.SupportTicket, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.SupportTicket{}, false, err
	}
	defer tx.Rollback(ctx)

	operation := "ticket.claim:" + record.TicketID.String()
	resourceID, applied, err := claimIdempotency(
		ctx, tx, record.StaffID, operation, record.IdempotencyKey,
		record.RequestHash, record.IdempotencyExpiry,
	)
	if err != nil {
		return domain.SupportTicket{}, false, err
	}
	if !applied {
		ticketID, parseErr := uuid.Parse(resourceID)
		if parseErr != nil {
			return domain.SupportTicket{}, false, parseErr
		}
		ticket, getErr := scanTicket(tx.QueryRow(ctx, `SELECT `+ticketColumns+` FROM support_tickets WHERE id = $1`, ticketID))
		if getErr == pgx.ErrNoRows {
			return domain.SupportTicket{}, false, domain.ErrNotFound
		}
		if getErr != nil {
			return domain.SupportTicket{}, false, getErr
		}
		if err := tx.Commit(ctx); err != nil {
			return domain.SupportTicket{}, false, err
		}
		return ticket, false, nil
	}

	var previousStatus domain.TicketStatus
	var previousVersion int
	var previousStaffID uuid.UUID
	var assignedQueue string
	err = tx.QueryRow(ctx, `
		SELECT status, version,
		       COALESCE(assigned_staff_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       COALESCE(assigned_queue, '')
		FROM support_tickets
		WHERE id = $1
		FOR UPDATE
	`, record.TicketID).Scan(&previousStatus, &previousVersion, &previousStaffID, &assignedQueue)
	if err == pgx.ErrNoRows {
		return domain.SupportTicket{}, false, domain.ErrNotFound
	}
	if err != nil {
		return domain.SupportTicket{}, false, fmt.Errorf("lock ticket for claim: %w", err)
	}
	if previousVersion != record.ExpectedVersion {
		return domain.SupportTicket{}, false, fmt.Errorf(
			"%w: expected ticket version %d but current version is %d",
			domain.ErrConflict, record.ExpectedVersion, previousVersion,
		)
	}
	if previousStaffID != uuid.Nil {
		return domain.SupportTicket{}, false, fmt.Errorf("%w: ticket is already assigned", domain.ErrConflict)
	}
	if previousStatus != domain.TicketStatusNew && previousStatus != domain.TicketStatusTriaged {
		return domain.SupportTicket{}, false, fmt.Errorf("%w: only NEW or TRIAGED tickets can be claimed", domain.ErrConflict)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE ticket_assignments
		SET ended_at = $2
		WHERE ticket_id = $1 AND ended_at IS NULL
	`, record.TicketID, record.ClaimedAt); err != nil {
		return domain.SupportTicket{}, false, fmt.Errorf("close previous ticket assignment: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO ticket_assignments (
			id, ticket_id, staff_id, queue, assigned_by, reason, assigned_at
		) VALUES ($1,$2,$3,NULLIF($4,''),$3,$5,$6)
	`, record.AssignmentID, record.TicketID, record.StaffID, assignedQueue, record.Reason, record.ClaimedAt); err != nil {
		return domain.SupportTicket{}, false, fmt.Errorf("insert ticket claim assignment: %w", err)
	}

	ticket, err := scanTicket(tx.QueryRow(ctx, `
		UPDATE support_tickets
		SET assigned_staff_id = $2, status = 'ASSIGNED', version = version + 1, updated_at = $3
		WHERE id = $1 AND version = $4 AND assigned_staff_id IS NULL
		RETURNING `+ticketColumns,
		record.TicketID, record.StaffID, record.ClaimedAt, record.ExpectedVersion,
	))
	if err == pgx.ErrNoRows {
		return domain.SupportTicket{}, false, fmt.Errorf("%w: ticket changed while it was being claimed", domain.ErrConflict)
	}
	if err != nil {
		return domain.SupportTicket{}, false, fmt.Errorf("claim ticket: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO ticket_status_history (id, ticket_id, from_status, to_status, actor_id, reason, created_at)
		VALUES ($1,$2,$3,'ASSIGNED',$4,$5,$6)
	`, idgen.New(), record.TicketID, previousStatus, record.StaffID, record.Reason, record.ClaimedAt); err != nil {
		return domain.SupportTicket{}, false, fmt.Errorf("insert claim status history: %w", err)
	}

	beforeState, _ := json.Marshal(map[string]any{
		"status": previousStatus, "assigned_staff_id": nil, "assigned_queue": nullableAuditString(assignedQueue),
		"version": previousVersion,
	})
	afterState, _ := json.Marshal(map[string]any{
		"status": ticket.Status, "assigned_staff_id": ticket.AssignedStaffID,
		"assigned_queue": nullableAuditString(ticket.AssignedQueue), "version": ticket.Version,
	})
	traceID := ""
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		traceID = spanContext.TraceID().String()
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (
			id, actor_id, action, resource_type, resource_id, reason,
			before_state, after_state, request_id, trace_id, correlation_id
		) VALUES ($1,$2,'TICKET_CLAIMED','SUPPORT_TICKET',$3,$4,$5::jsonb,$6::jsonb,NULLIF($7,''),NULLIF($8,''),NULLIF($9,''))
	`, idgen.New(), record.StaffID, record.TicketID.String(), record.Reason,
		string(beforeState), string(afterState), record.RequestID, traceID, requestcontext.CorrelationID(ctx)); err != nil {
		return domain.SupportTicket{}, false, fmt.Errorf("audit ticket claim: %w", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"ticket_id": ticket.ID, "assigned_staff_id": ticket.AssignedStaffID,
		"assigned_queue": nullableAuditString(ticket.AssignedQueue), "status": ticket.Status,
		"version": ticket.Version, "assigned_at": record.ClaimedAt,
	})
	if err := insertOutbox(
		ctx, tx, ticket.ID, "SUPPORT_TICKET", "vn.omama.care.ticket.assigned.v1",
		r.topic, ticket.ID.String(), payload,
	); err != nil {
		return domain.SupportTicket{}, false, err
	}
	if err := completeIdempotency(
		ctx, tx, record.StaffID, operation, record.IdempotencyKey, ticket.ID.String(), 200,
	); err != nil {
		return domain.SupportTicket{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.SupportTicket{}, false, fmt.Errorf("commit ticket claim: %w", err)
	}
	return ticket, true, nil
}

func nullableAuditString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (r *TicketRepository) AddInternalNote(ctx context.Context, record domain.AddInternalNoteRecord) (domain.TicketInternalNote, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.TicketInternalNote{}, false, err
	}
	defer tx.Rollback(ctx)

	operation := "ticket.add-internal-note:" + record.Note.TicketID.String()
	resourceID, created, err := claimIdempotency(
		ctx, tx, record.ActorID, operation, record.IdempotencyKey,
		record.RequestHash, record.IdempotencyExpiry,
	)
	if err != nil {
		return domain.TicketInternalNote{}, false, err
	}
	if !created {
		noteID, parseErr := uuid.Parse(resourceID)
		if parseErr != nil {
			return domain.TicketInternalNote{}, false, parseErr
		}
		note, getErr := getInternalNote(ctx, tx, record.Note.TicketID, noteID)
		if getErr != nil {
			return domain.TicketInternalNote{}, false, getErr
		}
		if err := tx.Commit(ctx); err != nil {
			return domain.TicketInternalNote{}, false, err
		}
		return note, false, nil
	}

	var status domain.TicketStatus
	err = tx.QueryRow(ctx, `SELECT status FROM support_tickets WHERE id = $1 FOR UPDATE`, record.Note.TicketID).Scan(&status)
	if err == pgx.ErrNoRows {
		return domain.TicketInternalNote{}, false, domain.ErrNotFound
	}
	if err != nil {
		return domain.TicketInternalNote{}, false, fmt.Errorf("lock ticket for internal note: %w", err)
	}
	if status == domain.TicketStatusClosed {
		return domain.TicketInternalNote{}, false, fmt.Errorf("%w: closed ticket does not accept internal notes", domain.ErrConflict)
	}

	note := record.Note
	if _, err := tx.Exec(ctx, `
		INSERT INTO ticket_internal_notes (id, ticket_id, author_id, content, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$5)
	`, note.ID, note.TicketID, note.AuthorID, note.Content, note.CreatedAt); err != nil {
		return domain.TicketInternalNote{}, false, fmt.Errorf("insert ticket internal note: %w", err)
	}

	afterState, _ := json.Marshal(map[string]any{"note_id": note.ID, "author_id": note.AuthorID})
	traceID := ""
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		traceID = spanContext.TraceID().String()
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (
			id, actor_id, action, resource_type, resource_id,
			after_state, request_id, trace_id, correlation_id
		) VALUES ($1,$2,'TICKET_INTERNAL_NOTE_ADDED','SUPPORT_TICKET',$3,$4::jsonb,NULLIF($5,''),NULLIF($6,''),NULLIF($7,''))
	`, idgen.New(), record.ActorID, note.TicketID.String(), string(afterState), record.RequestID, traceID,
		requestcontext.CorrelationID(ctx)); err != nil {
		return domain.TicketInternalNote{}, false, fmt.Errorf("audit ticket internal note: %w", err)
	}
	if err := completeIdempotency(
		ctx, tx, record.ActorID, operation, record.IdempotencyKey, note.ID.String(), 201,
	); err != nil {
		return domain.TicketInternalNote{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.TicketInternalNote{}, false, fmt.Errorf("commit ticket internal note: %w", err)
	}
	return note, true, nil
}

func getInternalNote(ctx context.Context, db ticketQuerier, ticketID, noteID uuid.UUID) (domain.TicketInternalNote, error) {
	var note domain.TicketInternalNote
	err := db.QueryRow(ctx, `
		SELECT id, ticket_id, author_id, content, created_at, updated_at
		FROM ticket_internal_notes
		WHERE id = $1 AND ticket_id = $2
	`, noteID, ticketID).Scan(
		&note.ID, &note.TicketID, &note.AuthorID, &note.Content, &note.CreatedAt, &note.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return domain.TicketInternalNote{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.TicketInternalNote{}, fmt.Errorf("get ticket internal note: %w", err)
	}
	return note, nil
}

func (r *TicketRepository) AddStaffReply(ctx context.Context, record domain.AddStaffReplyRecord) (domain.TicketMessage, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.TicketMessage{}, false, err
	}
	defer tx.Rollback(ctx)

	operation := "ticket.add-staff-reply:" + record.Message.TicketID.String()
	resourceID, created, err := claimIdempotency(
		ctx, tx, record.ActorID, operation, record.IdempotencyKey,
		record.RequestHash, record.IdempotencyExpiry,
	)
	if err != nil {
		return domain.TicketMessage{}, false, err
	}
	if !created {
		messageID, parseErr := uuid.Parse(resourceID)
		if parseErr != nil {
			return domain.TicketMessage{}, false, parseErr
		}
		message, getErr := getStaffReply(ctx, tx, record.Message.TicketID, messageID)
		if getErr != nil {
			return domain.TicketMessage{}, false, getErr
		}
		if err := tx.Commit(ctx); err != nil {
			return domain.TicketMessage{}, false, err
		}
		return message, false, nil
	}

	current, err := scanTicket(tx.QueryRow(ctx, `SELECT `+ticketColumns+` FROM support_tickets WHERE id = $1 FOR UPDATE`, record.Message.TicketID))
	if err == pgx.ErrNoRows {
		return domain.TicketMessage{}, false, domain.ErrNotFound
	}
	if err != nil {
		return domain.TicketMessage{}, false, fmt.Errorf("lock ticket for staff reply: %w", err)
	}
	if current.AssignedStaffID != record.ActorID {
		return domain.TicketMessage{}, false, domain.ErrForbidden
	}
	switch current.Status {
	case domain.TicketStatusAssigned, domain.TicketStatusInProgress,
		domain.TicketStatusWaitingCustomer, domain.TicketStatusWaitingInternal:
	default:
		return domain.TicketMessage{}, false, fmt.Errorf("%w: ticket status %s does not accept staff replies", domain.ErrConflict, current.Status)
	}

	message := record.Message
	if _, err := tx.Exec(ctx, `
		INSERT INTO ticket_messages (
			id, ticket_id, sender_id, sender_type, direction, delivery_status, content, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, message.ID, message.TicketID, message.SenderID, message.SenderType,
		message.Direction, message.DeliveryStatus, message.Content, message.CreatedAt); err != nil {
		return domain.TicketMessage{}, false, fmt.Errorf("insert staff ticket reply: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE support_tickets
		SET first_responded_at = COALESCE(first_responded_at, $2), updated_at = $2
		WHERE id = $1
	`, message.TicketID, message.CreatedAt); err != nil {
		return domain.TicketMessage{}, false, fmt.Errorf("set ticket first response: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE ticket_sla_milestones
		SET completed_at = COALESCE(completed_at, $2),
		    breached_at = CASE WHEN $2 > due_at THEN COALESCE(breached_at, $2) ELSE breached_at END,
		    updated_at = $2
		WHERE ticket_id = $1 AND milestone_type = 'FIRST_RESPONSE' AND completed_at IS NULL
	`, message.TicketID, message.CreatedAt); err != nil {
		return domain.TicketMessage{}, false, fmt.Errorf("complete ticket first-response SLA: %w", err)
	}

	afterState, _ := json.Marshal(map[string]any{
		"message_id": message.ID, "author_id": message.SenderID,
		"direction": message.Direction, "delivery_status": message.DeliveryStatus,
	})
	traceID := ""
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		traceID = spanContext.TraceID().String()
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (
			id, actor_id, action, resource_type, resource_id,
			after_state, request_id, trace_id, correlation_id
		) VALUES ($1,$2,'TICKET_STAFF_REPLY_ADDED','SUPPORT_TICKET',$3,$4::jsonb,NULLIF($5,''),NULLIF($6,''),NULLIF($7,''))
	`, idgen.New(), record.ActorID, message.TicketID.String(), string(afterState), record.RequestID, traceID,
		requestcontext.CorrelationID(ctx)); err != nil {
		return domain.TicketMessage{}, false, fmt.Errorf("audit staff ticket reply: %w", err)
	}
	payload, _ := json.Marshal(map[string]any{
		"ticket_id": message.TicketID, "message_id": message.ID, "staff_id": record.ActorID,
		"direction": message.Direction, "delivery_status": message.DeliveryStatus,
		"created_at": message.CreatedAt,
	})
	if err := insertOutbox(
		ctx, tx, message.TicketID, "SUPPORT_TICKET", "vn.omama.care.ticket.staff-message.created.v1",
		r.topic, message.TicketID.String(), payload,
	); err != nil {
		return domain.TicketMessage{}, false, err
	}
	if err := completeIdempotency(
		ctx, tx, record.ActorID, operation, record.IdempotencyKey, message.ID.String(), 201,
	); err != nil {
		return domain.TicketMessage{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.TicketMessage{}, false, fmt.Errorf("commit staff ticket reply: %w", err)
	}
	return message, true, nil
}

func getStaffReply(ctx context.Context, db ticketQuerier, ticketID, messageID uuid.UUID) (domain.TicketMessage, error) {
	var message domain.TicketMessage
	err := db.QueryRow(ctx, `
		SELECT id, ticket_id, sender_id, sender_type, direction, delivery_status, content, created_at
		FROM ticket_messages
		WHERE id = $1 AND ticket_id = $2 AND sender_type = 'STAFF'
	`, messageID, ticketID).Scan(
		&message.ID, &message.TicketID, &message.SenderID, &message.SenderType,
		&message.Direction, &message.DeliveryStatus, &message.Content, &message.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return domain.TicketMessage{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.TicketMessage{}, fmt.Errorf("get staff ticket reply: %w", err)
	}
	message.FileIDs = []uuid.UUID{}
	return message, nil
}

func (r *TicketRepository) TransitionForStaff(ctx context.Context, record domain.TransitionTicketRecord) (domain.SupportTicket, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.SupportTicket{}, false, err
	}
	defer tx.Rollback(ctx)

	operation := "ticket.transition:" + record.TicketID.String()
	resourceID, applied, err := claimIdempotency(
		ctx, tx, record.ActorID, operation, record.IdempotencyKey,
		record.RequestHash, record.IdempotencyExpiry,
	)
	if err != nil {
		return domain.SupportTicket{}, false, err
	}
	if !applied {
		ticketID, parseErr := uuid.Parse(resourceID)
		if parseErr != nil {
			return domain.SupportTicket{}, false, parseErr
		}
		ticket, getErr := scanTicket(tx.QueryRow(ctx, `SELECT `+ticketColumns+` FROM support_tickets WHERE id = $1`, ticketID))
		if getErr == pgx.ErrNoRows {
			return domain.SupportTicket{}, false, domain.ErrNotFound
		}
		if getErr != nil {
			return domain.SupportTicket{}, false, getErr
		}
		if err := tx.Commit(ctx); err != nil {
			return domain.SupportTicket{}, false, err
		}
		return ticket, false, nil
	}

	current, err := scanTicket(tx.QueryRow(ctx, `SELECT `+ticketColumns+` FROM support_tickets WHERE id = $1 FOR UPDATE`, record.TicketID))
	if err == pgx.ErrNoRows {
		return domain.SupportTicket{}, false, domain.ErrNotFound
	}
	if err != nil {
		return domain.SupportTicket{}, false, fmt.Errorf("lock ticket for transition: %w", err)
	}
	if current.Version != record.ExpectedVersion {
		return domain.SupportTicket{}, false, fmt.Errorf(
			"%w: expected ticket version %d but current version is %d",
			domain.ErrConflict, record.ExpectedVersion, current.Version,
		)
	}
	if !current.Status.CanTransitionTo(record.ToStatus) {
		return domain.SupportTicket{}, false, fmt.Errorf(
			"%w: transition from %s to %s is not allowed", domain.ErrConflict, current.Status, record.ToStatus,
		)
	}
	if current.AssignedStaffID != uuid.Nil && current.AssignedStaffID != record.ActorID {
		return domain.SupportTicket{}, false, domain.ErrForbidden
	}
	if current.Status != domain.TicketStatusNew && current.AssignedStaffID == uuid.Nil {
		return domain.SupportTicket{}, false, fmt.Errorf("%w: ticket must be assigned before this transition", domain.ErrConflict)
	}
	if current.Status == domain.TicketStatusResolved && record.ToStatus == domain.TicketStatusInProgress {
		if current.ResolvedAt == nil || record.TransitionedAt.Sub(*current.ResolvedAt) > record.ReopenWindow {
			return domain.SupportTicket{}, false, fmt.Errorf("%w: ticket reopen window has expired", domain.ErrConflict)
		}
	}

	if err := updateResolutionSLAForTransition(ctx, tx, current.Status, record.ToStatus, record.TicketID, record.TransitionedAt); err != nil {
		return domain.SupportTicket{}, false, err
	}

	ticket, err := scanTicket(tx.QueryRow(ctx, `
		UPDATE support_tickets
		SET status = $2::varchar,
		    version = version + 1,
		    resolved_at = CASE
		        WHEN $2::varchar = 'RESOLVED' THEN $3::timestamptz
		        WHEN status = 'RESOLVED' AND $2::varchar = 'IN_PROGRESS' THEN NULL
		        ELSE resolved_at
		    END,
		    closed_at = CASE WHEN $2::varchar = 'CLOSED' THEN $3::timestamptz ELSE closed_at END,
		    updated_at = $3::timestamptz
		WHERE id = $1 AND version = $4
		RETURNING `+ticketColumns,
		record.TicketID, record.ToStatus, record.TransitionedAt, record.ExpectedVersion,
	))
	if err == pgx.ErrNoRows {
		return domain.SupportTicket{}, false, fmt.Errorf("%w: ticket changed while transitioning", domain.ErrConflict)
	}
	if err != nil {
		return domain.SupportTicket{}, false, fmt.Errorf("transition ticket: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO ticket_status_history (id, ticket_id, from_status, to_status, actor_id, reason, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, idgen.New(), ticket.ID, current.Status, ticket.Status, record.ActorID, record.Reason, record.TransitionedAt); err != nil {
		return domain.SupportTicket{}, false, fmt.Errorf("insert transition history: %w", err)
	}

	beforeState, _ := json.Marshal(map[string]any{"status": current.Status, "version": current.Version})
	afterState, _ := json.Marshal(map[string]any{"status": ticket.Status, "version": ticket.Version})
	traceID := ""
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		traceID = spanContext.TraceID().String()
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (
			id, actor_id, action, resource_type, resource_id, reason,
			before_state, after_state, request_id, trace_id, correlation_id
		) VALUES ($1,$2,'TICKET_STATUS_CHANGED','SUPPORT_TICKET',$3,$4,$5::jsonb,$6::jsonb,NULLIF($7,''),NULLIF($8,''),NULLIF($9,''))
	`, idgen.New(), record.ActorID, ticket.ID.String(), record.Reason,
		string(beforeState), string(afterState), record.RequestID, traceID, requestcontext.CorrelationID(ctx)); err != nil {
		return domain.SupportTicket{}, false, fmt.Errorf("audit ticket transition: %w", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"ticket_id": ticket.ID, "from_status": current.Status, "to_status": ticket.Status,
		"actor_id": record.ActorID, "reason": record.Reason,
		"version": ticket.Version, "changed_at": record.TransitionedAt,
	})
	if err := insertOutbox(
		ctx, tx, ticket.ID, "SUPPORT_TICKET", "vn.omama.care.ticket.status-changed.v1",
		r.topic, ticket.ID.String(), payload,
	); err != nil {
		return domain.SupportTicket{}, false, err
	}
	if err := completeIdempotency(
		ctx, tx, record.ActorID, operation, record.IdempotencyKey, ticket.ID.String(), 200,
	); err != nil {
		return domain.SupportTicket{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.SupportTicket{}, false, fmt.Errorf("commit ticket transition: %w", err)
	}
	return ticket, true, nil
}

func updateResolutionSLAForTransition(
	ctx context.Context,
	tx pgx.Tx,
	fromStatus, toStatus domain.TicketStatus,
	ticketID uuid.UUID,
	transitionedAt time.Time,
) error {
	if fromStatus == domain.TicketStatusWaitingCustomer {
		if _, err := tx.Exec(ctx, `
			UPDATE ticket_sla_milestones
			SET due_at = due_at + ($2::timestamptz - paused_at),
			    total_paused_seconds = total_paused_seconds + GREATEST(0, EXTRACT(EPOCH FROM ($2::timestamptz - paused_at))::bigint),
			    paused_at = NULL,
			    updated_at = $2
			WHERE ticket_id = $1 AND milestone_type = 'RESOLUTION'
			  AND completed_at IS NULL AND paused_at IS NOT NULL
		`, ticketID, transitionedAt); err != nil {
			return fmt.Errorf("resume ticket resolution SLA: %w", err)
		}
	}
	if toStatus == domain.TicketStatusWaitingCustomer {
		if _, err := tx.Exec(ctx, `
			UPDATE ticket_sla_milestones
			SET paused_at = COALESCE(paused_at, $2), updated_at = $2
			WHERE ticket_id = $1 AND milestone_type = 'RESOLUTION' AND completed_at IS NULL
		`, ticketID, transitionedAt); err != nil {
			return fmt.Errorf("pause ticket resolution SLA: %w", err)
		}
	}
	if toStatus == domain.TicketStatusResolved {
		if _, err := tx.Exec(ctx, `
			UPDATE ticket_sla_milestones
			SET completed_at = COALESCE(completed_at, $2),
			    breached_at = CASE WHEN $2 > due_at THEN COALESCE(breached_at, $2) ELSE breached_at END,
			    updated_at = $2
			WHERE ticket_id = $1 AND milestone_type = 'RESOLUTION'
		`, ticketID, transitionedAt); err != nil {
			return fmt.Errorf("complete ticket resolution SLA: %w", err)
		}
	}
	return nil
}

func listInternalNotes(ctx context.Context, db ticketQuerier, ticketID uuid.UUID) ([]domain.TicketInternalNote, error) {
	rows, err := db.Query(ctx, `
		SELECT id, ticket_id, author_id, content, created_at, updated_at
		FROM ticket_internal_notes
		WHERE ticket_id = $1
		ORDER BY created_at, id
	`, ticketID)
	if err != nil {
		return nil, fmt.Errorf("list internal notes: %w", err)
	}
	defer rows.Close()

	items := make([]domain.TicketInternalNote, 0)
	for rows.Next() {
		var item domain.TicketInternalNote
		if err := rows.Scan(&item.ID, &item.TicketID, &item.AuthorID, &item.Content, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func listAssignments(ctx context.Context, db ticketQuerier, ticketID uuid.UUID) ([]domain.TicketAssignment, error) {
	rows, err := db.Query(ctx, `
		SELECT id, ticket_id,
		       COALESCE(staff_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       COALESCE(queue, ''), assigned_by, COALESCE(reason, ''), assigned_at, ended_at
		FROM ticket_assignments
		WHERE ticket_id = $1
		ORDER BY assigned_at, id
	`, ticketID)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}
	defer rows.Close()

	items := make([]domain.TicketAssignment, 0)
	for rows.Next() {
		var item domain.TicketAssignment
		if err := rows.Scan(
			&item.ID, &item.TicketID, &item.StaffID, &item.Queue, &item.AssignedBy,
			&item.Reason, &item.AssignedAt, &item.EndedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func listStatusHistory(ctx context.Context, db ticketQuerier, ticketID uuid.UUID) ([]domain.TicketStatusChange, error) {
	rows, err := db.Query(ctx, `
		SELECT id, ticket_id, from_status, to_status, actor_id, COALESCE(reason, ''), created_at
		FROM ticket_status_history
		WHERE ticket_id = $1
		ORDER BY created_at, id
	`, ticketID)
	if err != nil {
		return nil, fmt.Errorf("list status history: %w", err)
	}
	defer rows.Close()

	items := make([]domain.TicketStatusChange, 0)
	for rows.Next() {
		var item domain.TicketStatusChange
		if err := rows.Scan(
			&item.ID, &item.TicketID, &item.FromStatus, &item.ToStatus,
			&item.ActorID, &item.Reason, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func insertTicketAccessAudit(ctx context.Context, tx pgx.Tx, access domain.StaffTicketAccess) error {
	traceID := ""
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		traceID = spanContext.TraceID().String()
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (
			id, actor_id, action, resource_type, resource_id, request_id, trace_id, correlation_id
		) VALUES ($1,$2,'TICKET_CONTEXT_VIEWED','SUPPORT_TICKET',$3,NULLIF($4,''),NULLIF($5,''),NULLIF($6,''))
	`, idgen.New(), access.ActorID, access.TicketID.String(), access.RequestID, traceID,
		requestcontext.CorrelationID(ctx))
	if err != nil {
		return fmt.Errorf("audit ticket context access: %w", err)
	}
	return nil
}
