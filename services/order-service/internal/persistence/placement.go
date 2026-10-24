package persistence

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain/core"
)

// SHA-256 of the pinned schema file with CRLF normalized to LF.
const PlacedSchemaHash = "324cdc7d83d4a0b3bc1c14602b5277e633f9077aac1aa2ee8f0e1c3f3a764952"
const PlacedSchemaRef = "https://contracts.omama.example/order/v3/placed.event.json"

var ErrClock = errors.New("ORDER_CLOCK_MOVED_BACKWARDS")

type LeaseFence struct {
	JobID, Owner string
	Epoch        int64
}
type Placement struct {
	OperationID string
	Fence       LeaseFence
	Proof       core.ResourceProof
}

// Place consumes a canonical payload prepared by the target worker. PR3 supplies
// the actual canonicalize/reserve/proof/query executor; this method performs no
// external call and cannot be invoked through the sandbox HTTP transport.
func (s *Store) Place(ctx context.Context, in Placement) (core.Aggregate, error) {
	if !validID(in.OperationID) || !validID(in.Fence.JobID) || in.Fence.Owner == "" || in.Fence.Epoch < 1 || !validID(in.Proof.ReservationID) {
		return core.Aggregate{}, ErrInput
	}
	// IDs remain stable through retries of this invocation; operation/order
	// constraints recover committed results across separate invocations.
	intentID, eventID, streamID, cleanupID, cleanupJobID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	var out core.Aggregate
	err := s.runner.Do(ctx, func(tx pgx.Tx) error {
		var planned, scope, principal, status, method string
		var quote, cart string
		var revision int64
		var selection, quoteHash, canonicalHash string
		var canonical []byte
		var acceptedAt time.Time
		var customer, guest *string
		var ownerKind string
		var expectedReservation *string
		err := tx.QueryRow(ctx, `SELECT planned_order_id,scope,principal_id,status,quote_id,cart_id,cart_revision,selection_digest,quote_hash,
 canonical_cipher,COALESCE(canonical_hash,''),accepted_at,owner_kind,customer_id,guest_session_id,payment_method,reservation_id
 FROM checkout_operations WHERE id=$1 AND cohort=$2 FOR UPDATE`, in.OperationID, Cohort).Scan(&planned, &scope, &principal, &status, &quote, &cart, &revision, &selection, &quoteHash, &canonical, &canonicalHash, &acceptedAt, &ownerKind, &customer, &guest, &method, &expectedReservation)
		if err != nil {
			return err
		}
		if status == "SUCCEEDED" {
			var e error
			out, e = s.loadOrder(ctx, tx, planned, scope)
			if e == nil && (in.Proof.OrderID != out.ID() || in.Proof.ReservationID != out.Record().Reservation.ReservationID) {
				return ErrConflict
			}
			return e
		}
		if status != "PROCESSING" || canonical == nil {
			return ErrLease
		}
		if expectedReservation == nil || *expectedReservation != in.Proof.ReservationID {
			return core.ErrProof
		}
		data, err := s.cipher.Open("canonical:"+in.OperationID, canonical)
		if err != nil {
			return err
		}
		if hash(data) != canonicalHash {
			return ErrIntegrity
		}
		var input core.SnapshotInput
		if err = json.Unmarshal(data, &input); err != nil {
			return ErrIntegrity
		}
		owner := core.Owner{Kind: ownerKind, PrincipalID: principal, CustomerID: customer, GuestSessionID: guest}
		if input.OrderID != planned || input.OperationID != in.OperationID || input.Provenance.QuoteID != quote || input.Provenance.CartID != cart ||
			input.Provenance.CartRevision != revision || string(input.Provenance.SelectionDigest) != selection || string(input.Provenance.QuoteHash) != quoteHash ||
			!input.Owner.SameOwner(owner) || input.Owner.PrincipalID != principal || input.Payment.Method != method || !input.Provenance.AcceptedAt.Equal(acceptedAt) {
			return ErrIntegrity
		}
		var connection string
		if method == "VIETQR" {
			if input.Payment.ReceiverProfileID == nil || input.Payment.ReceiverProfileVersion == nil {
				return ErrInput
			}
			var receiverVersion int64
			var enabled bool
			var provider string
			err = tx.QueryRow(ctx, `SELECT r.connection_id,r.config_version,r.accepting_new_payments,c.provider
 FROM payment_receiver_profiles r JOIN payment_provider_connections c ON c.id=r.connection_id WHERE r.id=$1 FOR SHARE OF r`, *input.Payment.ReceiverProfileID).Scan(&connection, &receiverVersion, &enabled, &provider)
			if err != nil {
				return err
			}
			if !enabled || provider != "SEPAY" || receiverVersion != *input.Payment.ReceiverProfileVersion {
				return ErrInput
			}
		}
		var workflowID string
		err = tx.QueryRow(ctx, `SELECT id FROM workflows WHERE kind='CHECKOUT' AND aggregate_type='CHECKOUT_OPERATION' AND aggregate_id=$1
 AND stage='PLACE' AND state='RUNNING' FOR UPDATE`, in.OperationID).Scan(&workflowID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrLease
			}
			return err
		}
		var owned bool
		err = tx.QueryRow(ctx, `SELECT lease_owner=$2 AND lease_epoch=$3 AND lease_until>clock_timestamp()
 FROM jobs WHERE id=$1 AND cohort=$4 AND workflow_id=$5 AND kind='ORDER_PLACE' AND input_version=1 AND stage='PLACE' AND status='LEASED' FOR UPDATE`, in.Fence.JobID, in.Fence.Owner, in.Fence.Epoch, Cohort, workflowID).Scan(&owned)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrLease
			}
			return err
		}
		if !owned {
			return ErrLease
		}
		var at time.Time
		if err = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&at); err != nil {
			return err
		}
		if at.Before(input.Provenance.AcceptedAt) || at.Before(input.Provenance.CanonicalizedAt) {
			return ErrClock
		}
		input.PlacedAt = at // The final snapshot uses the locked transaction's decision time.
		snapshot, err := core.NewSnapshot(input)
		if err != nil {
			return err
		}
		out, err = core.Place(snapshot, planned, in.Proof, at)
		if err != nil {
			return err
		}
		snapCipher, err := s.cipher.Seal("snapshot:"+planned, snapshot.Bytes())
		if err != nil {
			return err
		}
		record, _ := json.Marshal(out.Record())
		stateCipher, err := s.cipher.Seal("state:"+planned, record)
		if err != nil {
			return err
		}
		payment := "PENDING"
		if method == "COD" {
			payment = "UNPAID"
		}
		_, err = tx.Exec(ctx, `INSERT INTO orders(id,operation_id,order_code,scope,principal_id,customer_id,warehouse_id,method,status,payment_status,stock_status,
 subtotal,shipping_fee,final_amount,snapshot_cipher,reservation_id,payment_expires_at,version,policy_version,created_at,updated_at,
 cohort,owner_kind,guest_session_id,snapshot_id,snapshot_schema_version,snapshot_hash,core_state_cipher)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,1,$18,$19,$19,$20,$21,$22,$23,'1.0',$24,$25)`,
			planned, in.OperationID, "ORD-"+planned, scope, principal, customer, input.WarehouseID, method, out.State(), payment, in.Proof.State, input.Money.Subtotal,
			input.Money.ShippingFee, input.Money.FinalAmount, snapCipher, in.Proof.ReservationID, input.Payment.ExpiresAt, input.Policies.Order, at,
			Cohort, ownerKind, guest, input.SnapshotID, snapshot.Hash(), stateCipher)
		if err != nil {
			return err
		}
		for _, item := range input.Items {
			if _, err = tx.Exec(ctx, `INSERT INTO order_items(order_id,sku,quantity,unit_price,line_total,product_name) VALUES($1,$2,$3,$4,$5,$6)`, planned, item.SKU, item.Quantity, item.UnitPrice, item.LineSubtotal, item.Name); err != nil {
				return err
			}
		}
		if method == "VIETQR" {
			// Payment code is stable per Order. The enabled Payment runtime matches
			// canonical API receipts; no network QR creation occurs in T3.
			if err = s.insertIntent(ctx, tx, intentID, planned, connection, input); err != nil {
				return err
			}
		}
		if _, err = tx.Exec(ctx, `INSERT INTO order_history(id,order_id,version,to_status,source,reason,occurred_at) VALUES($1,$2,1,$3,'checkout','PLACE',$4)`, uuid.NewString(), planned, out.State(), at); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO order_milestones(order_id,kind,decision_ref,occurred_at) VALUES($1,'PLACED',$2,$3)`, planned, in.Proof.DecisionID, at); err != nil {
			return err
		}
		if err = s.appendPlaced(ctx, tx, out, eventID, streamID, payment, at); err != nil {
			return err
		}
		guardTag, err := tx.Exec(ctx, `UPDATE checkout_submission_guards SET state='CONSUMED',version=version+1 WHERE operation_id=$1 AND state='ACTIVE'`, in.OperationID)
		if err != nil {
			return err
		}
		if guardTag.RowsAffected() != 1 {
			return ErrIntegrity
		}
		if _, err = tx.Exec(ctx, `UPDATE checkout_operations SET status='SUCCEEDED',stage='SUCCEEDED',order_id=$2,completed_at=$3,error_code=NULL,error_http=NULL,lease_owner=NULL,lease_until=NULL WHERE id=$1`, in.OperationID, planned, at); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE workflows SET state='COMPLETED',completion_outcome='SUCCEEDED',stage='SUCCEEDED',order_id=$2,version=version+1 WHERE id=$1`, workflowID, planned); err != nil {
			return err
		}
		jobTag, err := tx.Exec(ctx, `UPDATE jobs SET status='DONE',lease_owner=NULL,lease_until=NULL,order_id=$2
 WHERE id=$1 AND lease_owner=$3 AND lease_epoch=$4 AND lease_until>clock_timestamp()`, in.Fence.JobID, planned, in.Fence.Owner, in.Fence.Epoch)
		if err != nil {
			return err
		}
		if jobTag.RowsAffected() != 1 {
			return ErrLease
		}
		if err = s.followup(ctx, tx, cleanupID, cleanupJobID, "CART_CLEANUP", planned, in.OperationID, "CLEANUP", at); err != nil {
			return err
		}
		if method == "COD" {
			if err = s.followup(ctx, tx, uuid.NewString(), uuid.NewString(), "FULFILLMENT_READY", planned, in.OperationID, "BARRIER", at); err != nil {
				return err
			}
		}
		return s.audit(ctx, tx, planned, principal, "ORDER_PLACED", map[string]any{"operation_id": in.OperationID, "snapshot_hash": snapshot.Hash()})
	})
	if err != nil && (errors.Is(err, ErrCommitUnknown) || unique(err)) {
		lookup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		var id, scope string
		e := s.db.QueryRow(lookup, `SELECT id,scope FROM orders WHERE operation_id=$1 AND cohort=$2`, in.OperationID, Cohort).Scan(&id, &scope)
		if e == nil {
			return s.loadOrder(lookup, s.db, id, scope)
		}
	}
	if err != nil {
		return core.Aggregate{}, err
	}
	return out, nil
}

