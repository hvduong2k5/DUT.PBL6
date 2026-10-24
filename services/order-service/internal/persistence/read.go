package persistence

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain/core"
)

type reader interface {
	rowReader
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (s *Store) LoadOrder(ctx context.Context, id, scope string) (core.Aggregate, error) {
	if !validID(id) {
		return core.Aggregate{}, ErrInput
	}
	return s.loadOrder(ctx, s.db, id, scope)
}

func (s *Store) loadOrder(ctx context.Context, q reader, id, scope string) (core.Aggregate, error) {
	var snapshotCipher, stateCipher []byte
	var snapshotHash, snapshotID, schemaVersion, status, warehouse, reservation, method, ownerKind, principal, payment, stock string
	var version, subtotal, shipping, final int64
	var customer, guest *string
	var created, updated time.Time
	var expires *time.Time
	var operationalHold bool
	err := q.QueryRow(ctx, `SELECT snapshot_cipher,core_state_cipher,snapshot_hash,snapshot_id,snapshot_schema_version,status,version,
 subtotal,shipping_fee,final_amount,warehouse_id,reservation_id,method,owner_kind,principal_id,customer_id,guest_session_id,
 created_at,updated_at,payment_expires_at,payment_status,stock_status,operational_hold FROM orders WHERE id=$1 AND scope=$2 AND cohort=$3`, id, scope, Cohort).Scan(
		&snapshotCipher, &stateCipher, &snapshotHash, &snapshotID, &schemaVersion, &status, &version, &subtotal, &shipping, &final, &warehouse,
		&reservation, &method, &ownerKind, &principal, &customer, &guest, &created, &updated, &expires, &payment, &stock, &operationalHold)
	if err != nil {
		return core.Aggregate{}, err
	}
	data, err := s.cipher.Open("snapshot:"+id, snapshotCipher)
	if err != nil {
		return core.Aggregate{}, err
	}
	snapshot, err := core.DecodeSnapshot(data)
	if err != nil {
		return core.Aggregate{}, ErrIntegrity
	}
	data, err = s.cipher.Open("state:"+id, stateCipher)
	if err != nil {
		return core.Aggregate{}, err
	}
	var record core.StateRecord
	if json.Unmarshal(data, &record) != nil {
		return core.Aggregate{}, ErrIntegrity
	}
	a, err := core.Restore(snapshot, record)
	if err != nil {
		return core.Aggregate{}, ErrIntegrity
	}
	d := snapshot.Data()
	expectedPayment := "PENDING"
	if method == "COD" {
		expectedPayment = "UNPAID"
	} else if record.FirstPaid || record.Allocation.AllocationID != "" && !record.State.Terminal() {
		expectedPayment = "CONFIRMED"
	} else if record.State.Terminal() {
		expectedPayment = "UNPAID"
		if record.Allocation.AllocationID != "" {
			expectedPayment = "RECONCILIATION_REQUIRED"
		}
	}
	owner := core.Owner{Kind: ownerKind, PrincipalID: principal, CustomerID: customer, GuestSessionID: guest}
	var review bool
	if err = q.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM payment_order_projections WHERE order_id=$1 AND has_open_reconciliation)", id).Scan(&review); err != nil {
		return core.Aggregate{}, err
	}
	if review && record.Allocation.AllocationID == "" && record.State == core.PendingPayment {
		expectedPayment = "RECONCILIATION_REQUIRED"
	}
	var refunded int64
	if err = q.QueryRow(ctx, `SELECT COALESCE(sum(f.amount),0) FROM refunds f JOIN allocations a ON a.receipt_id=f.receipt_id WHERE a.order_id=$1 AND f.status='SUCCEEDED'`, id).Scan(&refunded); err != nil {
		return core.Aggregate{}, err
	}
	if refunded > 0 {
		if record.Allocation.Amount < refunded {
			return core.Aggregate{}, ErrIntegrity
		}
		expectedPayment = "PARTIALLY_REFUNDED"
		if refunded == record.Allocation.Amount {
			expectedPayment = "REFUNDED"
		}
	}
	expectedStock := record.Reservation.State
	if record.State == core.PaymentFinalizing {
		expectedStock = "COMMITTING"
	}
	if a.ID() != id || string(snapshot.Hash()) != snapshotHash || d.SnapshotID != snapshotID || d.SchemaVersion != schemaVersion ||
		string(a.State()) != status || a.Version() != version || d.Money.Subtotal != subtotal || d.Money.ShippingFee != shipping || d.Money.FinalAmount != final ||
		d.WarehouseID != warehouse || record.Reservation.ReservationID != reservation || d.Payment.Method != method || !a.Owner().SameOwner(owner) ||
		a.Owner().PrincipalID != principal || ownerScope(a.Owner()) != scope || !d.PlacedAt.Equal(created) || !record.DecidedAt.Equal(updated) || !sameTime(d.Payment.ExpiresAt, expires) || payment != expectedPayment || stock != expectedStock || operationalHold != (record.Blocks&core.HoldBlock != 0) {
		return core.Aggregate{}, ErrIntegrity
	}
	rows, err := q.Query(ctx, `SELECT sku,quantity,unit_price,line_total,product_name FROM order_items WHERE order_id=$1`, id)
	if err != nil {
		return core.Aggregate{}, err
	}
	defer rows.Close()
	items := make(map[string]core.SnapshotItem, len(d.Items))
	for _, item := range d.Items {
		items[item.SKU] = item
	}
	count := 0
	for rows.Next() {
		var sku, name string
		var qty, price, total int64
		if err = rows.Scan(&sku, &qty, &price, &total, &name); err != nil {
			return core.Aggregate{}, err
		}
		item, ok := items[sku]
		if !ok || item.Quantity != qty || item.UnitPrice != price || item.LineSubtotal != total || item.Name != name {
			return core.Aggregate{}, ErrIntegrity
		}
		count++
	}
	if err = rows.Err(); err != nil {
		return core.Aggregate{}, err
	}
	if count != len(d.Items) {
		return core.Aggregate{}, ErrIntegrity
	}
	return a, nil
}

func sameTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}
