package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
)

// LocalCasePolicyVersion identifies decision records issued by this
// service's own case authority (F-55).
const LocalCasePolicyVersion = "local-case-authority/1.0"

// localCaseAuthority is the in-service implementation of CaseAuthority
// (F-55). The original design read case decisions from an external care
// service (SEPAY_CARE_URL); that service was never deployed, so decisions
// are recorded in this service's own database by PAYMENT_CASE_DECIDE
// holders and consumed by reference at resolve time.
//
// Maker-checker is enforced on both sides of the flow: CreateCaseDecision
// refuses a maker who is the case's assignee, and CaseDecision below stops
// honouring a record whose maker has since become the assignee (the only
// principal allowed to resolve). Refund approvals and the evidence vault
// deliberately keep their external contract; refund execution is deferred.
type localCaseAuthority struct{ p *Payment }

// LocalCaseAuthority returns the decision authority backed by this
// service's own payment_case_decisions table.
func (p *Payment) LocalCaseAuthority() CaseAuthority { return localCaseAuthority{p} }

func (a localCaseAuthority) CanAssign(_ context.Context, receiver, assignee string) (bool, error) {
	// Assignment alone grants nothing: the assignee can only act with
	// their own manage-tier token, and a decision can never be consumed by
	// its maker. Eligibility is therefore just this store's configured
	// receiver plus a well-formed assignee principal id.
	return receiver == a.p.Receiver && validID(assignee), nil
}

func (a localCaseAuthority) CaseDecision(ctx context.Context, ref string) (CaseDecision, error) {
	var d CaseDecision
	var decidedBy string
	var evidence []byte
	e := a.p.Store.db.QueryRow(ctx, `SELECT decision_reference,case_id::text,connection_id::text,receiver_profile_id::text,resolution,allowed,evidence_ids,policy_version,decided_by::text FROM payment_case_decisions WHERE decision_reference=$1`, ref).Scan(&d.Reference, &d.Case, &d.Connection, &d.Receiver, &d.Resolution, &d.Allowed, &evidence, &d.Policy, &decidedBy)
	if errors.Is(e, pgx.ErrNoRows) {
		// Mirror the external client's contract: an unknown decision
		// reference is a capability failure, not a caller input error.
		return CaseDecision{}, ErrCapability
	}
	if e != nil {
		return CaseDecision{}, e
	}
	if d.Connection != a.p.Connection || d.Receiver != a.p.Receiver {
		return CaseDecision{}, ErrCapability
	}
	if json.Unmarshal(evidence, &d.Evidence) != nil {
		return CaseDecision{}, ErrIntegrity
	}
	var assignee *string
	if e = a.p.Store.db.QueryRow(ctx, `SELECT assigned_to::text FROM reconciliation_cases WHERE id=$1`, d.Case).Scan(&assignee); e != nil {
		return CaseDecision{}, e
	}
	if assignee != nil && *assignee == decidedBy {
		d.Allowed = false
	}
	return d, nil
}

// CreateCaseDecision records a resolution decision for an assigned case.
// The maker must hold payment.case.decide and must not be the case's
// assignee. Resolution preconditions mirror the ones CaseCommand enforces
// again at resolve time, so an undecidable resolution fails here instead
// of at consumption.
func (p *Payment) CreateCaseDecision(ctx context.Context, actor PaymentActor, caseID, resolution, reason string, evidence []string) (out CaseDecision, err error) {
	if !actor.allows(p, "payment.case.decide", false) {
		return out, ErrForbidden
	}
	if reason == "" || len(reason) > 1000 || len(evidence) == 0 || len(evidence) > 100 {
		return out, ErrInput
	}
	for _, id := range evidence {
		if id == "" || len(id) > 256 {
			return out, ErrInput
		}
	}
	err = p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		current, e := p.caseView(ctx, tx, caseID)
		if e != nil {
			return e
		}
		if current.Status == "RESOLVED" {
			return ErrConflict
		}
		if current.Assignee == nil {
			// A decision needs a known resolver for the maker-checker rule.
			return ErrConflict
		}
		if *current.Assignee == actor.ID {
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
		raw, _ := json.Marshal(evidence)
		var ref string
		if e = tx.QueryRow(ctx, `INSERT INTO payment_case_decisions(decision_reference,case_id,connection_id,receiver_profile_id,resolution,allowed,evidence_ids,policy_version,decided_by,reason) VALUES(gen_random_uuid()::text,$1,$2,$3,$4,true,$5,$6,$7,$8) RETURNING decision_reference`, caseID, p.Connection, p.Receiver, resolution, raw, LocalCasePolicyVersion, actor.ID, reason).Scan(&ref); e != nil {
			return e
		}
		out = CaseDecision{Reference: ref, Case: caseID, Connection: p.Connection, Receiver: p.Receiver, Resolution: resolution, Allowed: true, Evidence: evidence, Policy: LocalCasePolicyVersion}
		return p.Store.audit(ctx, tx, "", actor.ID, "PAYMENT_CASE_DECISION", map[string]any{"case_id": caseID, "resolution": resolution, "decision_reference": ref})
	})
	return
}

// ListCaseDecisions returns the decision records for one case, newest
// first. Visible to read holders under the same rule as ReadCase, and to
// decide holders (the approver role) for their connection.
func (p *Payment) ListCaseDecisions(ctx context.Context, actor PaymentActor, caseID string) ([]CaseDecision, error) {
	if !actor.allows(p, "payment.reconciliation.read", false) && !actor.allows(p, "payment.case.decide", false) {
		return nil, ErrForbidden
	}
	v, e := p.caseView(ctx, p.Store.db, caseID)
	if e != nil {
		return nil, e
	}
	if !actor.Permissions["payment.reconciliation.manage"] && !actor.Permissions["payment.case.decide"] && (v.Assignee == nil || *v.Assignee != actor.ID) {
		return nil, ErrForbidden
	}
	rows, e := p.Store.db.Query(ctx, `SELECT decision_reference,case_id::text,connection_id::text,receiver_profile_id::text,resolution,allowed,evidence_ids,policy_version FROM payment_case_decisions WHERE case_id=$1 ORDER BY created_at DESC,decision_reference`, caseID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []CaseDecision{}
	for rows.Next() {
		var d CaseDecision
		var evidence []byte
		if e = rows.Scan(&d.Reference, &d.Case, &d.Connection, &d.Receiver, &d.Resolution, &d.Allowed, &evidence, &d.Policy); e != nil {
			return nil, e
		}
		if json.Unmarshal(evidence, &d.Evidence) != nil {
			return nil, ErrIntegrity
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
