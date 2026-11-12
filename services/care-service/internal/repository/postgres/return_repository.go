package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/care-service/internal/domain"
	"github.com/omamx/care-service/internal/idgen"
	"github.com/omamx/care-service/internal/requestcontext"
	careReturns "github.com/omamx/care-service/internal/returns"
	"go.opentelemetry.io/otel/trace"
)

type ReturnRepository struct{ pool *pgxpool.Pool }

func NewReturnRepository(pool *pgxpool.Pool) *ReturnRepository { return &ReturnRepository{pool: pool} }

func (r *ReturnRepository) ReplayCreate(ctx context.Context, actorID uuid.UUID, key, requestHash string) (careReturns.Case, bool, error) {
	var storedHash, resourceID string
	err := r.pool.QueryRow(ctx, `SELECT request_hash,COALESCE(resource_id,'') FROM idempotency_records WHERE actor_id=$1 AND operation='return.create' AND idempotency_key=$2 AND expires_at>NOW()`, actorID, key).Scan(&storedHash, &resourceID)
	if err == pgx.ErrNoRows {
		return careReturns.Case{}, false, nil
	}
	if err != nil {
		return careReturns.Case{}, false, err
	}
	if storedHash != requestHash {
		return careReturns.Case{}, false, domain.ErrIdempotencyConflict
	}
	caseID, err := uuid.Parse(resourceID)
	if err != nil {
		return careReturns.Case{}, false, domain.ErrConflict
	}
	value, err := getReturnCase(ctx, r.pool, caseID, &actorID)
	return value, err == nil, err
}

