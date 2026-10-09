package transport

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }

type principalKey struct{}

func principal(r *http.Request) domain.Principal {
	p, _ := r.Context().Value(principalKey{}).(domain.Principal)
	return p
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if e := decoder.Decode(v); e != nil {
		var max *http.MaxBytesError
		if errors.As(e, &max) {
			return domain.E("BODY_TOO_LARGE", 413)
		}
		return domain.E("INVALID_REQUEST", 400)
	}
	var extra any
	if e := decoder.Decode(&extra); e != io.EOF {
		return domain.E("INVALID_REQUEST", 400)
	}
	return nil
}
func failure(s *app.Service, w http.ResponseWriter, e error) {
	code, status := s.ErrorCode(e)
	write(w, status, map[string]any{"error": map[string]any{"code": code, "retryable": status >= 500}})
}
func version(r *http.Request) (int64, error) {
	v := r.Header.Get("If-Match")
	if v == "" {
		return 0, domain.E("VERSION_REQUIRED", 428)
	}
	n, e := strconv.ParseInt(strings.Trim(v, "\""), 10, 64)
	if e != nil || n <= 0 {
		return 0, domain.E("INVALID_VERSION", 400)
	}
	return n, nil
}
func Handler(s *app.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && r.URL.Path == "/api/v1/checkout" {
				start := time.Now()
				rec := &statusWriter{ResponseWriter: w, status: 200}
				next.ServeHTTP(rec, r)
				s.Metrics.WithLabelValues("checkout_acceptance", strconv.Itoa(rec.status)).Observe(time.Since(start).Seconds())
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	r.Use(middleware.Timeout(3 * time.Second))
	r.Get("/livez", func(w http.ResponseWriter, _ *http.Request) { write(w, 200, map[string]bool{"live": true}) })
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if e := s.DB.Ping(r.Context()); e != nil {
			write(w, 503, map[string]bool{"ready": false})
			return
		}
		write(w, 200, map[string]bool{"ready": true})
	})
	r.Handle("/metrics", promhttp.HandlerFor(s.Registry(), promhttp.HandlerOpts{}))
	r.Post("/api/v1/guest-sessions", func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		now := time.Now()
		token := app.SignToken(s.C.JWTSecret, app.Claims{Subject: id, Role: "GUEST", Issuer: "order-sandbox", Audience: "order-api", Issued: now.Unix(), Expires: now.Add(30 * 24 * time.Hour).Unix()})
		write(w, 201, map[string]string{"session_id": id, "access_token": token})
	})
	r.Group(func(api chi.Router) {
		api.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				p, e := s.Authenticate(r.Header.Get("Authorization"))
				if e != nil {
					failure(s, w, e)
					return
				}
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, p)))
			})
		})
		api.Get("/api/v1/cart", func(w http.ResponseWriter, r *http.Request) {
			c, e := s.Cart(r.Context(), principal(r))
			if e != nil {
				failure(s, w, e)
				return
			}
			w.Header().Set("ETag", strconv.Quote(strconv.FormatInt(c.Revision, 10)))
			write(w, 200, c)
		})
		api.Post("/api/v1/cart/items", func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				SKU      string `json:"sku_code"`
				Qty      int    `json:"quantity"`
				Revision int64  `json:"revision"`
			}
			if e := decode(w, r, &in); e != nil {
				failure(s, w, e)
				return
			}
			if in.Qty <= 0 {
				failure(s, w, domain.E("INVALID_QUANTITY", 422))
				return
			}
			c, e := s.MutateCart(r.Context(), principal(r), in.SKU, in.Qty, in.Revision, true)
			if e != nil {
				failure(s, w, e)
				return
			}
			write(w, 200, c)
		})
		api.Put("/api/v1/cart/items/{item_id}", func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Qty      int   `json:"quantity"`
				Revision int64 `json:"revision"`
			}
			if e := decode(w, r, &in); e != nil {
				failure(s, w, e)
				return
			}
			if in.Qty <= 0 {
				failure(s, w, domain.E("INVALID_QUANTITY", 422))
				return
			}
			c, e := s.MutateCart(r.Context(), principal(r), chi.URLParam(r, "item_id"), in.Qty, in.Revision, false)
			if e != nil {
				failure(s, w, e)
				return
			}
			write(w, 200, c)
		})
		api.Delete("/api/v1/cart/items/{item_id}", func(w http.ResponseWriter, r *http.Request) {
			rev, e := strconv.ParseInt(strings.Trim(r.Header.Get("If-Match"), "\""), 10, 64)
			if e != nil {
				failure(s, w, domain.E("VERSION_REQUIRED", 428))
				return
			}
			_, e = s.MutateCart(r.Context(), principal(r), chi.URLParam(r, "item_id"), 0, rev, false)
			if e != nil {
				failure(s, w, e)
				return
			}
			w.WriteHeader(204)
		})
		api.Post("/api/v1/checkout/quote", func(w http.ResponseWriter, r *http.Request) {
			var in domain.QuoteInput
			if e := decode(w, r, &in); e != nil {
				failure(s, w, e)
				return
			}
			q, e := s.CreateQuote(r.Context(), principal(r), in)
			if e != nil {
				failure(s, w, e)
				return
			}
			write(w, 200, q)
		})
		api.Post("/api/v1/checkout", func(w http.ResponseWriter, r *http.Request) {
			var in domain.CheckoutInput
			if e := decode(w, r, &in); e != nil {
				failure(s, w, e)
				return
			}
			o, status, e := s.Accept(r.Context(), principal(r), r.Header.Get("X-Idempotency-Key"), in)
			if e != nil {
				code, httpStatus := s.ErrorCode(e)
				write(w, httpStatus, map[string]any{"error": map[string]string{"code": code}, "operation_id": o.ID})
				return
			}
			w.Header().Set("Location", "/api/v1/checkout-operations/"+o.ID)
			if o.OrderID != nil {
				w.Header().Set("Link", "</api/v1/orders/"+*o.OrderID+">; rel=\"item\"")
			}
			if status == 202 {
				w.Header().Set("Retry-After", "2")
			}
			write(w, status, o)
		})
		api.Get("/api/v1/checkout-operations/{id}", func(w http.ResponseWriter, r *http.Request) {
			o, e := s.Operation(r.Context(), chi.URLParam(r, "id"), principal(r))
			if e != nil {
				failure(s, w, e)
				return
			}
			if o.OrderID != nil {
				w.Header().Set("Link", "</api/v1/orders/"+*o.OrderID+">; rel=\"item\"")
			}
			write(w, 200, o)
		})
		list := func(staff, queue bool) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := principal(r)
				limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
				items, e := s.ListOrders(r.Context(), p, staff, queue, r.URL.Query().Get("status"), r.URL.Query().Get("cursor"), limit)
				if e != nil {
					failure(s, w, e)
					return
				}
				next := ""
				if len(items) > 0 && !queue {
					next = s.Cursor(items[len(items)-1], p.Scope(), r.URL.Query().Get("status"))
				}
				write(w, 200, map[string]any{"items": items, "next_cursor": next})
			}
		}
		api.Get("/api/v1/orders", list(false, false))
		api.Get("/api/v1/admin/orders", list(true, false))
		api.Get("/api/v1/admin/orders/queue", list(true, true))
		api.Get("/api/v1/orders/{id}", func(w http.ResponseWriter, r *http.Request) {
			o, e := s.GetOrder(r.Context(), chi.URLParam(r, "id"), principal(r), true)
			if e != nil {
				failure(s, w, e)
				return
			}
			w.Header().Set("ETag", strconv.Quote(strconv.FormatInt(o.Version, 10)))
			write(w, 200, o)
		})
		api.Get("/api/v1/orders/{id}/tracking", func(w http.ResponseWriter, r *http.Request) {
			o, e := s.GetOrder(r.Context(), chi.URLParam(r, "id"), principal(r), false)
			if e != nil {
				failure(s, w, e)
				return
			}
			write(w, 200, map[string]any{"shipment_id": o.Shipment, "order_status": o.Status})
		})
		cancel := func(review bool) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				v, e := version(r)
				if e != nil {
					failure(s, w, e)
					return
				}
				var in struct {
					Reason string `json:"reason"`
				}
				if r.ContentLength != 0 {
					if e = decode(w, r, &in); e != nil {
						failure(s, w, e)
						return
					}
				}
				result, e := s.Cancel(r.Context(), principal(r), chi.URLParam(r, "id"), r.Header.Get("X-Idempotency-Key"), v, in.Reason, review)
				if e != nil {
					failure(s, w, e)
					return
				}
				write(w, 202, result)
			}
		}
		api.Post("/api/v1/orders/{id}/cancel", cancel(false))
		api.Post("/api/v1/orders/{id}/cancellation-requests", cancel(true))
		api.Post("/api/v1/admin/orders/{id}/hold", func(w http.ResponseWriter, r *http.Request) {
			v, e := version(r)
			if e == nil {
				var in struct {
					Active *bool  `json:"active"`
					Reason string `json:"reason"`
				}
				if r.ContentLength != 0 {
					e = decode(w, r, &in)
				}
				active := true
				if in.Active != nil {
					active = *in.Active
				}
				if e == nil {
					e = s.SetHold(r.Context(), principal(r), chi.URLParam(r, "id"), v, active, r.Header.Get("X-Idempotency-Key"))
				}
			}
			if e != nil {
				failure(s, w, e)
				return
			}
			write(w, 200, map[string]string{"status": "HELD"})
		})
	})
	r.Post("/api/v1/payments/vietqr/callback", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		raw, e := io.ReadAll(r.Body)
		if e != nil {
			failure(s, w, domain.E("INVALID_CALLBACK", 400))
			return
		}
		stamp := r.Header.Get("X-Callback-Timestamp")
		t, e := strconv.ParseInt(stamp, 10, 64)
		if e != nil || time.Now().Unix()-t > 300 || t-time.Now().Unix() > 60 {
			failure(s, w, domain.E("INVALID_CALLBACK", 401))
			return
		}
		h := hmac.New(sha256.New, []byte(s.C.CallbackSecret))
		h.Write([]byte(stamp + "."))
		h.Write(raw)
		sig, e := hex.DecodeString(r.Header.Get("X-Callback-Signature"))
		if e != nil || !hmac.Equal(sig, h.Sum(nil)) {
			failure(s, w, domain.E("INVALID_CALLBACK", 401))
			return
		}
		var receipt domain.Receipt
		decoder := json.NewDecoder(strings.NewReader(string(raw)))
		decoder.DisallowUnknownFields()
		if e = decoder.Decode(&receipt); e == nil {
			var extra any
			if decoder.Decode(&extra) != io.EOF {
				e = domain.E("INVALID_RECEIPT", 400)
			} else {
				e = s.RecordReceipt(r.Context(), receipt)
			}
		} else {
			e = domain.E("INVALID_RECEIPT", 400)
		}
		if e != nil {
			failure(s, w, e)
			return
		}
		write(w, 200, map[string]bool{"accepted": true})
	})
	r.Route("/internal", func(in chi.Router) {
		in.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !s.Internal(r.Header.Get("X-Internal-Token")) {
					failure(s, w, domain.E("UNAUTHENTICATED", 401))
					return
				}
				next.ServeHTTP(w, r)
			})
		})
		in.Get("/ready-authorizations/{id}", func(w http.ResponseWriter, r *http.Request) {
			o, e := s.GetOrder(r.Context(), chi.URLParam(r, "id"), domain.Principal{Role: "ADMIN"}, false)
			if e != nil {
				failure(s, w, e)
				return
			}
			eligible := o.Ready && !o.Hold && !o.ActiveCase && o.Stock == "COMMITTED" && (o.Status == "PAID" || o.Status == "CONFIRMED_COD") && (o.Method == "COD" || o.Payment == "CONFIRMED" || o.Payment == "PARTIALLY_REFUNDED")
			write(w, 200, map[string]any{"authorized": eligible, "generation": o.Generation})
		})
		in.Post("/events", func(w http.ResponseWriter, r *http.Request) {
			var event domain.Event
			if e := decode(w, r, &event); e != nil {
				failure(s, w, e)
				return
			}
			status, e := s.ProcessEvent(r.Context(), event)
			if e != nil {
				failure(s, w, e)
				return
			}
			write(w, 202, map[string]string{"status": status})
		})
		in.Post("/refunds", func(w http.ResponseWriter, r *http.Request) {
			var v struct {
				Approval string `json:"approval_id"`
			}
			if e := decode(w, r, &v); e != nil {
				failure(s, w, e)
				return
			}
			id, e := s.AcceptRefund(r.Context(), v.Approval)
			if e != nil {
				failure(s, w, e)
				return
			}
			write(w, 202, map[string]string{"refund_id": id})
		})
		in.Post("/cancellations/approve", func(w http.ResponseWriter, r *http.Request) {
			var v struct {
				Approval string `json:"approval_id"`
			}
			if e := decode(w, r, &v); e != nil {
				failure(s, w, e)
				return
			}
			if e := s.ApproveCancellation(r.Context(), v.Approval); e != nil {
				failure(s, w, e)
				return
			}
			write(w, 202, map[string]bool{"accepted": true})
		})
		in.Post("/recovery/{id}", func(w http.ResponseWriter, r *http.Request) {
			p, e := s.Authenticate(r.Header.Get("Authorization"))
			if e == nil {
				e = s.Recover(r.Context(), p, chi.URLParam(r, "id"))
			}
			if e != nil {
				failure(s, w, e)
				return
			}
			write(w, 202, map[string]bool{"accepted": true})
		})
	})
	return r
}
