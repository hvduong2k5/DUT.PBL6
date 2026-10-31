package persistence

import (
	"context"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain/core"
	"github.com/omamx/order-service/internal/sepay"
)

// Payment binds one stable merchant/environment to immutable receiver history.
// It is not permitted to ingest into the legacy simulator cohort.
type Payment struct {
	GrantKey           *ecdsa.PrivateKey
	GrantKeyID         string
	Store              *Store
	Connection         string
	Receiver           string
	Egress             string
	Secret             string
	KeyVersion         string
	PreviousSecret     string
	PreviousKeyVersion string
	PreviousValidUntil time.Time
	Bootstrap          time.Time
}

func (p *Payment) referenceKey(reference, account, va, direction string, amount int64, date, content string) any {
	if reference == "" {
		return nil
	}
	at, e := sepay.BankTime(date)
	if e != nil {
		return nil
	}
	raw, _ := json.Marshal([]any{reference, account, va, direction, amount, at, content})
	mac := hmac.New(sha256.New, p.Store.auditKey)
	mac.Write(raw)
	return hex.EncodeToString(mac.Sum(nil))
}

func (p *Payment) Validate() error {
	if p.Store == nil || !validID(p.Connection) || !validID(p.Receiver) || p.Egress == "" || len(p.Secret) < 32 || p.KeyVersion == "" || p.Bootstrap.IsZero() {
		return ErrInput
	}
	return nil
}
func (p *Payment) Onboard(ctx context.Context, receiver sepay.Receiver) error {
	if p.Validate() != nil || receiver.Validate() != nil {
		return ErrInput
	}
	return p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", "sepay:onboard:"+p.Receiver); e != nil {
			return e
		}
		var provider, environment string
		if e := tx.QueryRow(ctx, "SELECT provider,environment FROM payment_provider_connections WHERE id=$1", p.Connection).Scan(&provider, &environment); e != nil {
			return e
		}
		if provider != "SEPAY" || environment != "TEST" {
			return ErrInput
		}
		var bank string
		var encrypted []byte
		if e := tx.QueryRow(ctx, "SELECT bank_account_uuid,config_cipher FROM payment_receiver_profiles WHERE id=$1 AND connection_id=$2", p.Receiver, p.Connection).Scan(&bank, &encrypted); e != nil {
			return e
		}
		data, e := p.Store.cipher.Open("receiver:"+p.Receiver, encrypted)
		if e != nil {
			return e
		}
		var stored sepay.Receiver
		if json.Unmarshal(data, &stored) != nil || stored != receiver || bank != receiver.BankID {
			return ErrIntegrity
		}
		for _, stream := range []string{"FAST", "OVERLAP", "DAILY", "BACKFILL"} {
			if _, e = tx.Exec(ctx, `INSERT INTO payment_sync_checkpoints(receiver_id,stream,bootstrap_at) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, p.Receiver, stream, p.Bootstrap); e != nil {
				return e
			}
			key := "sepay:sync:" + p.Receiver + ":" + stream
			var exists bool
			if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM jobs WHERE key=$1)", key).Scan(&exists); e != nil {
				return e
			}
			if exists {
				continue
			}
			workflow, job := uuid.NewString(), uuid.NewString()
			raw, _ := json.Marshal(map[string]string{"receiver_id": p.Receiver, "stream": stream})
			cipher, e := p.Store.cipher.Seal("job:"+job, raw)
			if e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO workflows(id,kind,workflow_version,aggregate_type,aggregate_id,stage,state,policy_version,stage_started_at) VALUES($1,'PAYMENT_SYNC',1,'RECEIVER',$2,'SCAN','RUNNING','payment-sepay-1.1',clock_timestamp())`, workflow, p.Receiver); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO jobs(id,kind,key,stage,status,cohort,workflow_id,input_version,input_cipher,input_hash,lane) VALUES($1,'PAYMENT_SYNC',$2,'SCAN','READY',$3,$4,1,$5,$6,'RECOVERY')`, job, key, Cohort, workflow, cipher, hash(raw)); e != nil {
				return e
			}
		}
		return nil
	})
}

// Observe has no provider/network call and never creates a financial receipt.
func (p *Payment) Observe(ctx context.Context, raw []byte, timestamp, signature string) error {
	if p.Validate() != nil {
		return ErrInput
	}
	if len(raw) > 65536 {
		return ErrInput
	}
	keyVersion := p.KeyVersion
	if e := sepay.Verify(raw, timestamp, signature, p.Secret, time.Now()); e != nil {
		if len(p.PreviousSecret) < 32 || p.PreviousKeyVersion == "" || !time.Now().Before(p.PreviousValidUntil) || sepay.Verify(raw, timestamp, signature, p.PreviousSecret, time.Now()) != nil {
			return e
		}
		keyVersion = p.PreviousKeyVersion
	}
	w, e := sepay.ParseWebhook(raw)
	if e != nil {
		return e
	}
	n := w
	n.Code = ""
	normalized, _ := json.Marshal(n)
	id, conflict := uuid.NewString(), uuid.NewString()
	cipher, e := p.Store.cipher.Seal("observation:"+id, raw)
	if e != nil {
		return e
	}
	return p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		tag, e := tx.Exec(ctx, `INSERT INTO payment_provider_observations(id,connection_id,external_id,normalized_hash,raw_hash,raw_cipher,auth_key_version,reference_key,receiver_profile_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(connection_id,external_id) DO NOTHING`, id, p.Connection, strconv.FormatInt(w.ID, 10), hash(normalized), hash(raw), cipher, keyVersion, p.referenceKey(w.Reference, w.Account, w.VA, w.Direction, w.Amount, w.Date, w.Content), p.Receiver)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			var original, oldHash string
			if e = tx.QueryRow(ctx, "SELECT id,normalized_hash FROM payment_provider_observations WHERE connection_id=$1 AND external_id=$2 FOR UPDATE", p.Connection, strconv.FormatInt(w.ID, 10)).Scan(&original, &oldHash); e != nil {
				return e
			}
			id = original
			if oldHash != hash(normalized) {
				evidence, e := p.Store.cipher.Seal("observation-conflict:"+conflict, raw)
				if e != nil {
					return e
				}
				if _, e = tx.Exec(ctx, `INSERT INTO payment_observation_conflicts(id,observation_id,incoming_hash,raw_cipher,reason) VALUES($1,$2,$3,$4,'WEBHOOK_DATA_CONFLICT') ON CONFLICT DO NOTHING`, conflict, id, hash(normalized), evidence); e != nil {
					return e
				}
				if _, e = tx.Exec(ctx, `INSERT INTO reconciliation_cases(id,observation_id,reason,evidence_hash) VALUES($1,$2,'WEBHOOK_DATA_CONFLICT',$3) ON CONFLICT DO NOTHING`, uuid.NewString(), id, hash(normalized)); e != nil {
					return e
				}
			}
			if _, e = tx.Exec(ctx, "UPDATE payment_provider_observations SET last_seen_at=clock_timestamp() WHERE id=$1", id); e != nil {
				return e
			}
		}
		// Wake existing FAST work without breaking its active lease or resetting its budget.
		var receiverCipher []byte
		if e = tx.QueryRow(ctx, "SELECT config_cipher FROM payment_receiver_profiles WHERE id=$1 AND connection_id=$2", p.Receiver, p.Connection).Scan(&receiverCipher); e != nil {
			return e
		}
		receiverRaw, e := p.Store.cipher.Open("receiver:"+p.Receiver, receiverCipher)
		if e != nil {
			return e
		}
		var configured sepay.Receiver
		if json.Unmarshal(receiverRaw, &configured) != nil {
			return ErrIntegrity
		}
		invalidReason := ""
		if w.Direction != "in" {
			invalidReason = "OUTGOING_OBSERVATION"
		} else if w.Account != configured.Account || w.VA != "" {
			invalidReason = "RECEIVER_MISMATCH"
		}
		if invalidReason != "" {
			if _, e = tx.Exec(ctx, "UPDATE payment_provider_observations SET status='MANUAL_REVIEW' WHERE id=$1", id); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, "INSERT INTO reconciliation_cases(id,observation_id,reason) VALUES($1,$2,$3) ON CONFLICT DO NOTHING", uuid.NewString(), id, invalidReason); e != nil {
				return e
			}
		}
		tag, e = tx.Exec(ctx, `UPDATE jobs SET status=CASE WHEN status='LEASED' THEN status ELSE 'READY' END,run_after=LEAST(run_after,clock_timestamp()+interval '500 milliseconds') WHERE key=$1 AND cohort=$2`, "sepay:sync:"+p.Receiver+":FAST", Cohort)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return ErrIntegrity
		}
		code, reason := sepay.PaymentCode(w.Content, w.Code)
		if reason == "" && w.Direction == "in" {
			var order string
			var expiry time.Time
			var receiver sepay.Receiver
			var encrypted []byte
			if e = tx.QueryRow(ctx, `SELECT i.order_id,i.expires_at,r.config_cipher FROM payment_intents i JOIN payment_receiver_profiles r ON r.id=i.receiver_profile_id WHERE i.connection_id=$1 AND i.payment_code=$2 AND i.receiver_profile_id=$3`, p.Connection, code, p.Receiver).Scan(&order, &expiry, &encrypted); e != nil && !errors.Is(e, pgx.ErrNoRows) {
				return e
			}
			if e == nil {
				data, e := p.Store.cipher.Open("receiver:"+p.Receiver, encrypted)
				if e != nil {
					return e
				}
				if json.Unmarshal(data, &receiver) != nil {
					return ErrIntegrity
				}
				if w.Account == receiver.Account && w.VA == "" {
					_, e = tx.Exec(ctx, `INSERT INTO payment_order_projections(order_id,verification_started_at,verification_due_at,support_reference) VALUES($1,clock_timestamp(),LEAST(clock_timestamp()+interval '60 seconds',$2),$3) ON CONFLICT(order_id) DO UPDATE SET verification_started_at=COALESCE(payment_order_projections.verification_started_at,excluded.verification_started_at),verification_due_at=COALESCE(payment_order_projections.verification_due_at,excluded.verification_due_at),support_reference=COALESCE(payment_order_projections.support_reference,excluded.support_reference),version=payment_order_projections.version+1`, order, expiry, uuid.NewString())
					if e != nil {
						return e
					}
				}
			}
		}
		return nil
	})
}

// savePaymentRoot changes only live state/projections, never the commercial snapshot.
func (s *Store) savePaymentRoot(ctx context.Context, tx pgx.Tx, old core.Aggregate, next core.Aggregate, payment, reason string) error {
	r := next.Record()
	raw, e := json.Marshal(r)
	if e != nil {
		return e
	}
	encrypted, e := s.cipher.Seal("state:"+next.ID(), raw)
	if e != nil {
		return e
	}
	stock := r.Reservation.State
	if next.State() == core.PaymentFinalizing {
		stock = "COMMITTING"
	}
	tag, e := tx.Exec(ctx, `UPDATE orders SET status=$2,payment_status=$3,stock_status=$4,version=$5,updated_at=$6,core_state_cipher=$7,operational_hold=$8,ready_authorized=false WHERE id=$1 AND cohort=$9 AND version=$10`, next.ID(), next.State(), payment, stock, next.Version(), r.DecidedAt, encrypted, r.Blocks&core.HoldBlock != 0, Cohort, old.Version())
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	if next.Version() != old.Version() {
		if _, e = tx.Exec(ctx, `INSERT INTO order_history(id,order_id,version,from_status,to_status,source,reason,occurred_at) VALUES($1,$2,$3,$4,$5,'PAYMENT',$6,$7)`, uuid.NewString(), next.ID(), next.Version(), old.State(), next.State(), reason, r.DecidedAt); e != nil {
			return e
		}
	}
	return s.audit(ctx, tx, next.ID(), "PAYMENT", reason, map[string]any{"version": next.Version()})
}

// Ingest takes a canonical API row; caller supplies receiver/run provenance, not a webhook ID.
func (p *Payment) Ingest(ctx context.Context, t sepay.Transaction) error {
	return p.ingest(ctx, t, nil)
}
func (p *Payment) ingest(ctx context.Context, t sepay.Transaction, fence *PaymentJob) error {
	facts, e := t.FinancialBytes()
	if e != nil {
		return e
	}
	code, reason := sepay.PaymentCode(t.Content, t.Code)
	var candidate string
	var existingOrder *string
	knownIdentity := false
	e = p.Store.db.QueryRow(ctx, "SELECT order_id FROM receipts WHERE connection_id=$1 AND transaction_id=$2", p.Connection, t.ID).Scan(&existingOrder)
	if e == nil {
		knownIdentity = true
		if existingOrder != nil {
			candidate = *existingOrder
		}
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	if code != "" && !knownIdentity {
		e = p.Store.db.QueryRow(ctx, "SELECT order_id FROM payment_intents WHERE connection_id=$1 AND payment_code=$2", p.Connection, code).Scan(&candidate)
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
	}
	receiptID := uuid.NewString()
	evidence, e := p.Store.cipher.Seal("receipt:"+receiptID, facts)
	if e != nil {
		return e
	}
	return p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		var a core.Aggregate
		var scope string
		var err error
		if candidate != "" {
			if err = tx.QueryRow(ctx, "SELECT scope FROM orders WHERE id=$1 AND cohort=$2 FOR UPDATE", candidate, Cohort).Scan(&scope); err != nil {
				return err
			}
			a, err = p.Store.loadOrder(ctx, tx, candidate, scope)
			if err != nil {
				return err
			}
		}
		var encrypted []byte
		var bank string
		if err = tx.QueryRow(ctx, "SELECT config_cipher,bank_account_uuid FROM payment_receiver_profiles WHERE id=$1 AND connection_id=$2", p.Receiver, p.Connection).Scan(&encrypted, &bank); err != nil {
			return err
		}
		data, err := p.Store.cipher.Open("receiver:"+p.Receiver, encrypted)
		if err != nil {
			return err
		}
		var receiver sepay.Receiver
		if json.Unmarshal(data, &receiver) != nil || receiver.Validate() != nil {
			return ErrIntegrity
		}
		receiverOK := t.BankAccountID == bank && t.Account == receiver.Account && t.VA == "" && t.VAID == "" && t.Direction == "in"
		var intent, profile, intentStatus string
		var amount int64
		if candidate != "" && !knownIdentity {
			if err = tx.QueryRow(ctx, `SELECT id,receiver_profile_id,status,amount FROM payment_intents WHERE order_id=$1 AND connection_id=$2 AND payment_code=$3 FOR UPDATE`, candidate, p.Connection, code).Scan(&intent, &profile, &intentStatus, &amount); err != nil {
				return err
			}
			if profile != p.Receiver {
				return ErrInput
			}
		}
		// Serializes missing identities across webhook/scheduled scans and separate candidate Orders.
		if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", p.Connection+":"+t.ID); err != nil {
			return err
		}
		var existing, oldHash string
		var existingOrder *string
		err = tx.QueryRow(ctx, "SELECT id,payload_hash,order_id FROM receipts WHERE connection_id=$1 AND transaction_id=$2 FOR UPDATE", p.Connection, t.ID).Scan(&existing, &oldHash, &existingOrder)
		if err == nil {
			// A concurrent first observation may have pinned a different root.
			// Retry from the original association instead of modifying an unlocked order.
			if existingOrder != nil && *existingOrder != candidate || existingOrder == nil && candidate != "" {
				return ErrConflict
			}
			if oldHash == hash(facts) {
				if fence != nil {
					if e := p.fence(ctx, tx, *fence); e != nil {
						return e
					}
				}
				return p.linkObservations(ctx, tx, t)
			}
			conflictCipher, e := p.Store.cipher.Seal("receipt-conflict:"+existing+":"+hash(facts), facts)
			if e != nil {
				return e
			}
			// Never overwrite financial facts. Original order association is authoritative.
			if fence != nil {
				if e = p.fence(ctx, tx, *fence); e != nil {
					return e
				}
			}
			if _, e = tx.Exec(ctx, `INSERT INTO reconciliation_cases(id,receipt_id,reason,evidence_hash,evidence_cipher) VALUES($1,$2,'PROVIDER_DATA_CONFLICT',$3,$4) ON CONFLICT DO NOTHING`, uuid.NewString(), existing, hash(facts), conflictCipher); e != nil {
				return e
			}
			if candidate != "" {
				if _, e = tx.Exec(ctx, `INSERT INTO payment_order_projections(order_id,payment_hold,has_open_reconciliation,support_reference,support_reason) VALUES($1,true,true,$2,'PROVIDER_DATA_CONFLICT') ON CONFLICT(order_id) DO UPDATE SET payment_hold=true,has_open_reconciliation=true,support_reason='PROVIDER_DATA_CONFLICT',version=payment_order_projections.version+1`, candidate, uuid.NewString()); e != nil {
					return e
				}
				if !a.State().Terminal() {
					var now time.Time
					if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
						return e
					}
					next, e := a.Block(core.DecisionContext{ExpectedVersion: a.Version(), At: now}, core.HoldBlock)
					if e != nil {
						return e
					}
					status := "CONFIRMED"
					if a.Record().Allocation.AllocationID == "" {
						status = "RECONCILIATION_REQUIRED"
					}
					return p.Store.savePaymentRoot(ctx, tx, a, next, status, "PROVIDER_DATA_CONFLICT")
				}
			}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if fence != nil {
			if e := p.fence(ctx, tx, *fence); e != nil {
				return e
			}
		}
		if !receiverOK {
			return ErrInput
		}
		var now time.Time
		if err = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
			return err
		}
		at, _ := sepay.BankTime(t.Date)
		var order any
		if candidate != "" {
			order = candidate
		}
		if _, err = tx.Exec(ctx, `INSERT INTO receipts(id,provider,transaction_id,connection_id,receiver_profile_id,order_id,reference,receiver,amount,currency,payload_hash,status,provider_status,provider_paid_at,evidence_cipher,api_observed_at,reference_key) VALUES($1,'SEPAY',$2,$3,$4::uuid,$5,$6,$4::text,$7,'VND',$8,'VERIFIED','SUCCEEDED',$9,$10,$11,$12)`, receiptID, t.ID, p.Connection, p.Receiver, order, code, t.In, hash(facts), at, evidence, now, p.referenceKey(t.Reference, t.Account, t.VA, t.Direction, t.In, t.Date, t.Content)); err != nil {
			return err
		}
		if candidate == "" {
			if reason == "" {
				reason = "UNMATCHED_REFERENCE"
			}
		} else {
			var allocated bool
			if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM allocations WHERE order_id=$1)", candidate).Scan(&allocated); err != nil {
				return err
			}
			if allocated {
				reason = "EXTRA_RECEIPT"
			} else if reason == "" {
				if a.State() != core.PendingPayment || !now.Before(*a.Snapshot().Data().Payment.ExpiresAt) || intentStatus != "ACTIVE" {
					reason = "LATE_PAYMENT"
				} else if a.Record().Blocks != 0 {
					reason = "BLOCKED_PAYMENT"
					if t.In < amount {
						var partial bool
						if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM receipts WHERE order_id=$1 AND id<>$2 AND amount<$3)", candidate, receiptID, amount).Scan(&partial); err != nil {
							return err
						}
						if partial {
							reason = "MULTIPLE_PARTIAL_PAYMENTS"
						}
					}
				} else if t.In < amount {
					reason = "UNDERPAYMENT"
				} else if t.In > amount {
					reason = "OVERPAYMENT"
				}
			}
		}
		if reason != "" {
			if _, err = tx.Exec(ctx, "UPDATE receipts SET status='RECONCILIATION_REQUIRED' WHERE id=$1", receiptID); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, "INSERT INTO reconciliation_cases(id,receipt_id,reason) VALUES($1,$2,$3)", uuid.NewString(), receiptID, reason); err != nil {
				return err
			}
			if candidate != "" {
				if _, err = tx.Exec(ctx, `INSERT INTO payment_order_projections(order_id,payment_hold,has_open_reconciliation,support_reference,support_reason) VALUES($1,true,true,$2,$3) ON CONFLICT(order_id) DO UPDATE SET payment_hold=true,has_open_reconciliation=true,support_reason=excluded.support_reason,version=payment_order_projections.version+1`, candidate, uuid.NewString(), reason); err != nil {
					return err
				}
				if !a.State().Terminal() {
					next, e := a.Block(core.DecisionContext{ExpectedVersion: a.Version(), At: now}, core.HoldBlock)
					if e != nil {
						return e
					}
					payment := "CONFIRMED"
					if a.Record().Allocation.AllocationID == "" {
						payment = "RECONCILIATION_REQUIRED"
					}
					if err = p.Store.savePaymentRoot(ctx, tx, a, next, payment, reason); err != nil {
						return err
					}
				}
			}
		} else {
			allocationID := uuid.NewString()
			next, e := a.Allocate(core.DecisionContext{ExpectedVersion: a.Version(), At: now}, core.AllocationEvidence{OrderID: candidate, AllocationID: allocationID, ReceiptID: receiptID, ReceiverProfileID: p.Receiver, Amount: t.In, Currency: "VND"})
			if e != nil {
				return e
			}
			if _, err = tx.Exec(ctx, "INSERT INTO allocations(receipt_id,order_id,amount) VALUES($1,$2,$3)", receiptID, candidate, t.In); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, "UPDATE receipts SET status='ALLOCATED' WHERE id=$1", receiptID); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, "UPDATE payment_intents SET status='SETTLED',version=version+1 WHERE id=$1", intent); err != nil {
				return err
			}
			if err = p.Store.savePaymentRoot(ctx, tx, a, next, "CONFIRMED", "EXACT_RECEIPT_ALLOCATED"); err != nil {
				return err
			}
			if err = p.Store.followup(ctx, tx, uuid.NewString(), uuid.NewString(), "PAYMENT_FINALIZE", candidate, a.Snapshot().Data().OperationID, "BARRIER", now); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `INSERT INTO payment_order_projections(order_id) VALUES($1) ON CONFLICT(order_id) DO UPDATE SET version=payment_order_projections.version+1`, candidate); err != nil {
				return err
			}
		}
		if err = p.linkObservations(ctx, tx, t); err != nil {
			return err
		}
		return p.receiptEvent(ctx, tx, receiptID, t, order, hash(facts), now)
	})
}

func (p *Payment) linkObservations(ctx context.Context, tx pgx.Tx, t sepay.Transaction) error {
	key := p.referenceKey(t.Reference, t.Account, t.VA, t.Direction, t.In, t.Date, t.Content)
	if key == nil {
		return nil
	}
	var count int
	var receipt *string
	if e := tx.QueryRow(ctx, "SELECT count(*),min(id::text) FROM receipts WHERE connection_id=$1 AND reference_key=$2", p.Connection, key).Scan(&count, &receipt); e != nil {
		return e
	}
	if count == 1 {
		_, e := tx.Exec(ctx, "UPDATE payment_provider_observations SET status='LINKED',linked_receipt=$3 WHERE connection_id=$1 AND reference_key=$2 AND status IN ('RECEIVED','LINKED')", p.Connection, key, *receipt)
		return e
	}
	if _, e := tx.Exec(ctx, "UPDATE payment_provider_observations SET status='MANUAL_REVIEW',linked_receipt=NULL WHERE connection_id=$1 AND reference_key=$2", p.Connection, key); e != nil {
		return e
	}
	_, e := tx.Exec(ctx, `INSERT INTO reconciliation_cases(id,observation_id,reason) SELECT gen_random_uuid(),id,'AMBIGUOUS_REFERENCE_KEY' FROM payment_provider_observations WHERE connection_id=$1 AND reference_key=$2 ON CONFLICT DO NOTHING`, p.Connection, key)
	return e
}
