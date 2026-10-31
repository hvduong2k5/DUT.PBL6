package simulator

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/order-service/internal/domain"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"
)

//go:embed schema.sql
var schema string

type Simulator struct {
	DB                                             *pgxpool.Pool
	Token, JWT, CallbackSecret, Receiver, OrderURL string
	Client                                         *http.Client
}

func New(ctx context.Context, dsn, token, jwt, callback, receiver string) (*Simulator, error) {
	if dsn == "" || len(token) < 16 || len(jwt) < 32 || len(callback) < 32 || receiver == "" {
		return nil, errors.New("simulator requires explicit database and credentials")
	}
	db, e := pgxpool.New(ctx, dsn)
	if e != nil {
		return nil, e
	}
	if _, e = db.Exec(ctx, schema); e != nil {
		db.Close()
		return nil, e
	}
	return &Simulator{DB: db, Token: token, JWT: jwt, CallbackSecret: callback, Receiver: receiver, Client: &http.Client{Timeout: 2 * time.Second}}, nil
}
func response(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func errResponse(w http.ResponseWriter, e error) {
	var d *domain.Error
	if errors.As(e, &d) {
		response(w, d.HTTP, map[string]string{"code": d.Code})
		return
	}
	response(w, 500, map[string]string{"code": "SIMULATOR_ERROR"})
}
func read(r *http.Request, v any) error {
	defer r.Body.Close()
	d := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func (s *Simulator) Fault(ctx context.Context, name string) string {
	var mode string
	e := s.DB.QueryRow(ctx, "UPDATE faults SET remaining=remaining-1 WHERE name=$1 AND remaining>0 RETURNING mode", name).Scan(&mode)
	if e != nil {
		return ""
	}
	return mode
}
func (s *Simulator) SetFault(ctx context.Context, name, mode string, count int) error {
	_, e := s.DB.Exec(ctx, "INSERT INTO faults(name,mode,remaining) VALUES($1,$2,$3) ON CONFLICT(name) DO UPDATE SET mode=EXCLUDED.mode,remaining=EXCLUDED.remaining", name, mode, count)
	return e
}
func (s *Simulator) after(ctx context.Context, w http.ResponseWriter, name string, v any) {
	switch s.Fault(ctx, name) {
	case "lost_ack":
		response(w, 503, map[string]string{"code": "DEPENDENCY_UNKNOWN"})
		return
	case "delay_ack":
		time.Sleep(2500 * time.Millisecond)
	}
	response(w, 200, v)
}
func (s *Simulator) inventory(ctx context.Context, action, order, key string, items []domain.Item, ttl int, approval string, cod bool) (domain.Reservation, error) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return domain.Reservation{}, e
	}
	defer tx.Rollback(ctx)
	if _, e = uuid.Parse(order); e != nil {
		return domain.Reservation{}, domain.E("INVALID_ORDER_ID", 400)
	}
	if len(key) < 10 {
		return domain.Reservation{}, domain.E("INVALID_KEY", 400)
	}
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1)::bigint)", order); e != nil {
		return domain.Reservation{}, e
	}
	var r domain.Reservation
	var raw []byte
	var oldHash string
	e = tx.QueryRow(ctx, "SELECT id,order_id,state,expires_at,items,input_hash FROM reservations WHERE order_id=$1 FOR UPDATE", order).Scan(&r.ID, &r.OrderID, &r.State, &r.ExpiresAt, &raw, &oldHash)
	exists := e == nil
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return r, e
	}
	var oldItems []domain.Item
	if exists {
		if e = json.Unmarshal(raw, &oldItems); e != nil {
			return r, e
		}
	}
	var now time.Time
	if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
		return r, e
	}
	if exists && r.State == "RESERVED" && !now.Before(r.ExpiresAt) {
		for _, i := range oldItems {
			if _, e = tx.Exec(ctx, "UPDATE stock SET available=available+$2 WHERE sku=$1", i.SKU, i.Quantity); e != nil {
				return r, e
			}
		}
		r.State = "EXPIRED"
		if _, e = tx.Exec(ctx, "UPDATE reservations SET state='EXPIRED' WHERE order_id=$1", order); e != nil {
			return r, e
		}
	}
	switch action {
	case "reserve":
		for index := range items {
			items[index] = domain.Item{SKU: items[index].SKU, Quantity: items[index].Quantity}
		}
		sort.Slice(items, func(i, j int) bool { return items[i].SKU < items[j].SKU })
		hash := domain.Hash(items)
		if exists {
			if r.State == "RESERVED" && hash != oldHash {
				return r, domain.E("IDEMPOTENCY_CONFLICT", 409)
			}
			return r, tx.Commit(ctx)
		}
		if ttl <= 0 || ttl > 900 {
			return r, domain.E("INVALID_TTL", 422)
		}
		if len(items) == 0 || len(items) > 100 {
			return r, domain.E("INVALID_ITEMS", 422)
		}
		previous := ""
		for _, i := range items {
			if i.Quantity <= 0 || i.Quantity > 999 || i.SKU == previous {
				return r, domain.E("INVALID_ITEMS", 422)
			}
			previous = i.SKU
			var available int
			if e = tx.QueryRow(ctx, "SELECT available FROM stock WHERE sku=$1 FOR UPDATE", i.SKU).Scan(&available); e != nil {
				return r, domain.E("INSUFFICIENT_STOCK", 409)
			}
			if available < i.Quantity {
				return r, domain.E("INSUFFICIENT_STOCK", 409)
			}
		}
		for _, i := range items {
			if _, e = tx.Exec(ctx, "UPDATE stock SET available=available-$2 WHERE sku=$1", i.SKU, i.Quantity); e != nil {
				return r, e
			}
		}
		r = domain.Reservation{ID: uuid.NewString(), OrderID: order, State: "RESERVED", ExpiresAt: now.Add(time.Duration(ttl) * time.Second)}
		raw, e = json.Marshal(items)
		if e != nil {
			return r, e
		}
		_, e = tx.Exec(ctx, "INSERT INTO reservations(order_id,id,operation_key,input_hash,items,state,expires_at) VALUES($1,$2,$3,$4,$5,'RESERVED',$6)", order, r.ID, key, hash, raw, r.ExpiresAt)
	case "query":
		if !exists {
			return r, domain.E("RESERVATION_NOT_FOUND", 404)
		}
	case "finalize":
		if !exists {
			return r, domain.E("RESERVATION_NOT_FOUND", 404)
		}
		if r.State == "RESERVED" {
			for _, i := range oldItems {
				if _, e = tx.Exec(ctx, "UPDATE stock SET physical=physical-$2 WHERE sku=$1", i.SKU, i.Quantity); e != nil {
					return r, e
				}
			}
			r.State = "COMMITTED"
			_, e = tx.Exec(ctx, "UPDATE reservations SET state='COMMITTED' WHERE order_id=$1", order)
		}
	case "release":
		if !exists {
			r = domain.Reservation{ID: uuid.NewString(), OrderID: order, State: "RELEASED", ExpiresAt: now}
			_, e = tx.Exec(ctx, "INSERT INTO reservations(order_id,id,operation_key,input_hash,items,state,expires_at) VALUES($1,$2,$3,'','[]','RELEASED',$4)", order, r.ID, key, now)
		} else if r.State == "RESERVED" {
			for _, i := range oldItems {
				if _, e = tx.Exec(ctx, "UPDATE stock SET available=available+$2 WHERE sku=$1", i.SKU, i.Quantity); e != nil {
					return r, e
				}
			}
			r.State = "RELEASED"
			_, e = tx.Exec(ctx, "UPDATE reservations SET state='RELEASED' WHERE order_id=$1", order)
		}
	case "reverse":
		var state string
		e = tx.QueryRow(ctx, "SELECT state FROM fulfillment WHERE order_id=$1", order).Scan(&state)
		if e != nil || state != "CANCELLED" {
			return r, domain.E("FULFILLMENT_NOT_CANCELLED", 409)
		}
		if !cod {
			var data []byte
			if e = tx.QueryRow(ctx, "SELECT payload FROM approvals WHERE id=$1", approval).Scan(&data); e != nil {
				return r, domain.E("INVALID_APPROVAL", 422)
			}
			var a domain.Approval
			if json.Unmarshal(data, &a) != nil || a.OrderID != order || a.Type != "CANCEL" || a.State != "APPROVED" {
				return r, domain.E("INVALID_APPROVAL", 422)
			}
		}
		if r.State == "COMMITTED" {
			for _, i := range oldItems {
				if _, e = tx.Exec(ctx, "UPDATE stock SET physical=physical+$2,available=available+$2 WHERE sku=$1", i.SKU, i.Quantity); e != nil {
					return r, e
				}
			}
			r.State = "RELEASED"
			_, e = tx.Exec(ctx, "UPDATE reservations SET state='RELEASED' WHERE order_id=$1", order)
		}
	default:
		return r, domain.E("UNKNOWN_ACTION", 400)
	}
	if e != nil {
		return r, e
	}
	return r, tx.Commit(ctx)
}
func (s *Simulator) fulfil(ctx context.Context, action, id string, generation int64) (map[string]any, error) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	if _, e = uuid.Parse(id); e != nil {
		return nil, domain.E("INVALID_ORDER_ID", 400)
	}
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1)::bigint)", "task:"+id); e != nil {
		return nil, e
	}
	var state string
	var gen int64
	var task *string
	e = tx.QueryRow(ctx, "SELECT state,generation,task_id FROM fulfillment WHERE order_id=$1 FOR UPDATE", id).Scan(&state, &gen, &task)
	exists := e == nil
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return nil, e
	}
	result := map[string]any{}
	switch action {
	case "authorize":
		if generation <= 0 {
			return nil, domain.E("INVALID_GENERATION", 422)
		}
		if !exists {
			_, e = tx.Exec(ctx, "INSERT INTO fulfillment(order_id,generation,state) VALUES($1,$2,'READY')", id, generation)
			state, gen = "READY", generation
		}
		if exists && state == "READY" && generation > gen {
			_, e = tx.Exec(ctx, "UPDATE fulfillment SET generation=$2 WHERE order_id=$1", id, generation)
			gen = generation
		}
		result["authorized"] = state == "READY" && gen == generation
	case "cancel":
		if !exists {
			_, e = tx.Exec(ctx, "INSERT INTO fulfillment(order_id,generation,state) VALUES($1,0,'CANCELLED')", id)
			state = "CANCELLED"
		} else if state == "READY" {
			_, e = tx.Exec(ctx, "UPDATE fulfillment SET state='CANCELLED' WHERE order_id=$1", id)
			state = "CANCELLED"
		}
		result["cancelled"] = state == "CANCELLED"
	case "accept":
		if s.OrderURL != "" {
			req, err := http.NewRequestWithContext(ctx, "GET", s.OrderURL+"/internal/ready-authorizations/"+id, nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("X-Internal-Token", s.Token)
			resp, err := s.Client.Do(req)
			if err != nil {
				return nil, domain.E("AUTHORIZATION_UNKNOWN", 503)
			}
			var auth struct {
				Authorized bool  `json:"authorized"`
				Generation int64 `json:"generation"`
			}
			err = json.NewDecoder(resp.Body).Decode(&auth)
			resp.Body.Close()
			if err != nil || resp.StatusCode != 200 || !auth.Authorized || auth.Generation != generation {
				return nil, domain.E("READY_NOT_AUTHORIZED", 409)
			}
		}
		if state == "CANCELLED" || !exists || generation != gen {
			return nil, domain.E("READY_NOT_AUTHORIZED", 409)
		}
		if state == "READY" {
			taskID := uuid.NewString()
			task = &taskID
			_, e = tx.Exec(ctx, "UPDATE fulfillment SET state='ACCEPTED',task_id=$2 WHERE order_id=$1", id, taskID)
			if e != nil {
				return nil, e
			}
			_, e = tx.Exec(ctx, "INSERT INTO resources(id,order_id,source,version,generation) VALUES($1,$2,'fulfillment',1,$3)", taskID, id, generation)
		}
		result["task_id"] = task
		result["generation"] = gen
	default:
		return nil, domain.E("INVALID_ACTION", 400)
	}
	if e != nil {
		return nil, e
	}
	return result, tx.Commit(ctx)
}
func (s *Simulator) pay(ctx context.Context, r domain.Receipt) (domain.Receipt, error) {
	if r.TransactionID == "" {
		r.TransactionID = uuid.NewString()
	}
	r.Provider = "SIMULATOR"
	r.Receiver = s.Receiver
	r.Status = "SUCCEEDED"
	r.PaidAt = time.Now().UTC()
	if e := r.Validate(s.Receiver); e != nil {
		return r, e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return r, e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(440035)"); e != nil {
		return r, e
	}
	b, e := json.Marshal(r)
	if e != nil {
		return r, e
	}
	_, e = tx.Exec(ctx, "INSERT INTO provider_payments(transaction_id,reference,payload) VALUES($1,$2,$3) ON CONFLICT DO NOTHING", r.TransactionID, r.Reference, b)
	if e != nil {
		return r, e
	}
	var saved []byte
	if e = tx.QueryRow(ctx, "SELECT payload FROM provider_payments WHERE transaction_id=$1", r.TransactionID).Scan(&saved); e != nil {
		return r, e
	}
	var old domain.Receipt
	if e = json.Unmarshal(saved, &old); e != nil {
		return r, e
	}
	if old.Amount != r.Amount || old.Reference != r.Reference || old.Currency != r.Currency {
		return r, domain.E("IDEMPOTENCY_CONFLICT", 409)
	}
	return old, tx.Commit(ctx)
}
func (s *Simulator) Callback(ctx context.Context, r domain.Receipt) error {
	b, e := json.Marshal(r)
	if e != nil {
		return e
	}
	stamp := strconv.FormatInt(time.Now().Unix(), 10)
	h := hmac.New(sha256.New, []byte(s.CallbackSecret))
	h.Write([]byte(stamp + "."))
	h.Write(b)
	req, e := http.NewRequestWithContext(ctx, "POST", s.OrderURL+"/api/v1/payments/vietqr/callback", bytes.NewReader(b))
	if e != nil {
		return e
	}
	req.Header.Set("X-Callback-Timestamp", stamp)
	req.Header.Set("X-Callback-Signature", hex.EncodeToString(h.Sum(nil)))
	req.Header.Set("Content-Type", "application/json")
	resp, e := s.Client.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return domain.E("CALLBACK_NOT_ACCEPTED", resp.StatusCode)
	}
	return nil
}
func (s *Simulator) refund(ctx context.Context, key, transaction string, amount int64) (map[string]any, error) {
	if amount <= 0 || len(key) < 10 {
		return nil, domain.E("INVALID_REFUND", 422)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1)::bigint)", "refund:"+transaction); e != nil {
		return nil, e
	}
	hash := domain.Hash(map[string]any{"transaction_id": transaction, "amount": amount})
	var oldHash, status, ref string
	e = tx.QueryRow(ctx, "SELECT input_hash,status,reference FROM provider_refunds WHERE operation_key=$1", key).Scan(&oldHash, &status, &ref)
	if e == nil {
		if oldHash != hash {
			return nil, domain.E("IDEMPOTENCY_CONFLICT", 409)
		}
		return map[string]any{"status": status, "reference": ref}, tx.Commit(ctx)
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return nil, e
	}
	var b []byte
	if e = tx.QueryRow(ctx, "SELECT payload FROM provider_payments WHERE transaction_id=$1 FOR UPDATE", transaction).Scan(&b); e != nil {
		return nil, domain.E("PAYMENT_NOT_FOUND", 404)
	}
	var receipt domain.Receipt
	if e = json.Unmarshal(b, &receipt); e != nil {
		return nil, e
	}
	var spent int64
	if e = tx.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM provider_refunds WHERE transaction_id=$1 AND status='SUCCEEDED'", transaction).Scan(&spent); e != nil {
		return nil, e
	}
	if amount > receipt.Amount-spent {
		return nil, domain.E("REFUND_BUDGET_EXCEEDED", 409)
	}
	ref = uuid.NewString()
	status = "SUCCEEDED"
	if s.Fault(ctx, "refund_definitive") == "failed" {
		status = "FAILED"
	}
	_, e = tx.Exec(ctx, "INSERT INTO provider_refunds(operation_key,transaction_id,amount,status,reference,input_hash) VALUES($1,$2,$3,$4,$5,$6)", key, transaction, amount, status, ref, hash)
	if e != nil {
		return nil, e
	}
	return map[string]any{"status": status, "reference": ref}, tx.Commit(ctx)
}