func (r *ReturnRepository) Create(ctx context.Context, record careReturns.CreateRecord) (careReturns.Case, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return careReturns.Case{}, false, err
	}
	defer tx.Rollback(ctx)
	resourceID, created, err := claimIdempotency(ctx, tx, record.Case.CustomerID, "return.create", record.IdempotencyKey, record.RequestHash, record.IdempotencyExpiry)
	if err != nil {
		return careReturns.Case{}, false, err
	}
	if !created {
		id, parseErr := uuid.Parse(resourceID)
		if parseErr != nil {
			return careReturns.Case{}, false, parseErr
		}
		value, getErr := getReturnCase(ctx, tx, id, &record.Case.CustomerID)
		if getErr != nil {
			return careReturns.Case{}, false, getErr
		}
		if err := tx.Commit(ctx); err != nil {
			return careReturns.Case{}, false, err
		}
		return value, false, nil
	}

	lockIDs := make([]string, 0, len(record.Case.Lines))
	for _, line := range record.Case.Lines {
		lockIDs = append(lockIDs, line.OrderLineID)
	}
	sort.Strings(lockIDs)
	caseLinesByOrderLine := make(map[string]careReturns.CaseLine, len(record.Case.Lines))
	for _, line := range record.Case.Lines {
		caseLinesByOrderLine[line.OrderLineID] = line
	}
	for _, orderLineID := range lockIDs {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, orderLineID); err != nil {
			return careReturns.Case{}, false, fmt.Errorf("lock return quantity: %w", err)
		}
		var activeQuantity int
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(l.requested_quantity), 0)
			FROM return_case_lines l JOIN return_cases c ON c.id = l.return_case_id
			WHERE l.order_line_id = $1 AND c.order_id = $2 AND c.status IN ('REQUESTED','NEEDS_INFORMATION','REVIEWING','APPROVED','PARTIALLY_APPROVED','IN_FULFILLMENT')
		`, orderLineID, record.Case.OrderID).Scan(&activeQuantity); err != nil {
			return careReturns.Case{}, false, fmt.Errorf("check active return quantity: %w", err)
		}
		var source careReturns.OrderLine
		for _, candidate := range record.OrderContext.Lines {
			if candidate.OrderLineID == orderLineID {
				source = candidate
				break
			}
		}
		caseLine := caseLinesByOrderLine[orderLineID]
		if caseLine.OrderLineID == "" || source.OrderLineID == "" || source.PreviouslyCommittedReturnQuantity+activeQuantity+caseLine.RequestedQuantity > source.PurchasedQuantity {
			return careReturns.Case{}, false, domain.ErrConflict
		}
	}

	orderSnapshot, err := json.Marshal(record.OrderContext)
	if err != nil {
		return careReturns.Case{}, false, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO return_cases (
			id, code, customer_id, order_id, requested_resolution, reason_code, customer_note,
			status, order_snapshot, source_contract_version, source_captured_at,
			return_eligible_at, eligibility_expires_at, version, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,'REQUESTED',$8::jsonb,$9,$10,$11,$12,1,$13,$13)
	`, record.Case.ID, record.Case.Code, record.Case.CustomerID, record.Case.OrderID,
		record.Case.Resolution, record.Case.ReasonCode, record.Case.CustomerNote, string(orderSnapshot),
		record.OrderContext.ContractVersion, record.OrderContext.CapturedAt,
		record.OrderContext.ReturnEligibleAt, record.OrderContext.EligibilityExpiresAt, record.Case.CreatedAt); err != nil {
		return careReturns.Case{}, false, fmt.Errorf("insert return case: %w", err)
	}
	for _, line := range record.Case.Lines {
		var source careReturns.OrderLine
		for _, candidate := range record.OrderContext.Lines {
			if candidate.OrderLineID == line.OrderLineID {
				source = candidate
				break
			}
		}
		snapshot, _ := json.Marshal(source)
		if _, err := tx.Exec(ctx, `
			INSERT INTO return_case_lines (
				id, return_case_id, order_line_id, product_id, sku, product_name,
				purchased_quantity, previously_committed_return_quantity, requested_quantity,
				refundable_ceiling_minor, refundable_unit_ceiling_minor, currency, source_snapshot, source_captured_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb,$14)
		`, line.ID, record.Case.ID, line.OrderLineID, line.ProductID, line.SKU, line.ProductName,
			line.PurchasedQuantity, source.PreviouslyCommittedReturnQuantity, line.RequestedQuantity,
			line.RefundableCeiling, line.RefundableUnitCeiling, line.Currency, string(snapshot), record.OrderContext.CapturedAt); err != nil {
			return careReturns.Case{}, false, fmt.Errorf("insert return line: %w", err)
		}
	}
	for _, fileID := range record.Case.EvidenceFileIDs {
		command, err := tx.Exec(ctx, `UPDATE media_files SET status='ATTACHED', attached_at=COALESCE(attached_at,$3), updated_at=$3, version=version+1
			WHERE id=$1 AND owner_id=$2 AND purpose='RETURN_EVIDENCE' AND status='READY' AND scan_status='CLEAN' AND expires_at>$3`, fileID, record.Case.CustomerID, record.Case.CreatedAt)
		if err != nil {
			return careReturns.Case{}, false, fmt.Errorf("attach return evidence: %w", err)
		}
		if command.RowsAffected() != 1 {
			return careReturns.Case{}, false, domain.ErrMediaNotReady
		}
		if _, err := tx.Exec(ctx, `INSERT INTO return_evidence (id, return_case_id, media_id, submitted_by, created_at) VALUES ($1,$2,$3,$4,$5)`, idgen.New(), record.Case.ID, fileID, record.Case.CustomerID, record.Case.CreatedAt); err != nil {
			return careReturns.Case{}, false, fmt.Errorf("insert return evidence: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO media_references (id, media_id, aggregate_type, aggregate_id, created_by) VALUES ($1,$2,'RETURN_CASE',$3,$4)`, idgen.New(), fileID, record.Case.ID, record.Case.CustomerID); err != nil {
			return careReturns.Case{}, false, fmt.Errorf("insert return evidence media reference: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO return_status_history (id,return_case_id,from_status,to_status,actor_id,reason,created_at) VALUES ($1,$2,NULL,'REQUESTED',$3,$4,$5)`, idgen.New(), record.Case.ID, record.Case.CustomerID, "Customer submitted return request", record.Case.CreatedAt); err != nil {
		return careReturns.Case{}, false, fmt.Errorf("insert return history: %w", err)
	}
	if err := writeReturnAudit(ctx, tx, record.Case.CustomerID, "RETURN_CASE_CREATED", record.Case.ID, "Customer submitted return request", nil, map[string]any{"status": "REQUESTED", "version": 1, "order_id": record.Case.OrderID}, record.RequestID); err != nil {
		return careReturns.Case{}, false, err
	}
	if err := completeIdempotency(ctx, tx, record.Case.CustomerID, "return.create", record.IdempotencyKey, record.Case.ID.String(), 201); err != nil {
		return careReturns.Case{}, false, err
	}
	value, err := getReturnCase(ctx, tx, record.Case.ID, &record.Case.CustomerID)
	if err != nil {
		return careReturns.Case{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return careReturns.Case{}, false, fmt.Errorf("commit return case: %w", err)
	}
	return value, true, nil
}

func (r *ReturnRepository) GetOwned(ctx context.Context, customerID, caseID uuid.UUID) (careReturns.Case, error) {
	return getReturnCase(ctx, r.pool, caseID, &customerID)
}

func (r *ReturnRepository) Get(ctx context.Context, caseID uuid.UUID) (careReturns.Case, error) {
	return getReturnCase(ctx, r.pool, caseID, nil)
}

func (r *ReturnRepository) ListOwned(ctx context.Context, customerID uuid.UUID, status domain.ReturnCaseStatus, page, pageSize int) ([]careReturns.Case, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM return_cases WHERE customer_id=$1 AND ($2::varchar='' OR status=$2)`, customerID, string(status)).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id FROM return_cases WHERE customer_id=$1 AND ($2::varchar='' OR status=$2) ORDER BY created_at DESC,id DESC LIMIT $3 OFFSET $4`, customerID, string(status), pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uuid.UUID, 0, pageSize)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, 0, err
	}
	rows.Close()
	items := make([]careReturns.Case, 0, len(ids))
	for _, id := range ids {
		value, err := getReturnCase(ctx, r.pool, id, &customerID)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, value)
	}
	return items, total, nil
}

