package simulator

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"net/http"
	"strconv"
	"time"
)

func (s *Simulator) Handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if e := s.DB.Ping(r.Context()); e != nil {
			errResponse(w, e)
			return
		}
		response(w, 200, map[string]bool{"ready": true})
	})
	r.Group(func(api chi.Router) {
		api.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Internal-Token") != s.Token {
					errResponse(w, domain.E("UNAUTHENTICATED", 401))
					return
				}
				next.ServeHTTP(w, r)
			})
		})
		api.Post("/catalog/validate", func(w http.ResponseWriter, r *http.Request) {
			var input struct {
				Items []domain.Item `json:"items"`
			}
			if e := read(r, &input); e != nil {
				errResponse(w, domain.E("INVALID_ITEMS", 400))
				return
			}
			tx, e := s.DB.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
			if e != nil {
				errResponse(w, e)
				return
			}
			defer tx.Rollback(r.Context())
			items := []domain.Item{}
			for _, i := range input.Items {
				var active bool
				e := tx.QueryRow(r.Context(), "SELECT price,weight,name,active,version FROM catalog WHERE sku=$1", i.SKU).Scan(&i.UnitPrice, &i.Weight, &i.Name, &active, &i.Version)
				if e != nil || !active {
					errResponse(w, domain.E("SKU_NOT_AVAILABLE", 422))
					return
				}
				items = append(items, i)
			}
			if _, e := domain.Total(items); e != nil {
				errResponse(w, e)
				return
			}
			snapshotID := uuid.NewString()
			var expiry time.Time
			if e = tx.QueryRow(r.Context(), "SELECT clock_timestamp()+interval '15 minutes'").Scan(&expiry); e != nil {
				errResponse(w, e)
				return
			}
			raw, e := json.Marshal(items)
			if e != nil {
				errResponse(w, e)
				return
			}
			if _, e = tx.Exec(r.Context(), "INSERT INTO catalog_quotes(id,snapshot,expires_at) VALUES($1,$2,$3)", snapshotID, raw, expiry); e != nil {
				errResponse(w, e)
				return
			}
			if e = tx.Commit(r.Context()); e != nil {
				errResponse(w, e)
				return
			}
			s.after(r.Context(), w, "catalog", map[string]any{"items": items, "snapshot_id": snapshotID, "expires_at": expiry})
		})
		api.Post("/address/validate", func(w http.ResponseWriter, r *http.Request) {
			var a domain.Address
			if e := read(r, &a); e != nil {
				errResponse(w, domain.E("INVALID_ADDRESS", 400))
				return
			}
			if e := a.Validate(); e != nil {
				errResponse(w, e)
				return
			}
			if a.Province != "75" || a.Ward != "HUE-01" {
				errResponse(w, domain.E("INVALID_ADMINISTRATIVE_UNIT", 422))
				return
			}
			response(w, 200, map[string]bool{"valid": true})
		})
		api.Post("/shipping/quote", func(w http.ResponseWriter, r *http.Request) {
			var input struct {
				Address domain.Address `json:"address"`
				Items   []domain.Item  `json:"items"`
			}
			if e := read(r, &input); e != nil {
				errResponse(w, domain.E("INVALID_QUOTE", 400))
				return
			}
			if e := input.Address.Validate(); e != nil {
				errResponse(w, e)
				return
			}
			s.after(r.Context(), w, "shipping", map[string]any{"quote_reference": uuid.NewString(), "fee": int64(25000), "expires_at": time.Now().UTC().Add(5 * time.Minute), "cod_eligible": true})
		})
		api.Get("/profile/{user}", func(w http.ResponseWriter, r *http.Request) {
			var id string
			if e := s.DB.QueryRow(r.Context(), "SELECT customer_id FROM profiles WHERE user_id=$1", chi.URLParam(r, "user")).Scan(&id); e != nil {
				errResponse(w, domain.E("CUSTOMER_NOT_FOUND", 404))
				return
			}
			response(w, 200, map[string]string{"customer_id": id})
		})
		api.Get("/profile/address/{id}", func(w http.ResponseWriter, r *http.Request) {
			var id string
			if e := s.DB.QueryRow(r.Context(), "SELECT customer_id FROM profiles WHERE address_id=$1 AND customer_id=$2", chi.URLParam(r, "id"), r.URL.Query().Get("customer_id")).Scan(&id); e != nil {
				errResponse(w, domain.E("ADDRESS_NOT_FOUND", 404))
				return
			}
			response(w, 200, map[string]any{"address": domain.Address{Name: "Test customer", Phone: "+84905123456", Street: "Synthetic address", Ward: "HUE-01", Province: "75"}, "version": int64(1)})
		})
		for _, action := range []string{"reserve", "finalize", "release", "reverse"} {
			a := action
			api.Post("/inventory/"+a, func(w http.ResponseWriter, r *http.Request) {
				var in struct {
					Order    string        `json:"order_id"`
					Key      string        `json:"operation_key"`
					Items    []domain.Item `json:"items"`
					TTL      int           `json:"ttl_seconds"`
					Approval string        `json:"approval_id"`
					COD      bool          `json:"cod_unpaid"`
				}
				if e := read(r, &in); e != nil {
					errResponse(w, domain.E("INVALID_REQUEST", 400))
					return
				}
				res, e := s.inventory(r.Context(), a, in.Order, in.Key, in.Items, in.TTL, in.Approval, in.COD)
				if e != nil {
					errResponse(w, e)
					return
				}
				s.after(r.Context(), w, a, res)
			})
		}
		api.Get("/inventory/reservations/{order}", func(w http.ResponseWriter, r *http.Request) {
			res, e := s.inventory(r.Context(), "query", chi.URLParam(r, "order"), "query-reservation", nil, 0, "", false)
			if e != nil {
				errResponse(w, e)
				return
			}
			response(w, 200, res)
		})
		for _, action := range []string{"authorize", "cancel", "accept"} {
			a := action
			api.Post("/fulfillment/"+a, func(w http.ResponseWriter, r *http.Request) {
				var in struct {
					Order      string `json:"order_id"`
					Generation int64  `json:"generation"`
					Key        string `json:"operation_key"`
				}
				if e := read(r, &in); e != nil {
					errResponse(w, domain.E("INVALID_REQUEST", 400))
					return
				}
				res, e := s.fulfil(r.Context(), a, in.Order, in.Generation)
				if e != nil {
					errResponse(w, e)
					return
				}
				s.after(r.Context(), w, "fulfillment_"+a, res)
			})
		}
		api.Get("/resources/{id}", func(w http.ResponseWriter, r *http.Request) {
			var order, source string
			var version, generation int64
			e := s.DB.QueryRow(r.Context(), "SELECT order_id,source,version,generation FROM resources WHERE id=$1", chi.URLParam(r, "id")).Scan(&order, &source, &version, &generation)
			if e != nil {
				errResponse(w, domain.E("RESOURCE_NOT_FOUND", 404))
				return
			}
			response(w, 200, map[string]any{"order_id": order, "source": source, "version": version, "generation": generation})
		})
		api.Get("/care/approvals/{id}", func(w http.ResponseWriter, r *http.Request) {
			var b []byte
			e := s.DB.QueryRow(r.Context(), "SELECT payload FROM approvals WHERE id=$1", chi.URLParam(r, "id")).Scan(&b)
			if e != nil {
				errResponse(w, domain.E("APPROVAL_NOT_FOUND", 404))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(b)
		})
		api.Post("/care/completion-barrier", func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Order string `json:"order_id"`
			}
			if e := read(r, &in); e != nil {
				errResponse(w, domain.E("INVALID_REQUEST", 400))
				return
			}
			ok, e := s.caseBarrier(r.Context(), in.Order)
			if e != nil {
				errResponse(w, e)
				return
			}
			response(w, 200, map[string]bool{"eligible": ok})
		})
		api.Get("/provider/statement", func(w http.ResponseWriter, r *http.Request) {
			if s.Fault(r.Context(), "payment_query") == "unknown" {
				errResponse(w, domain.E("PAYMENT_UNKNOWN", 503))
				return
			}
			after, e := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
			if e != nil || after < 0 {
				errResponse(w, domain.E("INVALID_CURSOR", 400))
				return
			}
			rows, e := s.DB.Query(r.Context(), "SELECT sequence,payload FROM provider_payments WHERE sequence>$1 ORDER BY sequence LIMIT 100", after)
			if e != nil {
				errResponse(w, e)
				return
			}
			defer rows.Close()
			receipts := []domain.Receipt{}
			cursor := after
			for rows.Next() {
				var raw []byte
				if e = rows.Scan(&cursor, &raw); e != nil {
					errResponse(w, e)
					return
				}
				var receipt domain.Receipt
				if e = json.Unmarshal(raw, &receipt); e != nil {
					errResponse(w, e)
					return
				}
				receipts = append(receipts, receipt)
			}
			if e = rows.Err(); e != nil {
				errResponse(w, e)
				return
			}
			response(w, 200, map[string]any{"cursor": cursor, "receipts": receipts})
		})
		api.Get("/provider/payments", func(w http.ResponseWriter, r *http.Request) {
			if s.Fault(r.Context(), "payment_query") == "unknown" {
				errResponse(w, domain.E("PAYMENT_UNKNOWN", 503))
				return
			}
			rows, e := s.DB.Query(r.Context(), "SELECT payload FROM provider_payments WHERE reference=$1 ORDER BY transaction_id", r.URL.Query().Get("reference"))
			if e != nil {
				errResponse(w, e)
				return
			}
			defer rows.Close()
			out := []domain.Receipt{}
			for rows.Next() {
				var b []byte
				var receipt domain.Receipt
				if e = rows.Scan(&b); e != nil {
					errResponse(w, e)
					return
				}
				if e = json.Unmarshal(b, &receipt); e != nil {
					errResponse(w, e)
					return
				}
				out = append(out, receipt)
			}
			response(w, 200, out)
		})
		api.Get("/provider/refunds/{key}", func(w http.ResponseWriter, r *http.Request) {
			if s.Fault(r.Context(), "refund_query") == "unknown" {
				response(w, 200, map[string]string{"status": "UNKNOWN"})
				return
			}
			var status, ref string
			e := s.DB.QueryRow(r.Context(), "SELECT status,reference FROM provider_refunds WHERE operation_key=$1", chi.URLParam(r, "key")).Scan(&status, &ref)
			if errors.Is(e, pgx.ErrNoRows) {
				response(w, 200, map[string]string{"status": "NOT_FOUND"})
				return
			}
			if e != nil {
				errResponse(w, e)
				return
			}
			response(w, 200, map[string]string{"status": status, "reference": ref})
		})
		api.Post("/provider/refunds", func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Key         string `json:"operation_key"`
				Transaction string `json:"transaction_id"`
				Amount      int64  `json:"amount"`
			}
			if e := read(r, &in); e != nil {
				errResponse(w, domain.E("INVALID_REQUEST", 400))
				return
			}
			res, e := s.refund(r.Context(), in.Key, in.Transaction, in.Amount)
			if e != nil {
				errResponse(w, e)
				return
			}
			s.after(r.Context(), w, "refund", res)
		})
		api.Post("/test/token", func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				User      string `json:"user_id"`
				Role      string `json:"role"`
				Warehouse string `json:"warehouse"`
			}
			if e := read(r, &in); e != nil {
				errResponse(w, domain.E("INVALID_REQUEST", 400))
				return
			}
			if in.User == "" {
				in.User = uuid.NewString()
			}
			user, e := uuid.Parse(in.User)
			if e != nil {
				errResponse(w, domain.E("INVALID_USER_ID", 400))
				return
			}
			if in.Role == "" {
				in.Role = "CUSTOMER"
			}
			if in.Warehouse == "" {
				in.Warehouse = "HUE"
			}
			customer := uuid.NewSHA1(uuid.NameSpaceURL, []byte("customer:"+in.User)).String()
			address := uuid.NewSHA1(uuid.NameSpaceURL, []byte("address:"+in.User)).String()
			if _, e = s.DB.Exec(r.Context(), "INSERT INTO profiles(user_id,customer_id,address_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING", user, customer, address); e != nil {
				errResponse(w, e)
				return
			}
			now := time.Now()
			token := app.SignToken(s.JWT, app.Claims{Subject: in.User, Role: in.Role, Warehouse: in.Warehouse, Issuer: "order-sandbox", Audience: "order-api", Issued: now.Unix(), Expires: now.Add(24 * time.Hour).Unix()})
			response(w, 200, map[string]string{"token": token, "user_id": in.User, "customer_id": customer, "address_id": address})
		})
		api.Post("/test/fault", func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Name  string `json:"name"`
				Mode  string `json:"mode"`
				Count int    `json:"count"`
			}
			if e := read(r, &in); e != nil {
				errResponse(w, domain.E("INVALID_REQUEST", 400))
				return
			}
			if e := s.SetFault(r.Context(), in.Name, in.Mode, in.Count); e != nil {
				errResponse(w, e)
				return
			}
			response(w, 200, map[string]bool{"set": true})
		})
		api.Post("/test/catalog", func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				SKU    string `json:"sku_code"`
				Price  int64  `json:"unit_price"`
				Active bool   `json:"active"`
			}
			if e := read(r, &in); e != nil {
				errResponse(w, domain.E("INVALID_REQUEST", 400))
				return
			}
			if in.Price <= 0 {
				errResponse(w, domain.E("INVALID_PRICE", 422))
				return
			}
			if _, e := s.DB.Exec(r.Context(), "UPDATE catalog SET price=$2,active=$3,version=version+1 WHERE sku=$1", in.SKU, in.Price, in.Active); e != nil {
				errResponse(w, e)
				return
			}
			response(w, 200, map[string]bool{"updated": true})
		})
		api.Post("/test/pay", func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Transaction string `json:"transaction_id"`
				Reference   string `json:"reference"`
				Amount      int64  `json:"amount"`
				Currency    string `json:"currency"`
				Callback    bool   `json:"callback"`
			}
			if e := read(r, &in); e != nil {
				errResponse(w, domain.E("INVALID_REQUEST", 400))
				return
			}
			if in.Currency == "" {
				in.Currency = "VND"
			}
			receipt, e := s.pay(r.Context(), domain.Receipt{TransactionID: in.Transaction, Reference: in.Reference, Amount: in.Amount, Currency: in.Currency})
			if e != nil {
				errResponse(w, e)
				return
			}
			if in.Callback && s.OrderURL != "" {
				_ = s.Callback(r.Context(), receipt)
			}
			response(w, 200, receipt)
		})
		api.Post("/test/approval", func(w http.ResponseWriter, r *http.Request) {
			var a domain.Approval
			if e := read(r, &a); e != nil {
				errResponse(w, domain.E("INVALID_REQUEST", 400))
				return
			}
			if a.ID == "" {
				a.ID = uuid.NewString()
			}
			if a.State == "" {
				a.State = "APPROVED"
			}
			if a.Actor == "" {
				a.Actor = "SALES_MANAGER"
			}
			b, e := json.Marshal(a)
			if e != nil {
				errResponse(w, e)
				return
			}
			tag, e := s.DB.Exec(r.Context(), "INSERT INTO approvals(id,payload) VALUES($1,$2) ON CONFLICT DO NOTHING", a.ID, b)
			if e != nil {
				errResponse(w, e)
				return
			}
			if tag.RowsAffected() == 0 {
				var old []byte
				if e = s.DB.QueryRow(r.Context(), "SELECT payload FROM approvals WHERE id=$1", a.ID).Scan(&old); e != nil {
					errResponse(w, e)
					return
				}
				if domain.Hash(string(old)) != domain.Hash(string(b)) {
					errResponse(w, domain.E("IDEMPOTENCY_CONFLICT", 409))
					return
				}
			}
			response(w, 200, a)
		})
		api.Post("/test/event", func(w http.ResponseWriter, r *http.Request) {
			var event domain.Event
			if e := read(r, &event); e != nil {
				errResponse(w, domain.E("INVALID_EVENT", 400))
				return
			}
			event, e := s.event(r.Context(), event)
			event.Signature = domain.EventSignature(event, s.Token)
			if e != nil {
				errResponse(w, e)
				return
			}
			response(w, 200, event)
		})
	})
	return r
}
func (s *Simulator) caseBarrier(ctx context.Context, order string) (bool, error) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1)::bigint)", "care:"+order); e != nil {
		return false, e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO care(order_id) VALUES($1) ON CONFLICT DO NOTHING", order); e != nil {
		return false, e
	}
	var active bool
	if e = tx.QueryRow(ctx, "SELECT active FROM care WHERE order_id=$1 FOR UPDATE", order).Scan(&active); e != nil {
		return false, e
	}
	if !active {
		if _, e = tx.Exec(ctx, "UPDATE care SET finalized=true WHERE order_id=$1", order); e != nil {
			return false, e
		}
	}
	return !active, tx.Commit(ctx)
}
func (s *Simulator) event(ctx context.Context, event domain.Event) (domain.Event, error) {
	if _, e := uuid.Parse(event.OrderID); e != nil {
		return event, domain.E("INVALID_ORDER_ID", 400)
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return event, e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1)::bigint)", "care:"+event.OrderID); e != nil {
		return event, e
	}
	defaultVersion := int64(1)
	switch event.Type {
	case "PACKING_ACCEPTED", "PACKING_SEALED":
		event.Source = "fulfillment"
		var state string
		var task *string
		e = tx.QueryRow(ctx, "SELECT task_id,generation,state FROM fulfillment WHERE order_id=$1", event.OrderID).Scan(&task, &event.Generation, &state)
		if e != nil || task == nil || state != "ACCEPTED" {
			return event, domain.E("TASK_NOT_ACCEPTED", 409)
		}
		event.ResourceID = *task
		if event.Type == "PACKING_SEALED" {
			defaultVersion = 2
			if event.Seal == "" {
				event.Seal = "SYNTHETIC-SEAL"
			}
		}
	case "SHIPMENT_CREATED", "SHIPMENT_DISPATCHED", "SHIPMENT_DELIVERED", "SHIPMENT_FAILED":
		event.Source = "shipping"
		event.ResourceID = uuid.NewSHA1(uuid.NameSpaceOID, []byte("shipment:"+event.OrderID)).String()
		switch event.Type {
		case "SHIPMENT_DISPATCHED":
			defaultVersion = 2
		case "SHIPMENT_DELIVERED", "SHIPMENT_FAILED":
			defaultVersion = 3
		}
	case "CASE_OPENED", "CASE_CLOSED":
		event.Source = "care"
		event.ResourceID = uuid.NewSHA1(uuid.NameSpaceOID, []byte("case:"+event.OrderID)).String()
		if _, e = tx.Exec(ctx, "INSERT INTO care(order_id) VALUES($1) ON CONFLICT DO NOTHING", event.OrderID); e != nil {
			return event, e
		}
		var finalized bool
		if e = tx.QueryRow(ctx, "SELECT finalized FROM care WHERE order_id=$1 FOR UPDATE", event.OrderID).Scan(&finalized); e != nil {
			return event, e
		}
		if finalized && event.Type == "CASE_OPENED" {
			return event, domain.E("COMPLETION_WINDOW_CLOSED", 409)
		}
		if _, e = tx.Exec(ctx, "UPDATE care SET active=$2 WHERE order_id=$1", event.OrderID, event.Type == "CASE_OPENED"); e != nil {
			return event, e
		}
		if event.Type == "CASE_CLOSED" {
			defaultVersion = 2
		}
	default:
		return event, domain.E("INVALID_EVENT_TYPE", 422)
	}
	if event.Version <= 0 {
		event.Version = defaultVersion
	}
	_, e = tx.Exec(ctx, `INSERT INTO resources(id,order_id,source,version,generation) VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO UPDATE SET version=GREATEST(resources.version,EXCLUDED.version)`, event.ResourceID, event.OrderID, event.Source, event.Version, event.Generation)
	if e != nil {
		return event, e
	}
	return event, tx.Commit(ctx)
}
