package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/care-service/internal/domain"
	"github.com/omamx/care-service/internal/idgen"
	"go.opentelemetry.io/otel/trace"
)

type TicketRepository struct {
	pool  *pgxpool.Pool
	topic string
}

func NewTicketRepository(pool *pgxpool.Pool, topic string) *TicketRepository {
	return &TicketRepository{pool: pool, topic: topic}
}

func (r *TicketRepository) Create(ctx context.Context, record domain.CreateTicketRecord) (domain.TicketDetail, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return domain.TicketDetail{}, false, err
	}
	defer tx.Rollback(ctx)

	resourceID, created, err := claimIdempotency(ctx, tx, record.Ticket.CustomerID, "ticket.create", record.IdempotencyKey, record.RequestHash, record.IdempotencyExpiry)
	if err != nil {
		return domain.TicketDetail{}, false, err
	}
	if !created {
		id, parseErr := uuid.Parse(resourceID)
		if parseErr != nil {
			return domain.TicketDetail{}, false, parseErr
		}
		detail, getErr := getTicketDetail(ctx, tx, id, record.Ticket.CustomerID)
		if getErr != nil {
			return domain.TicketDetail{}, false, getErr
		}
		if err := tx.Commit(ctx); err != nil {
			return domain.TicketDetail{}, false, err
		}
		return detail, false, nil
	}

	ticket := record.Ticket
	_, err = tx.Exec(ctx, `
		INSERT INTO support_tickets (
			id, code, customer_id, subject, category, priority, status, order_id,
			first_response_due_at, resolution_due_at, version, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9,$10,$11,$12,$12)
	`, ticket.ID, ticket.Code, ticket.CustomerID, ticket.Subject, ticket.Category, ticket.Priority,
		ticket.Status, ticket.OrderID, ticket.FirstResponseDueAt, ticket.ResolutionDueAt,
		ticket.Version, ticket.CreatedAt)
	if err != nil {
		return domain.TicketDetail{}, false, fmt.Errorf("insert ticket: %w", err)
	}

	message := record.FirstMessage
	if _, err := tx.Exec(ctx, `
		INSERT INTO ticket_messages (id, ticket_id, sender_id, sender_type, direction, content, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, message.ID, message.TicketID, message.SenderID, message.SenderType, message.Direction, message.Content, message.CreatedAt); err != nil {
		return domain.TicketDetail{}, false, fmt.Errorf("insert first ticket message: %w", err)
	}
	if err := attachMedia(ctx, tx, ticket.CustomerID, message.ID, record.FileIDs); err != nil {
		return domain.TicketDetail{}, false, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO ticket_status_history (id, ticket_id, from_status, to_status, actor_id, reason, created_at)
		VALUES ($1,$2,NULL,$3,$4,'Ticket created',$5)
	`, idgen.New(), ticket.ID, ticket.Status, ticket.CustomerID, ticket.CreatedAt); err != nil {
		return domain.TicketDetail{}, false, fmt.Errorf("insert ticket status history: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO ticket_sla_milestones (id, ticket_id, milestone_type, due_at, created_at, updated_at)
		VALUES ($1,$2,'FIRST_RESPONSE',$3,$5,$5), ($4,$2,'RESOLUTION',$6,$5,$5)
	`, idgen.New(), ticket.ID, ticket.FirstResponseDueAt, idgen.New(), ticket.CreatedAt, ticket.ResolutionDueAt); err != nil {
		return domain.TicketDetail{}, false, fmt.Errorf("insert ticket SLA: %w", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"ticket_id":   ticket.ID,
		"ticket_code": ticket.Code,
		"customer_id": ticket.CustomerID,
		"category":    ticket.Category,
		"priority":    ticket.Priority,
		"status":      ticket.Status,
		"created_at":  ticket.CreatedAt,
	})
	if err := insertOutbox(ctx, tx, ticket.ID, "SUPPORT_TICKET", "vn.omama.care.ticket.created.v1", r.topic, ticket.ID.String(), payload); err != nil {
		return domain.TicketDetail{}, false, err
	}
	if err := completeIdempotency(ctx, tx, ticket.CustomerID, "ticket.create", record.IdempotencyKey, ticket.ID.String(), 201); err != nil {
		return domain.TicketDetail{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.TicketDetail{}, false, fmt.Errorf("commit create ticket: %w", err)
	}

	message.FileIDs = append([]uuid.UUID(nil), record.FileIDs...)
	return domain.TicketDetail{Ticket: ticket, Messages: []domain.TicketMessage{message}}, true, nil
}

func (r *TicketRepository) ListOwned(ctx context.Context, customerID uuid.UUID, limit int, cursor string) (domain.TicketPage, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	args := []any{customerID, limit + 1}
	query := `SELECT ` + ticketColumns + ` FROM support_tickets WHERE customer_id = $1`
	if cursor != "" {
		createdAt, id, err := decodeCursor(cursor)
		if err != nil {
			return domain.TicketPage{}, fmt.Errorf("%w: invalid cursor", domain.ErrInvalidArgument)
		}
		query += ` AND (created_at, id) < ($3, $4)`
		args = append(args, createdAt, id)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT $2`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return domain.TicketPage{}, fmt.Errorf("list tickets: %w", err)
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

	next := ""
	if len(items) > limit {
		last := items[limit-1]
		next = encodeCursor(last.CreatedAt, last.ID)
		items = items[:limit]
	}
	return domain.TicketPage{Items: items, NextCursor: next}, nil
}

func (r *TicketRepository) GetOwned(ctx context.Context, id, customerID uuid.UUID) (domain.TicketDetail, error) {
	return getTicketDetail(ctx, r.pool, id, customerID)
}

func (r *TicketRepository) AddCustomerMessage(ctx context.Context, record domain.AddTicketMessageRecord) (domain.TicketMessage, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return domain.TicketMessage{}, false, err
	}
	defer tx.Rollback(ctx)

	resourceID, created, err := claimIdempotency(ctx, tx, record.CustomerID, "ticket.add-message:"+record.Message.TicketID.String(), record.IdempotencyKey, record.RequestHash, record.IdempotencyExpiry)
	if err != nil {
		return domain.TicketMessage{}, false, err
	}
	if !created {
		id, parseErr := uuid.Parse(resourceID)
		if parseErr != nil {
			return domain.TicketMessage{}, false, parseErr
		}
		message, getErr := getMessageOwned(ctx, tx, id, record.CustomerID)
		if getErr != nil {
			return domain.TicketMessage{}, false, getErr
		}
		if err := tx.Commit(ctx); err != nil {
			return domain.TicketMessage{}, false, err
		}
		return message, false, nil
	}

	var status domain.TicketStatus
	err = tx.QueryRow(ctx, `SELECT status FROM support_tickets WHERE id = $1 AND customer_id = $2 FOR UPDATE`, record.Message.TicketID, record.CustomerID).Scan(&status)
	if err == pgx.ErrNoRows {
		return domain.TicketMessage{}, false, domain.ErrNotFound
	}
	if err != nil {
		return domain.TicketMessage{}, false, err
	}
	if status == domain.TicketStatusClosed {
		return domain.TicketMessage{}, false, fmt.Errorf("%w: closed ticket does not accept messages", domain.ErrConflict)
	}

	message := record.Message
	_, err = tx.Exec(ctx, `
		INSERT INTO ticket_messages (id, ticket_id, sender_id, sender_type, direction, content, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, message.ID, message.TicketID, message.SenderID, message.SenderType, message.Direction, message.Content, message.CreatedAt)
	if err != nil {
		return domain.TicketMessage{}, false, fmt.Errorf("insert ticket message: %w", err)
	}
	if err := attachMedia(ctx, tx, record.CustomerID, message.ID, record.FileIDs); err != nil {
		return domain.TicketMessage{}, false, err
	}

	payload, _ := json.Marshal(map[string]any{
		"ticket_id":   message.TicketID,
		"message_id":  message.ID,
		"customer_id": record.CustomerID,
		"created_at":  message.CreatedAt,
	})
	if err := insertOutbox(ctx, tx, message.TicketID, "SUPPORT_TICKET", "vn.omama.care.ticket.message.created.v1", r.topic, message.TicketID.String(), payload); err != nil {
		return domain.TicketMessage{}, false, err
	}
	operation := "ticket.add-message:" + message.TicketID.String()
	if err := completeIdempotency(ctx, tx, record.CustomerID, operation, record.IdempotencyKey, message.ID.String(), 201); err != nil {
		return domain.TicketMessage{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.TicketMessage{}, false, err
	}
	message.FileIDs = append([]uuid.UUID(nil), record.FileIDs...)
	return message, true, nil
}

func attachMedia(ctx context.Context, tx pgx.Tx, ownerID, aggregateID uuid.UUID, fileIDs []uuid.UUID) error {
	seen := make(map[uuid.UUID]struct{}, len(fileIDs))
	for _, fileID := range fileIDs {
		if _, duplicate := seen[fileID]; duplicate {
			return fmt.Errorf("%w: duplicate file ID", domain.ErrInvalidArgument)
		}
		seen[fileID] = struct{}{}
		commandTag, err := tx.Exec(ctx, `
			UPDATE media_files
			SET status = 'ATTACHED', attached_at = COALESCE(attached_at, NOW()), updated_at = NOW(), version = version + 1
			WHERE id = $1 AND owner_id = $2 AND purpose = 'TICKET_ATTACHMENT'
			  AND status = 'READY' AND scan_status = 'CLEAN' AND expires_at > NOW()
		`, fileID, ownerID)
		if err != nil {
			return fmt.Errorf("attach media: %w", err)
		}
		if commandTag.RowsAffected() != 1 {
			return domain.ErrMediaNotReady
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO media_references (id, media_id, aggregate_type, aggregate_id, created_by)
			VALUES ($1,$2,'TICKET_MESSAGE',$3,$4)
		`, idgen.New(), fileID, aggregateID, ownerID); err != nil {
			return fmt.Errorf("insert media reference: %w", err)
		}
	}
	return nil
}