func (r *ReturnRepository) List(ctx context.Context, status domain.ReturnCaseStatus, page, pageSize int) ([]careReturns.Case, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM return_cases WHERE ($1::varchar = '' OR status=$1)`, string(status)).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id FROM return_cases WHERE ($1::varchar = '' OR status=$1) ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`, string(status), pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uuid.UUID, 0, pageSize)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, 0, err
	}
	rows.Close()
	items := make([]careReturns.Case, 0, len(ids))
	for _, id := range ids {
		value, err := getReturnCase(ctx, r.pool, id, nil)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, value)
	}
	return items, total, nil
}

func (r *ReturnRepository) StatusCounts(ctx context.Context) (map[domain.ReturnCaseStatus]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT status,count(*) FROM return_cases GROUP BY status ORDER BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[domain.ReturnCaseStatus]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[domain.ReturnCaseStatus(status)] = count
	}
	return counts, rows.Err()
}

func (r *ReturnRepository) Action(ctx context.Context, record careReturns.ActionRecord) (careReturns.Case, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return careReturns.Case{}, false, err
	}
	defer tx.Rollback(ctx)
	operation := "return.action:" + record.CaseID.String()
	resourceID, applied, err := claimIdempotency(ctx, tx, record.ActorID, operation, record.IdempotencyKey, record.RequestHash, record.IdempotencyExpiry)
	if err != nil {
		return careReturns.Case{}, false, err
	}
	if !applied {
		id, parseErr := uuid.Parse(resourceID)
		if parseErr != nil {
			return careReturns.Case{}, false, parseErr
		}
		value, getErr := getReturnCase(ctx, tx, id, nil)
		if getErr != nil {
			return careReturns.Case{}, false, getErr
		}
		if err := tx.Commit(ctx); err != nil {
			return careReturns.Case{}, false, err
		}
		return value, false, nil
	}
	var current careReturns.Case
	var snapshot []byte
	var currentStatus string
	err = tx.QueryRow(ctx, `SELECT id,code,customer_id,order_id,status,requested_resolution,reason_code,customer_note,order_snapshot,version,created_at,updated_at FROM return_cases WHERE id=$1 FOR UPDATE`, record.CaseID).Scan(
		&current.ID, &current.Code, &current.CustomerID, &current.OrderID, &currentStatus, &current.Resolution, &current.ReasonCode, &current.CustomerNote, &snapshot, &current.Version, &current.CreatedAt, &current.UpdatedAt)
	if err == pgx.ErrNoRows {
		return careReturns.Case{}, false, domain.ErrNotFound
	}
	if err != nil {
		return careReturns.Case{}, false, fmt.Errorf("lock return case: %w", err)
	}
	current.Status = domain.ReturnCaseStatus(currentStatus)
	if current.Version != record.ExpectedVersion {
		return careReturns.Case{}, false, domain.ErrConflict
	}
	if err := json.Unmarshal(snapshot, &current.OrderContext); err != nil {
		return careReturns.Case{}, false, fmt.Errorf("decode stored Order snapshot: %w", err)
	}
	current.Lines, err = getReturnLines(ctx, tx, current.ID)
	if err != nil {
		return careReturns.Case{}, false, err
	}
	from := current.Status
	to := from
	switch record.Action {
	case "START_REVIEW":
		if !from.CanTransitionTo(domain.ReturnCaseReviewing) {
			return careReturns.Case{}, false, domain.ErrConflict
		}
		to = domain.ReturnCaseReviewing
	case "REQUEST_INFORMATION":
		if !from.CanTransitionTo(domain.ReturnCaseNeedsInformation) {
			return careReturns.Case{}, false, domain.ErrConflict
		}
		to = domain.ReturnCaseNeedsInformation
	case "DECIDE":
		if current.Status != domain.ReturnCaseReviewing {
			return careReturns.Case{}, false, domain.ErrConflict
		}
		if record.Decision == nil || (record.Decision.Outcome != domain.ReturnCaseApproved && record.Decision.Outcome != domain.ReturnCasePartiallyApproved && record.Decision.Outcome != domain.ReturnCaseRejected) {
			return careReturns.Case{}, false, domain.ErrInvalidArgument
		}
		if !from.CanTransitionTo(record.Decision.Outcome) {
			return careReturns.Case{}, false, domain.ErrConflict
		}
		to = record.Decision.Outcome
	default:
		return careReturns.Case{}, false, domain.ErrInvalidArgument
	}
	updated, err := scanReturnCase(tx.QueryRow(ctx, `UPDATE return_cases SET status=$2,version=version+1,updated_at=$3 WHERE id=$1 AND version=$4 RETURNING id,code,customer_id,order_id,status,requested_resolution,reason_code,customer_note,version,created_at,updated_at`, record.CaseID, string(to), record.At, record.ExpectedVersion))
	if err != nil {
		return careReturns.Case{}, false, fmt.Errorf("update return case: %w", err)
	}
	updated.OrderContext = current.OrderContext
	if _, err := tx.Exec(ctx, `INSERT INTO return_status_history (id,return_case_id,from_status,to_status,actor_id,reason,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, idgen.New(), record.CaseID, string(from), string(to), record.ActorID, record.Note, record.At); err != nil {
		return careReturns.Case{}, false, fmt.Errorf("insert return action history: %w", err)
	}
	if record.Decision != nil {
		decision := record.Decision
		if _, err := tx.Exec(ctx, `INSERT INTO return_decisions (id,return_case_id,decision_version,outcome,decided_by,reason_code,reason,return_required,approved_total_minor,authoritative_ceiling_minor,currency,source_snapshot,source_captured_at,decided_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13,$14)`, decision.ID, record.CaseID, decision.Version, string(decision.Outcome), decision.ActorID, decision.ReasonCode, decision.Reason, decision.ReturnRequired, decision.ApprovedTotalMinor, decision.AuthoritativeCeiling, decision.Currency, string(snapshot), current.OrderContext.CapturedAt, decision.DecidedAt); err != nil {
			return careReturns.Case{}, false, fmt.Errorf("insert immutable return decision: %w", err)
		}
		for _, line := range decision.Lines {
			outcome := "REJECTED"
			if line.ApprovedQuantity > 0 {
				outcome = "APPROVED"
			}
			if _, err := tx.Exec(ctx, `INSERT INTO return_decision_lines (id,return_case_id,return_decision_id,return_case_line_id,outcome,approved_quantity,approved_amount_minor,reason_code) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, idgen.New(), record.CaseID, decision.ID, line.ReturnCaseLineID, outcome, line.ApprovedQuantity, line.ApprovedAmountMinor, line.ReasonCode); err != nil {
				return careReturns.Case{}, false, fmt.Errorf("insert return decision line: %w", err)
			}
		}
		goodsStatus := "NOT_REQUIRED"
		if decision.ReturnRequired {
			goodsStatus = "AWAITING_CUSTOMER_SHIPMENT"
		}
		if _, err := tx.Exec(ctx, `INSERT INTO returned_goods_tracking (id,return_case_id,return_decision_id,status,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$5)`, idgen.New(), record.CaseID, decision.ID, goodsStatus, record.At); err != nil {
			return careReturns.Case{}, false, fmt.Errorf("create returned-goods projection: %w", err)
		}
		updated.Decision = decision
	}
	if err := writeReturnAudit(ctx, tx, record.ActorID, "RETURN_CASE_"+record.Action, record.CaseID, record.Note, map[string]any{"status": from, "version": record.ExpectedVersion}, map[string]any{"status": to, "version": updated.Version}, record.RequestID); err != nil {
		return careReturns.Case{}, false, err
	}
	if err := completeIdempotency(ctx, tx, record.ActorID, operation, record.IdempotencyKey, record.CaseID.String(), 200); err != nil {
		return careReturns.Case{}, false, err
	}
	value, err := getReturnCase(ctx, tx, record.CaseID, nil)
	if err != nil {
		return careReturns.Case{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return careReturns.Case{}, false, err
	}
	return value, true, nil
}