func paymentCode() (string, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := []byte("OMP000000000000")
	for i := 3; i < len(code); i++ {
		v, err := rand.Int(rand.Reader, big.NewInt(36))
		if err != nil {
			return "", err
		}
		code[i] = alphabet[v.Int64()]
	}
	return string(code), nil
}

func (s *Store) insertIntent(ctx context.Context, tx pgx.Tx, id, orderID, connection string, input core.SnapshotInput) error {
	for attempt := 0; attempt < 3; attempt++ {
		code, err := paymentCode()
		if err != nil {
			return err
		}
		var stored string
		err = tx.QueryRow(ctx, `INSERT INTO payment_intents(id,order_id,connection_id,receiver_profile_id,payment_code,amount,currency,status,expires_at)
 VALUES($1,$2,$3,$4,$5,$6,'VND','ACTIVE',$7) ON CONFLICT(connection_id,payment_code) DO NOTHING RETURNING id`, id, orderID, connection, *input.Payment.ReceiverProfileID, code, input.Money.FinalAmount, *input.Payment.ExpiresAt).Scan(&stored)
		if err == nil {
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	return ErrConflict
}

func (s *Store) followup(ctx context.Context, tx pgx.Tx, workflow, job, kind, orderID, operationID, lane string, at time.Time) error {
	input, _ := json.Marshal(map[string]string{"order_id": orderID, "operation_id": operationID})
	encrypted, err := s.cipher.Seal("job:"+job, input)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO workflows(id,kind,workflow_version,aggregate_type,aggregate_id,order_id,stage,state,policy_version,stage_started_at)
 VALUES($1,$2,1,'ORDER',$3,$3,'START','RUNNING','order-core-1.0',$4)`, workflow, kind, orderID, at); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO jobs(id,order_id,kind,key,stage,status,cohort,workflow_id,input_version,input_cipher,input_hash,lane)
 VALUES($1,$2,$3,$4,'START','READY',$5,$6,1,$7,$8,$9)`, job, orderID, kind, kind+":"+orderID, Cohort, workflow, encrypted, hash(input), lane)
	return err
}

func (s *Store) appendPlaced(ctx context.Context, tx pgx.Tx, a core.Aggregate, eventID, streamID, payment string, at time.Time) error {
	r := a.Record()
	d := a.Snapshot().Data()
	_, err := tx.Exec(ctx, `INSERT INTO event_streams(id,aggregate_type,aggregate_id,topic,partition_key,routing_version,last_sequence)
 VALUES($1,'ORDER',$2::uuid,'order.events.v3',$2::text,1,1)`, streamID, a.ID())
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{"specversion": "1.0", "id": eventID, "source": "https://omama.vn/services/order-service",
		"type": "vn.omama.order.placed.v3", "subject": "order:" + a.ID(), "time": at.UTC().Format(time.RFC3339Nano), "datacontenttype": "application/json",
		"data": map[string]any{"aggregate_id": a.ID(), "version": int64(1), "ordinal": 0, "status": a.State(), "payment_status": payment,
			"stock_status": r.Reservation.State, "items_digest": d.Provenance.ItemsDigest, "warehouse_id": d.WarehouseID,
			"reservation_id": r.Reservation.ReservationID, "stock_resource_version": r.Reservation.ResourceVersion}})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(event_id,stream_id,aggregate_type,aggregate_id,aggregate_version,ordinal,stream_sequence,order_id,
 event_type,schema_ref,schema_hash,payload_bytes,payload_hash,headers,occurred_at)
 VALUES($1,$2,'ORDER',$3,1,0,1,$3,'vn.omama.order.placed.v3',$4,$5,$6,$7,'{}',$8)`, eventID, streamID, a.ID(), PlacedSchemaRef, PlacedSchemaHash, payload, hash(payload), at)
	return err
}
