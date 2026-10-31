package persistence

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"strings"
)

func (p *Payment) queueCursor(actor PaymentActor, kind, status, last string) string {
	raw, _ := json.Marshal([]string{actor.ID, p.Connection, p.Receiver, kind, status, last})
	body := base64.RawURLEncoding.EncodeToString(raw)
	h := hmac.New(sha256.New, p.guestKey())
	h.Write([]byte(body))
	return body + "." + base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func (p *Payment) queueAfter(actor PaymentActor, kind, status, cursor string) (string, error) {
	if cursor == "" {
		return "ffffffff-ffff-ffff-ffff-ffffffffffff", nil
	}
	parts := strings.Split(cursor, ".")
	if len(parts) != 2 || len(cursor) > 4096 {
		return "", ErrInput
	}
	signature, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		return "", ErrInput
	}
	h := hmac.New(sha256.New, p.guestKey())
	h.Write([]byte(parts[0]))
	if !hmac.Equal(signature, h.Sum(nil)) {
		return "", ErrInput
	}
	raw, e := base64.RawURLEncoding.DecodeString(parts[0])
	if e != nil {
		return "", ErrInput
	}
	var values []string
	if json.Unmarshal(raw, &values) != nil || len(values) != 6 || values[0] != actor.ID || values[1] != p.Connection || values[2] != p.Receiver || values[3] != kind || values[4] != status || !validID(values[5]) {
		return "", ErrInput
	}
	return values[5], nil
}

type RefundPage struct {
	Items []RefundView `json:"items"`
	Next  *string      `json:"next_cursor"`
}
type CasePage struct {
	Items []PaymentCase `json:"items"`
	Next  *string       `json:"next_cursor"`
}

func (p *Payment) RefundQueue(ctx context.Context, actor PaymentActor, status, cursor string, limit int) (out RefundPage, err error) {
	out.Items = []RefundView{}
	if !actor.allows(p, "payment.refund.read", false) {
		return out, ErrForbidden
	}
	if limit < 1 || limit > 100 || len(status) > 64 {
		return out, ErrInput
	}
	after, e := p.queueAfter(actor, "REFUND", status, cursor)
	if e != nil {
		return out, e
	}
	rows, e := p.Store.db.Query(ctx, `SELECT f.id::text FROM refunds f JOIN receipts r ON r.id=f.receipt_id WHERE f.execution_mode='MANUAL' AND r.connection_id=$1 AND r.receiver_profile_id=$2 AND ($3='' OR f.status=$3) AND f.id<$4::uuid ORDER BY f.id DESC LIMIT $5`, p.Connection, p.Receiver, status, after, limit+1)
	if e != nil {
		return out, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return out, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	more := len(ids) > limit
	if more {
		ids = ids[:limit]
	}
	for _, id := range ids {
		view, e := p.refundView(ctx, p.Store.db, id)
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, view)
	}
	if more {
		next := p.queueCursor(actor, "REFUND", status, ids[len(ids)-1])
		out.Next = &next
	}
	return out, nil
}
func (p *Payment) CaseQueue(ctx context.Context, actor PaymentActor, status, cursor string, limit int) (out CasePage, err error) {
	out.Items = []PaymentCase{}
	if !actor.allows(p, "payment.reconciliation.read", false) {
		return out, ErrForbidden
	}
	if limit < 1 || limit > 100 || len(status) > 64 {
		return out, ErrInput
	}
	after, e := p.queueAfter(actor, "CASE", status, cursor)
	if e != nil {
		return out, e
	}
	rows, e := p.Store.db.Query(ctx, `SELECT c.id::text FROM reconciliation_cases c LEFT JOIN receipts r ON r.id=c.receipt_id LEFT JOIN payment_provider_observations o ON o.id=c.observation_id WHERE ((r.connection_id=$1 AND r.receiver_profile_id=$2) OR (o.connection_id=$1 AND o.receiver_profile_id=$2)) AND ($3='' OR c.status=$3) AND c.id<$4::uuid AND ($5 OR c.assigned_to=$6::uuid) ORDER BY c.id DESC LIMIT $7`, p.Connection, p.Receiver, status, after, actor.Permissions["payment.reconciliation.manage"], actor.ID, limit+1)
	if e != nil {
		return out, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return out, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	more := len(ids) > limit
	if more {
		ids = ids[:limit]
	}
	for _, id := range ids {
		view, e := p.caseView(ctx, p.Store.db, id)
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, view)
	}
	if more {
		next := p.queueCursor(actor, "CASE", status, ids[len(ids)-1])
		out.Next = &next
	}
	return out, nil
}
func (p *Payment) SweepManualRefunds(ctx context.Context) error {
	return p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		rows, e := tx.Query(ctx, `UPDATE refunds f SET status='UNKNOWN',execution_state='UNKNOWN',version=version+1 FROM receipts r WHERE r.id=f.receipt_id AND r.connection_id=$1 AND r.receiver_profile_id=$2 AND f.execution_mode='MANUAL' AND f.execution_state IN ('STARTED','EVIDENCE_PENDING') AND EXISTS(SELECT 1 FROM refund_execution_attempts a WHERE a.refund_id=f.id AND a.attempt_no=f.current_attempt AND a.started_at<clock_timestamp()-interval '15 minutes') RETURNING f.id::text`, p.Connection, p.Receiver)
		if e != nil {
			return e
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return e
			}
			ids = append(ids, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, id := range ids {
			if e = p.Store.audit(ctx, tx, "", "PAYMENT", "REFUND_OUTCOME_UNKNOWN", map[string]any{"refund_id": id}); e != nil {
				return e
			}
		}
		return nil
	})
}