func (r *ReturnRepository) AttachReverseShipment(ctx context.Context, caseID, decisionID uuid.UUID, shipmentID string, at time.Time) (careReturns.Case, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return careReturns.Case{}, err
	}
	defer tx.Rollback(ctx)
	var current string
	var version int
	err = tx.QueryRow(ctx, `SELECT status,version FROM return_cases WHERE id=$1 FOR UPDATE`, caseID).Scan(&current, &version)
	if err == pgx.ErrNoRows {
		return careReturns.Case{}, domain.ErrNotFound
	}
	if err != nil {
		return careReturns.Case{}, err
	}
	var existing string
	var actorID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT COALESCE(t.reverse_shipment_id,''),d.decided_by FROM returned_goods_tracking t JOIN return_decisions d ON d.id=t.return_decision_id WHERE t.return_case_id=$1 AND t.return_decision_id=$2 FOR UPDATE OF t`, caseID, decisionID).Scan(&existing, &actorID)
	if err == pgx.ErrNoRows {
		return careReturns.Case{}, domain.ErrNotFound
	}
	if err != nil {
		return careReturns.Case{}, err
	}
	if existing != "" {
		if existing != shipmentID {
			return careReturns.Case{}, domain.ErrConflict
		}
		value, e := getReturnCase(ctx, tx, caseID, nil)
		if e != nil {
			return careReturns.Case{}, e
		}
		if e = tx.Commit(ctx); e != nil {
			return careReturns.Case{}, e
		}
		return value, nil
	}
	if domain.ReturnCaseStatus(current) != domain.ReturnCaseApproved && domain.ReturnCaseStatus(current) != domain.ReturnCasePartiallyApproved {
		return careReturns.Case{}, domain.ErrConflict
	}
	if _, err := tx.Exec(ctx, `UPDATE returned_goods_tracking SET reverse_shipment_id=$3,status='AWAITING_CUSTOMER_SHIPMENT',updated_at=$4,version=version+1 WHERE return_case_id=$1 AND return_decision_id=$2`, caseID, decisionID, shipmentID, at); err != nil {
		return careReturns.Case{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE return_cases SET status='IN_FULFILLMENT',version=version+1,updated_at=$2 WHERE id=$1 AND version=$3`, caseID, at, version); err != nil {
		return careReturns.Case{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO return_status_history (id,return_case_id,from_status,to_status,actor_id,reason,created_at) VALUES ($1,$2,$3,'IN_FULFILLMENT',$4,$5,$6)`, idgen.New(), caseID, current, actorID, "Reverse shipment created", at); err != nil {
		return careReturns.Case{}, err
	}
	if err := writeReturnAudit(ctx, tx, uuid.Nil, "RETURN_REVERSE_SHIPMENT_CREATED", caseID, "Reverse shipment created", map[string]any{"status": current}, map[string]any{"status": "IN_FULFILLMENT", "reverse_shipment_id": shipmentID}, ""); err != nil {
		return careReturns.Case{}, err
	}
	value, err := getReturnCase(ctx, tx, caseID, nil)
	if err != nil {
		return careReturns.Case{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return careReturns.Case{}, err
	}
	return value, nil
}

type returnQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func getReturnCase(ctx context.Context, db returnQuerier, caseID uuid.UUID, owner *uuid.UUID) (careReturns.Case, error) {
	query := `SELECT id,code,customer_id,order_id,status,requested_resolution,reason_code,customer_note,order_snapshot,version,created_at,updated_at FROM return_cases WHERE id=$1`
	args := []any{caseID}
	if owner != nil {
		query += ` AND customer_id=$2`
		args = append(args, *owner)
	}
	var value careReturns.Case
	var status string
	var snapshot []byte
	err := db.QueryRow(ctx, query, args...).Scan(&value.ID, &value.Code, &value.CustomerID, &value.OrderID, &status, &value.Resolution, &value.ReasonCode, &value.CustomerNote, &snapshot, &value.Version, &value.CreatedAt, &value.UpdatedAt)
	if err == pgx.ErrNoRows {
		return careReturns.Case{}, domain.ErrNotFound
	}
	if err != nil {
		return careReturns.Case{}, err
	}
	value.Status = domain.ReturnCaseStatus(status)
	if err := json.Unmarshal(snapshot, &value.OrderContext); err != nil {
		return careReturns.Case{}, err
	}
	value.OrderNumber = value.OrderContext.OrderNumber
	value.Lines, err = getReturnLines(ctx, db, caseID)
	if err != nil {
		return careReturns.Case{}, err
	}
	rows, err := db.Query(ctx, `SELECT media_id FROM return_evidence WHERE return_case_id=$1 ORDER BY created_at,id`, caseID)
	if err != nil {
		return careReturns.Case{}, err
	}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return careReturns.Case{}, err
		}
		value.EvidenceFileIDs = append(value.EvidenceFileIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return careReturns.Case{}, err
	}
	rows.Close()
	var shipmentID string
	_ = db.QueryRow(ctx, `SELECT COALESCE(reverse_shipment_id,'') FROM returned_goods_tracking WHERE return_case_id=$1 ORDER BY created_at DESC LIMIT 1`, caseID).Scan(&shipmentID)
	value.ReverseShipmentID = shipmentID
	decision, err := getLatestDecision(ctx, db, caseID)
	if err != nil {
		return careReturns.Case{}, err
	}
	value.Decision = decision
	return value, nil
}

func getReturnLines(ctx context.Context, db returnQuerier, caseID uuid.UUID) ([]careReturns.CaseLine, error) {
	rows, err := db.Query(ctx, `SELECT id,order_line_id,product_id,sku,product_name,purchased_quantity,requested_quantity,refundable_ceiling_minor,refundable_unit_ceiling_minor,currency FROM return_case_lines WHERE return_case_id=$1 ORDER BY created_at,id`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lines := []careReturns.CaseLine{}
	for rows.Next() {
		var l careReturns.CaseLine
		if err := rows.Scan(&l.ID, &l.OrderLineID, &l.ProductID, &l.SKU, &l.ProductName, &l.PurchasedQuantity, &l.RequestedQuantity, &l.RefundableCeiling, &l.RefundableUnitCeiling, &l.Currency); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

func getLatestDecision(ctx context.Context, db returnQuerier, caseID uuid.UUID) (*careReturns.Decision, error) {
	var d careReturns.Decision
	var outcome string
	err := db.QueryRow(ctx, `SELECT id,decision_version,outcome,decided_by,reason_code,reason,return_required,approved_total_minor,authoritative_ceiling_minor,currency,decided_at FROM return_decisions WHERE return_case_id=$1 ORDER BY decision_version DESC LIMIT 1`, caseID).Scan(&d.ID, &d.Version, &outcome, &d.ActorID, &d.ReasonCode, &d.Reason, &d.ReturnRequired, &d.ApprovedTotalMinor, &d.AuthoritativeCeiling, &d.Currency, &d.DecidedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d.Outcome = domain.ReturnCaseStatus(outcome)
	rows, err := db.Query(ctx, `SELECT return_case_line_id,approved_quantity,approved_amount_minor,reason_code FROM return_decision_lines WHERE return_decision_id=$1 ORDER BY id`, d.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var line careReturns.DecisionLineInput
		if err := rows.Scan(&line.ReturnCaseLineID, &line.ApprovedQuantity, &line.ApprovedAmountMinor, &line.ReasonCode); err != nil {
			return nil, err
		}
		d.Lines = append(d.Lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &d, nil
}

func scanReturnCase(row pgx.Row) (careReturns.Case, error) {
	var c careReturns.Case
	var status string
	err := row.Scan(&c.ID, &c.Code, &c.CustomerID, &c.OrderID, &status, &c.Resolution, &c.ReasonCode, &c.CustomerNote, &c.Version, &c.CreatedAt, &c.UpdatedAt)
	c.Status = domain.ReturnCaseStatus(status)
	return c, err
}

func writeReturnAudit(ctx context.Context, tx pgx.Tx, actorID uuid.UUID, action string, caseID uuid.UUID, reason string, before, after any, requestID string) error {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	traceID := ""
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		traceID = sc.TraceID().String()
	}
	_, err := tx.Exec(ctx, `INSERT INTO audit_logs (id,actor_id,action,resource_type,resource_id,reason,before_state,after_state,request_id,trace_id,correlation_id) VALUES ($1,NULLIF($2,'00000000-0000-0000-0000-000000000000'::uuid),$3,'RETURN_CASE',$4,$5,$6::jsonb,$7::jsonb,NULLIF($8,''),NULLIF($9,''),NULLIF($10,''))`, idgen.New(), actorID, action, caseID.String(), reason, string(beforeJSON), string(afterJSON), requestID, traceID, requestcontext.CorrelationID(ctx))
	if err != nil {
		return fmt.Errorf("write return audit: %w", err)
	}
	return nil
}

var _ careReturns.Repository = (*ReturnRepository)(nil)