func insertOutbox(ctx context.Context, tx pgx.Tx, aggregateID uuid.UUID, aggregateType, eventType, topic, partitionKey string, payload []byte) error {
	traceparent := ""
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		traceparent = fmt.Sprintf(
			"00-%s-%s-%02x",
			spanContext.TraceID().String(),
			spanContext.SpanID().String(),
			byte(spanContext.TraceFlags()),
		)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO outbox_events (
			id, aggregate_type, aggregate_id, event_type, topic, partition_key, payload, traceparent
		) VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''))
	`, idgen.New(), aggregateType, aggregateID.String(), eventType, topic, partitionKey, payload, traceparent)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

const ticketColumns = `
	id, code, customer_id, subject, category, priority, status, COALESCE(order_id, ''), version,
	first_response_due_at, resolution_due_at, created_at, updated_at
`

func scanTicket(row rowScanner) (domain.SupportTicket, error) {
	var ticket domain.SupportTicket
	err := row.Scan(
		&ticket.ID, &ticket.Code, &ticket.CustomerID, &ticket.Subject, &ticket.Category,
		&ticket.Priority, &ticket.Status, &ticket.OrderID, &ticket.Version,
		&ticket.FirstResponseDueAt, &ticket.ResolutionDueAt, &ticket.CreatedAt, &ticket.UpdatedAt,
	)
	return ticket, err
}

type ticketQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func getTicketDetail(ctx context.Context, db ticketQuerier, id, customerID uuid.UUID) (domain.TicketDetail, error) {
	ticket, err := scanTicket(db.QueryRow(ctx, `SELECT `+ticketColumns+` FROM support_tickets WHERE id = $1 AND customer_id = $2`, id, customerID))
	if err == pgx.ErrNoRows {
		return domain.TicketDetail{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.TicketDetail{}, fmt.Errorf("get ticket: %w", err)
	}

	rows, err := db.Query(ctx, `
		SELECT m.id, m.ticket_id, m.sender_id, m.sender_type, m.direction, m.content, m.created_at,
		       COALESCE(array_agg(r.media_id) FILTER (WHERE r.media_id IS NOT NULL), '{}')
		FROM ticket_messages m
		LEFT JOIN media_references r ON r.aggregate_type = 'TICKET_MESSAGE' AND r.aggregate_id = m.id
		WHERE m.ticket_id = $1
		GROUP BY m.id
		ORDER BY m.created_at, m.id
	`, id)
	if err != nil {
		return domain.TicketDetail{}, fmt.Errorf("list ticket messages: %w", err)
	}
	defer rows.Close()

	messages := make([]domain.TicketMessage, 0)
	for rows.Next() {
		var message domain.TicketMessage
		if err := rows.Scan(&message.ID, &message.TicketID, &message.SenderID, &message.SenderType, &message.Direction, &message.Content, &message.CreatedAt, &message.FileIDs); err != nil {
			return domain.TicketDetail{}, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return domain.TicketDetail{}, err
	}
	return domain.TicketDetail{Ticket: ticket, Messages: messages}, nil
}

func getMessageOwned(ctx context.Context, db ticketQuerier, id, customerID uuid.UUID) (domain.TicketMessage, error) {
	var message domain.TicketMessage
	err := db.QueryRow(ctx, `
		SELECT m.id, m.ticket_id, m.sender_id, m.sender_type, m.direction, m.content, m.created_at,
		       COALESCE(array_agg(r.media_id) FILTER (WHERE r.media_id IS NOT NULL), '{}')
		FROM ticket_messages m
		JOIN support_tickets t ON t.id = m.ticket_id
		LEFT JOIN media_references r ON r.aggregate_type = 'TICKET_MESSAGE' AND r.aggregate_id = m.id
		WHERE m.id = $1 AND t.customer_id = $2
		GROUP BY m.id
	`, id, customerID).Scan(&message.ID, &message.TicketID, &message.SenderID, &message.SenderType, &message.Direction, &message.Content, &message.CreatedAt, &message.FileIDs)
	if err == pgx.ErrNoRows {
		return domain.TicketMessage{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.TicketMessage{}, err
	}
	return message, nil
}

func encodeCursor(createdAt time.Time, id uuid.UUID) string {
	raw := strconv.FormatInt(createdAt.UnixNano(), 10) + ":" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(cursor string) (time.Time, uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, domain.ErrInvalidArgument
	}
	nano, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	return time.Unix(0, nano).UTC(), id, nil
}
