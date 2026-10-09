package app

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
	"time"
)

type operationWork struct {
	ID, Scope, Principal, Planned, Status, Stage string
	Input, Canonical                             []byte
	Epoch                                        int64
	Attempts                                     int
	Reservation                                  *string
	Expires, Commercial                          *time.Time
	Accepted                                     time.Time
	Error                                        *string
	ErrorHTTP                                    *int
}

func (s *Service) claimOperation(ctx context.Context) (*operationWork, error) {
	var w operationWork
	e := s.DB.QueryRow(ctx, `WITH picked AS (SELECT id FROM checkout_operations WHERE status IN ('ACCEPTED','PROCESSING','WAITING_RETRY','COMPENSATING') AND run_after<=clock_timestamp() AND (lease_until IS NULL OR lease_until<clock_timestamp()) ORDER BY accepted_at FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE checkout_operations c SET lease_owner=$1,lease_until=clock_timestamp()+interval '10 seconds',lease_epoch=lease_epoch+1,attempts=attempts+1,status=CASE WHEN c.stage='COMPENSATE' THEN 'COMPENSATING' ELSE 'PROCESSING' END FROM picked WHERE c.id=picked.id RETURNING c.id,c.scope,c.principal_id,c.planned_order_id,c.status,c.stage,c.input_cipher,c.canonical_cipher,c.lease_epoch,c.attempts,c.reservation_id,c.reservation_expires_at,c.commercial_expires_at,c.accepted_at,c.error_code,c.error_http`, s.workerID).Scan(&w.ID, &w.Scope, &w.Principal, &w.Planned, &w.Status, &w.Stage, &w.Input, &w.Canonical, &w.Epoch, &w.Attempts, &w.Reservation, &w.Expires, &w.Commercial, &w.Accepted, &w.Error, &w.ErrorHTTP)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, nil
	}
	return &w, e
}
func (s *Service) opTransaction(ctx context.Context, w *operationWork, fn func(pgx.Tx) error) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var ok bool
	e = tx.QueryRow(ctx, `SELECT lease_owner=$2 AND lease_epoch=$3 AND lease_until>clock_timestamp() FROM checkout_operations WHERE id=$1 FOR UPDATE`, w.ID, s.workerID, w.Epoch).Scan(&ok)
	if e != nil {
		return e
	}
	if !ok {
		return domain.E("STALE_LEASE", 409)
	}
	if e = fn(tx); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Service) retryOperation(ctx context.Context, w *operationWork, cause error) {
	code, _ := s.ErrorCode(cause)
	status := "WAITING_RETRY"
	if w.Stage == "COMPENSATE" {
		status = "COMPENSATING"
	}
	if w.Attempts >= 10 {
		status = "MANUAL_REVIEW"
	}
	_ = s.opTransaction(ctx, w, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx, `UPDATE checkout_operations SET status=$2,error_code=COALESCE(error_code,$3),run_after=clock_timestamp()+($4::double precision*interval '1 second'),lease_until=NULL,lease_owner=NULL WHERE id=$1`, w.ID, status, code, backoff(w.Attempts))
		return e
	})
}
func backoff(n int) float64 {
	if n > 6 {
		n = 6
	}
	return .05 * float64(uint64(1)<<uint(n))
}
func (s *Service) failOperation(ctx context.Context, w *operationWork, cause error, compensate bool) error {
	code, status := s.ErrorCode(cause)
	if status >= 500 {
		status = 422
	}
	return s.opTransaction(ctx, w, func(tx pgx.Tx) error {
		if compensate {
			_, e := tx.Exec(ctx, `UPDATE checkout_operations SET status='COMPENSATING',stage='COMPENSATE',error_code=$2,error_http=$3,attempts=0,lease_until=NULL,lease_owner=NULL WHERE id=$1`, w.ID, code, status)
			return e
		}
		_, e := tx.Exec(ctx, `UPDATE checkout_operations SET status='FAILED',error_code=$2,error_http=$3,completed_at=clock_timestamp(),lease_until=NULL,lease_owner=NULL WHERE id=$1`, w.ID, code, status)
		if e == nil {
			s.Metrics.WithLabelValues("checkout_completion", "FAILED").Observe(time.Since(w.Accepted).Seconds())
		}
		return e
	})
}
func (s *Service) WorkPlacement(ctx context.Context) (bool, error) {
	w, e := s.claimOperation(ctx)
	if e != nil || w == nil {
		return false, e
	}
	var stored domain.StoredQuote
	if e = s.Open(w.Input, &stored); e != nil {
		s.retryOperation(ctx, w, e)
		return true, e
	}
	q := stored.Quote
	switch w.Stage {
	case "VALIDATE":
		if q.AddressID != "" {
			address, version, err := s.Address(ctx, stored.Principal, q.AddressID)
			if err != nil {
				code, http := s.ErrorCode(err)
				if http >= 500 {
					s.retryOperation(ctx, w, err)
				} else {
					_ = s.failOperation(ctx, w, domain.E(code, http), false)
				}
				return true, nil
			}
			if domain.Hash(address) != domain.Hash(q.Address) || version != q.AddressVersion {
				_ = s.failOperation(ctx, w, domain.E("ADDRESS_CHANGED", 409), false)
				return true, nil
			}
		}
		q, e = s.Canonical(ctx, q)
		if e != nil {
			_, http := s.ErrorCode(e)
			if http >= 500 {
				s.retryOperation(ctx, w, e)
			} else {
				_ = s.failOperation(ctx, w, e, false)
			}
			return true, nil
		}
		data, err := s.Seal(q)
		if err != nil {
			s.retryOperation(ctx, w, err)
			return true, err
		}
		e = s.opTransaction(ctx, w, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `UPDATE checkout_operations SET canonical_cipher=$2,stage='RESERVE',commercial_expires_at=clock_timestamp()+interval '15 minutes',attempts=0,lease_until=NULL,lease_owner=NULL WHERE id=$1`, w.ID, data)
			return err
		})
	case "RESERVE":
		if e = s.Open(w.Canonical, &q); e != nil {
			return true, e
		}
		var r domain.Reservation
		e = s.Call(ctx, "POST", "/inventory/reserve", map[string]any{"order_id": w.Planned, "operation_key": "checkout:" + w.ID + ":reserve", "items": q.Items, "ttl_seconds": 900}, &r)
		if e != nil {
			_, http := s.ErrorCode(e)
			if http >= 500 {
				s.retryOperation(ctx, w, e)
			} else {
				_ = s.failOperation(ctx, w, e, true)
			}
			return true, nil
		}
		if r.OrderID != w.Planned || r.State != "RESERVED" || r.ID == "" {
			_ = s.failOperation(ctx, w, domain.E("RESERVATION_EXPIRED", 409), true)
			return true, nil
		}
		stage := "PLACE"
		if q.Method == "COD" {
			stage = "COD_FINALIZE"
		}
		e = s.opTransaction(ctx, w, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `UPDATE checkout_operations SET reservation_id=$2,reservation_expires_at=$3,stage=$4,attempts=0,lease_until=NULL,lease_owner=NULL WHERE id=$1`, w.ID, r.ID, r.ExpiresAt, stage)
			return err
		})
	case "COD_FINALIZE":
		var r domain.Reservation
		e = s.Call(ctx, "POST", "/inventory/finalize", map[string]any{"order_id": w.Planned, "operation_key": "checkout:" + w.ID + ":finalize"}, &r)
		if e != nil {
			s.retryOperation(ctx, w, e)
			return true, nil
		}
		if r.State != "COMMITTED" {
			_ = s.failOperation(ctx, w, domain.E("RESERVATION_EXPIRED", 409), true)
			return true, nil
		}
		e = s.opTransaction(ctx, w, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `UPDATE checkout_operations SET stage='PLACE',attempts=0,lease_until=NULL,lease_owner=NULL WHERE id=$1`, w.ID)
			return err
		})
	case "PLACE":
		if e = s.Open(w.Canonical, &q); e != nil {
			return true, e
		}
		if w.Expires == nil || w.Commercial == nil || w.Reservation == nil {
			return true, errors.New("missing durable reservation")
		}
		expires := *w.Expires
		if w.Commercial.Before(expires) {
			expires = *w.Commercial
		}
		if q.CatalogExpiresAt.Before(expires) {
			expires = q.CatalogExpiresAt
		}
		if q.ShippingExpiresAt.Before(expires) {
			expires = q.ShippingExpiresAt
		}
		status, stock, payment := "PENDING_PAYMENT", "RESERVED", "PENDING"
		if q.Method == "COD" {
			status, stock, payment = "CONFIRMED_COD", "COMMITTED", "UNPAID"
		}
		data, err := s.Seal(q)
		if err != nil {
			return true, err
		}
		e = s.opTransaction(ctx, w, func(tx pgx.Tx) error {
			var now time.Time
			if err := tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
				return err
			}
			if !now.Add(30*time.Second).Before(expires) && q.Method != "COD" {
				return domain.E("RESERVATION_EXPIRED", 409)
			}
			_, err := tx.Exec(ctx, `INSERT INTO orders(id,operation_id,order_code,scope,principal_id,customer_id,method,status,payment_status,stock_status,subtotal,shipping_fee,final_amount,snapshot_cipher,reservation_id,payment_expires_at,provider_reference,sla_due_at) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,clock_timestamp()+interval '4 hours')`, w.Planned, w.ID, "ORD-"+w.Planned, w.Scope, w.Principal, q.CustomerID, q.Method, status, payment, stock, q.Subtotal, q.Shipping, q.Final, data, *w.Reservation, expires, "OMAMA-"+w.Planned)
			if err != nil {
				return err
			}
			for _, i := range q.Items {
				_, err = tx.Exec(ctx, "INSERT INTO order_items(order_id,sku,quantity,unit_price,line_total,product_name) VALUES($1,$2,$3,$4,$5,$6)", w.Planned, i.SKU, i.Quantity, i.UnitPrice, i.UnitPrice*int64(i.Quantity), i.Name)
				if err != nil {
					return err
				}
			}
			_, err = tx.Exec(ctx, "INSERT INTO order_history(id,order_id,version,to_status,source) VALUES($1,$2,1,$3,'CHECKOUT')", uuid.NewString(), w.Planned, status)
			if err != nil {
				return err
			}
			if err = s.Emit(ctx, tx, w.Planned, "ORDER", 1, 0, "placed", status, payment, stock, 0); err != nil {
				return err
			}
			if q.Method == "COD" {
				if err = s.insertJob(ctx, tx, w.Planned, "READY", "ready:"+w.Planned, nil); err != nil {
					return err
				}
			}
			if err = s.Audit(ctx, tx, w.Planned, w.Principal, "ORDER_PLACED", map[string]any{"operation_id": w.ID}); err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `UPDATE checkout_operations SET order_id=$2,status='SUCCEEDED',error_code=NULL,error_http=NULL,completed_at=clock_timestamp(),lease_until=NULL,lease_owner=NULL WHERE id=$1`, w.ID, w.Planned)
			return err
		})
		if e != nil {
			code, http := s.ErrorCode(e)
			if http < 500 {
				_ = s.failOperation(ctx, w, domain.E(code, http), true)
			} else {
				s.retryOperation(ctx, w, e)
			}
		} else {
			s.Metrics.WithLabelValues("checkout_completion", "SUCCEEDED").Observe(time.Since(w.Accepted).Seconds())
		}
	case "COMPENSATE":
		var r domain.Reservation
		e = s.Call(ctx, "POST", "/inventory/release", map[string]any{"order_id": w.Planned, "operation_key": "checkout:" + w.ID + ":release"}, &r)
		if e != nil {
			s.retryOperation(ctx, w, e)
			return true, nil
		}
		if r.State == "COMMITTED" {
			s.retryOperation(ctx, w, domain.E("COMMITTED_STOCK_REQUIRES_REVIEW", 409))
			return true, nil
		}
		e = s.opTransaction(ctx, w, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `UPDATE checkout_operations SET status='FAILED',completed_at=clock_timestamp(),lease_until=NULL,lease_owner=NULL WHERE id=$1`, w.ID)
			return err
		})
		if e == nil {
			s.Metrics.WithLabelValues("checkout_completion", "FAILED").Observe(time.Since(w.Accepted).Seconds())
		}
	default:
		e = errors.New("unknown durable stage")
		s.retryOperation(ctx, w, e)
	}
	return true, e
}
func (s *Service) insertJob(ctx context.Context, tx pgx.Tx, order, kind, key string, payload any) error {
	if payload == nil {
		payload = map[string]any{}
	}
	b, e := jsonMarshal(payload)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO jobs(id,order_id,kind,key,payload) VALUES($1,$2,$3,$4,$5) ON CONFLICT(key) DO NOTHING`, uuid.NewString(), order, kind, key, b)
	return e
}
