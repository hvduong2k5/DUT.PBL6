package transport

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"io"
	"net/http"
)

func mountMVP(api chi.Router, s *app.Service) {
	api.Get("/api/v1/admin/payments/notifications/{id}", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.SePayNotificationReview(r.Context(), principal(r), chi.URLParam(r, "id"))
		if e != nil {
			failure(s, w, e)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		write(w, 200, v)
	})
	api.Post("/api/v1/admin/cancellation-requests/{id}/decision", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Approve *bool  `json:"approve"`
			Reason  string `json:"reason"`
		}
		if e := decode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		if in.Approve == nil {
			failure(s, w, domain.E("DECISION_REQUIRED", 422))
			return
		}
		v, e := s.ReviewCancellation(r.Context(), principal(r), chi.URLParam(r, "id"), *in.Approve, in.Reason)
		if e != nil {
			failure(s, w, e)
			return
		}
		write(w, 200, v)
	})
	api.Post("/api/v1/orders/{id}/payment-checkout", func(w http.ResponseWriter, r *http.Request) {
		form, e := s.PaymentCheckout(r.Context(), principal(r), chi.URLParam(r, "id"))
		if e != nil {
			failure(s, w, e)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		write(w, 200, form)
	})
	api.Get("/api/v1/admin/payments/review", func(w http.ResponseWriter, r *http.Request) {
		items, e := s.PaymentReview(r.Context(), principal(r))
		if e != nil {
			failure(s, w, e)
			return
		}
		write(w, 200, items)
	})
	api.Post("/api/v1/admin/manual-refunds", func(w http.ResponseWriter, r *http.Request) {
		var in app.ManualRefundInput
		if e := decode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		f, e := s.ApproveManualRefund(r.Context(), principal(r), r.Header.Get("X-Idempotency-Key"), in)
		if e != nil {
			failure(s, w, e)
			return
		}
		write(w, 200, f)
	})
	api.Get("/api/v1/admin/manual-refunds/{id}", func(w http.ResponseWriter, r *http.Request) {
		f, e := s.GetManualRefund(r.Context(), principal(r), chi.URLParam(r, "id"))
		if e != nil {
			failure(s, w, e)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		write(w, 200, f)
	})
	api.Post("/api/v1/admin/manual-refunds/{id}/claim", func(w http.ResponseWriter, r *http.Request) {
		f, e := s.ClaimManualRefund(r.Context(), principal(r), chi.URLParam(r, "id"))
		if e != nil {
			failure(s, w, e)
			return
		}
		write(w, 200, f)
	})
	api.Post("/api/v1/admin/manual-refunds/{id}/unknown", func(w http.ResponseWriter, r *http.Request) {
		f, e := s.MarkManualRefundUnknown(r.Context(), principal(r), chi.URLParam(r, "id"))
		if e != nil {
			failure(s, w, e)
			return
		}
		write(w, 200, f)
	})
	api.Post("/api/v1/admin/manual-refunds/{id}/evidence", func(w http.ResponseWriter, r *http.Request) {
		var in app.RefundEvidence
		if e := decode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		f, e := s.SubmitManualRefundEvidence(r.Context(), principal(r), chi.URLParam(r, "id"), in)
		if e != nil {
			failure(s, w, e)
			return
		}
		write(w, 200, f)
	})
	api.Post("/api/v1/admin/manual-refunds/{id}/confirm", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Hash string `json:"evidence_hash"`
		}
		if e := decode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		f, e := s.ConfirmManualRefund(r.Context(), principal(r), chi.URLParam(r, "id"), in.Hash)
		if e != nil {
			failure(s, w, e)
			return
		}
		write(w, 200, f)
	})
}
func sepayIPN(s *app.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.MVP() || s.SePay == nil {
			http.NotFound(w, r)
			return
		}
		if !s.C.SePay.VerifyIPN(r.Header.Get("X-Secret-Key")) {
			failure(s, w, domain.E("INVALID_SEPAY_AUTH", 401))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		raw, e := io.ReadAll(r.Body)
		if e != nil {
			failure(s, w, domain.E("INVALID_SEPAY_NOTIFICATION", 400))
			return
		}
		if !json.Valid(raw) {
			failure(s, w, domain.E("INVALID_SEPAY_NOTIFICATION", 400))
			return
		}
		if e = s.ReceiveSePay(r.Context(), raw); e != nil {
			failure(s, w, e)
			return
		}
		write(w, 200, map[string]bool{"success": true})
	}
}
