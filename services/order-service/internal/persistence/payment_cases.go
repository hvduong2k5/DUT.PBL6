package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
)

type PaymentCase struct {
	ID          string   `json:"case_id"`
	Order       *string  `json:"order_id"`
	Receipt     *string  `json:"receipt_id"`
	Observation *string  `json:"observation_id"`
	Support     string   `json:"support_reference"`
	Reason      string   `json:"reason"`
	Status      string   `json:"status"`
	Assignee    *string  `json:"assigned_to"`
	Version     int64    `json:"version"`
	Evidence    []string `json:"evidence_ids"`
	Refunds     []string `json:"refund_ids"`
}
type CaseDecision struct {
	Reference  string   `json:"decision_reference"`
	Case       string   `json:"case_id"`
	Connection string   `json:"connection_id"`
	Receiver   string   `json:"receiver_profile_id"`
	Resolution string   `json:"resolution"`
	Allowed    bool     `json:"allowed"`
	Evidence   []string `json:"evidence_ids"`
	Policy     string   `json:"policy_version"`
}
type CaseAuthority interface {
	CaseDecision(context.Context, string) (CaseDecision, error)
	CanAssign(context.Context, string, string) (bool, error)
}

func (p *Payment) caseView(ctx context.Context, q rowReader, id string) (v PaymentCase, e error) {
	e = q.QueryRow(ctx, `SELECT c.id,r.order_id,c.receipt_id,c.observation_id,c.support_reference,c.reason,c.status,c.assigned_to::text,c.version,ARRAY(SELECT e.id::text FROM payment_evidence e JOIN refunds f ON f.id=e.refund_id WHERE f.receipt_id=c.receipt_id ORDER BY e.created_at,e.id),ARRAY(SELECT f.id::text FROM refunds f WHERE f.receipt_id=c.receipt_id ORDER BY f.created_at,f.id) FROM reconciliation_cases c LEFT JOIN receipts r ON r.id=c.receipt_id LEFT JOIN payment_provider_observations o ON o.id=c.observation_id WHERE c.id=$1 AND (r.connection_id=$2 AND r.receiver_profile_id=$3 OR o.connection_id=$2 AND o.receiver_profile_id=$3)`, id, p.Connection, p.Receiver).Scan(&v.ID, &v.Order, &v.Receipt, &v.Observation, &v.Support, &v.Reason, &v.Status, &v.Assignee, &v.Version, &v.Evidence, &v.Refunds)
	return
}
func (p *Payment) ReadCase(ctx context.Context, actor PaymentActor, id string) (PaymentCase, error) {
	if !actor.allows(p, "payment.reconciliation.read", false) {
		return PaymentCase{}, ErrForbidden
	}
	v, e := p.caseView(ctx, p.Store.db, id)
	if e != nil {
		return v, e
	}
	if !actor.Permissions["payment.reconciliation.manage"] && (v.Assignee == nil || *v.Assignee != actor.ID) {
		return PaymentCase{}, ErrForbidden
	}
	return v, nil
}
func (p *Payment) CaseCommand(ctx context.Context, actor PaymentActor, authority CaseAuthority, id, action, key string, version int64, reason, assignee, decisionRef, resolution string, evidence []string) (out PaymentCase, err error) {
	if !actor.allows(p, "payment.reconciliation.manage", false) {
		return out, ErrForbidden
	}
	if !commandKey(key) || reason == "" || len(reason) > 1000 {
		return out, ErrInput
	}
	if authority == nil {
		return out, ErrCapability
	}
	var decision CaseDecision
	if action == "assign" {
		if !validID(assignee) {
			return out, ErrInput
		}
		allowed, e := authority.CanAssign(ctx, p.Receiver, assignee)
		if e != nil {
			return out, e
		}
		if !allowed {
			return out, ErrForbidden
		}
	} else if action == "resolve" {
		var e error
		decision, e = authority.CaseDecision(ctx, decisionRef)
		if e != nil {
			return out, e
		}
		if !decision.Allowed || decision.Reference != decisionRef || decision.Case != id || decision.Connection != p.Connection || decision.Receiver != p.Receiver || decision.Resolution != resolution || decision.Policy == "" || len(evidence) == 0 {
			return out, ErrInput
		}
		a, _ := json.Marshal(decision.Evidence)
		b, _ := json.Marshal(evidence)
		if hash(a) != hash(b) {
			return out, ErrInput
		}
	} else {
		return out, ErrInput
	}
	request, _ := json.Marshal([]any{version, reason, assignee, decisionRef, resolution, evidence})
	requestHash := hash(request)
	kind := "PAYMENT_CASE:" + id + ":" + action
	err = p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		initial, e := p.caseView(ctx, tx, id)
		if e != nil {
			return e
		}
		if initial.Order != nil {
			var root string
			if e = tx.QueryRow(ctx, "SELECT id FROM orders WHERE id=$1 AND cohort=$2 FOR UPDATE", *initial.Order, Cohort).Scan(&root); e != nil {
				return e
			}
		}
		if initial.Receipt != nil {
			var receipt string
			if e = tx.QueryRow(ctx, "SELECT id FROM receipts WHERE id=$1 FOR UPDATE", *initial.Receipt).Scan(&receipt); e != nil {
				return e
			}
		}
		var locked string
		if e = tx.QueryRow(ctx, "SELECT id FROM reconciliation_cases WHERE id=$1 FOR UPDATE", id).Scan(&locked); e != nil {
			return e
		}
		var previous string
		var prior []byte
		e = tx.QueryRow(ctx, "SELECT request_hash,response FROM command_registry WHERE scope=$1 AND kind=$2 AND key=$3", "staff:"+actor.ID, kind, key).Scan(&previous, &prior)
		if e == nil {
			if previous != requestHash {
				return ErrConflict
			}
			if json.Unmarshal(prior, &out) != nil {
				return ErrIntegrity
			}
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		current, e := p.caseView(ctx, tx, id)
		if e != nil {
			return e
		}
		if current.Version != version || current.Status == "RESOLVED" {
			return ErrConflict
		}
		if action == "assign" {
			_, e = tx.Exec(ctx, "UPDATE reconciliation_cases SET assigned_to=$2,status='ASSIGNED',version=version+1 WHERE id=$1", id, assignee)
		} else {
			if current.Assignee == nil || *current.Assignee != actor.ID {
				return ErrForbidden
			}
			switch resolution {
			case "REFUNDED":
				if current.Receipt == nil {
					return ErrConflict
				}
				var amount, refunded int64
				if e = tx.QueryRow(ctx, `SELECT r.amount,COALESCE((SELECT sum(f.amount) FROM refunds f WHERE f.receipt_id=r.id AND f.status='SUCCEEDED'),0) FROM receipts r WHERE r.id=$1`, *current.Receipt).Scan(&amount, &refunded); e != nil {
					return e
				}
				if amount != refunded {
					return ErrConflict
				}
			case "NO_PAYMENT_CONFIRMED":
				if current.Receipt != nil {
					return ErrConflict
				}
				var linked *string
				if e = tx.QueryRow(ctx, "SELECT linked_receipt FROM payment_provider_observations WHERE id=$1", *current.Observation).Scan(&linked); e != nil {
					return e
				}
				if linked != nil {
					return ErrConflict
				}
			case "EXTERNAL_FUNDS_RESOLVED":
			default:
				return ErrInput
			}
			raw, _ := json.Marshal(decision)
			cipher, e := p.Store.cipher.Seal("case-resolution:"+id, raw)
			if e != nil {
				return e
			}
			_, e = tx.Exec(ctx, "UPDATE reconciliation_cases SET status='RESOLVED',resolution=$2,decision_reference=$3,resolution_cipher=$4,version=version+1 WHERE id=$1", id, resolution, decisionRef, cipher)
		}
		if e != nil {
			return e
		}
		// Resolving a money case is not permission to remove unrelated operational holds.
		if current.Order != nil {
			_, e = tx.Exec(ctx, `UPDATE payment_order_projections SET has_open_reconciliation=EXISTS(SELECT 1 FROM reconciliation_cases c JOIN receipts r ON r.id=c.receipt_id WHERE r.order_id=$1 AND c.status<>'RESOLVED'),version=version+1 WHERE order_id=$1`, *current.Order)
			if e != nil {
				return e
			}
		}
		out, e = p.caseView(ctx, tx, id)
		if e != nil {
			return e
		}
		raw, _ := json.Marshal(out)
		if _, e = tx.Exec(ctx, `INSERT INTO command_registry(scope,kind,key,request_hash,order_id,response) VALUES($1,$2,$3,$4,$5,$6)`, "staff:"+actor.ID, kind, key, requestHash, out.Order, raw); e != nil {
			return e
		}
		return p.Store.audit(ctx, tx, "", actor.ID, "PAYMENT_CASE_"+action, map[string]any{"case_id": id, "reason": reason, "version": out.Version})
	})
	return
}
