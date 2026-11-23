package transport

import (
	"bytes"
	"github.com/go-chi/chi/v5"
	"github.com/omamx/order-service/internal/app"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/internal/sepay"
	"io"
	"net/http"
	"strconv"
)

func paymentDecode(w http.ResponseWriter, r *http.Request, out any) error {
	raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 65536))
	if e != nil {
		return domain.E("INVALID_REQUEST", 400)
	}
	if sepay.Decode(raw, out) != nil {
		return domain.E("INVALID_REQUEST", 400)
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	return decode(w, r, out)
}
func paymentStaff(s *app.Service, w http.ResponseWriter, r *http.Request) (persistence.PaymentActor, bool) {
	w.Header().Set("Cache-Control", "no-store")
	if s.Payment == nil {
		failure(s, w, domain.E("PAYMENT_UNAVAILABLE", 503))
		return persistence.PaymentActor{}, false
	}
	actor, e := s.PaymentStaff(r.Header.Get("Authorization"))
	if e != nil {
		sepayError(s, w, e)
		return actor, false
	}
	return actor, true
}
func sepayStaffRoutes(s *app.Service, r chi.Router) {
	r.Post("/internal/payments/refunds", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !s.Internal(r.Header.Get("X-Internal-Token")) {
			failure(s, w, domain.E("UNAUTHENTICATED", 401))
			return
		}
		if s.Payment == nil {
			failure(s, w, domain.E("PAYMENT_UNAVAILABLE", 503))
			return
		}
		var in struct {
			Approval string `json:"approval_id"`
		}
		if e := paymentDecode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		view, e := s.Payment.AcceptManualRefund(r.Context(), s.PaymentApproval, in.Approval, r.Header.Get("X-Idempotency-Key"))
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 201, view)
	})
	r.Get("/api/v1/admin/payments/refunds", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		limit := 25
		if r.URL.Query().Get("limit") != "" {
			limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
		}
		page, e := s.Payment.RefundQueue(r.Context(), actor, r.URL.Query().Get("status"), r.URL.Query().Get("cursor"), limit)
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 200, page)
	})
	r.Get("/api/v1/admin/payments/refunds/{refund_id}", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		view, e := s.Payment.ReadRefund(r.Context(), actor, chi.URLParam(r, "refund_id"))
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 200, view)
	})
	for _, action := range []string{"claim", "start", "report-uncertain", "retry-attempt", "cancel-obligation"} {
		action := action
		r.Post("/api/v1/admin/payments/refunds/{refund_id}/"+action, func(w http.ResponseWriter, r *http.Request) {
			actor, ok := paymentStaff(s, w, r)
			if !ok {
				return
			}
			var in struct {
				Version  int64  `json:"expected_version"`
				Reason   string `json:"reason"`
				Decision string `json:"care_decision_reference,omitempty"`
			}
			if e := paymentDecode(w, r, &in); e != nil {
				failure(s, w, e)
				return
			}
			if in.Version < 1 || len(in.Reason) > 1000 {
				sepayError(s, w, persistence.ErrInput)
				return
			}
			id := chi.URLParam(r, "refund_id")
			var refs []string
			if action == "cancel-obligation" {
				if in.Decision == "" {
					sepayError(s, w, persistence.ErrInput)
					return
				}
				refs = []string{in.Decision}
			} else if in.Decision != "" {
				sepayError(s, w, persistence.ErrInput)
				return
			}
			view, e := s.Payment.RefundCommand(r.Context(), actor, s.PaymentApproval, id, action, r.Header.Get("X-Idempotency-Key"), in.Version, in.Reason, refs...)
			if e != nil {
				sepayError(s, w, e)
				return
			}
			if action == "start" {
				instruction, e := s.Payment.RefundStartInstruction(r.Context(), actor, s.PaymentApproval, id)
				if e != nil {
					sepayError(s, w, e)
					return
				}
				write(w, 200, instruction)
				return
			}
			write(w, 200, view)
		})
	}
	r.Post("/api/v1/admin/payments/refunds/{refund_id}/evidence", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		var in struct {
			Version  int64    `json:"expected_version"`
			Reason   string   `json:"reason"`
			Attempt  string   `json:"attempt_id"`
			Evidence []string `json:"evidence_ids"`
		}
		if e := paymentDecode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		if len(in.Evidence) < 1 || len(in.Evidence) > 100 || in.Reason == "" || !sepay.ValidUUID(in.Attempt) {
			sepayError(s, w, persistence.ErrInput)
			return
		}
		id := chi.URLParam(r, "refund_id")
		e := s.Payment.CheckRefundAttempt(r.Context(), actor, id, in.Attempt)
		if e != nil {
			sepayError(s, w, e)
			return
		}
		view, e := s.Payment.AttachRefundEvidenceBatch(r.Context(), actor, s.PaymentEvidence, id, in.Attempt, r.Header.Get("X-Idempotency-Key"), in.Version, in.Reason, in.Evidence)
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 200, view)
	})
	r.Post("/api/v1/admin/payments/refunds/{refund_id}/verify", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		var in struct {
			Version      int64                           `json:"expected_version"`
			Reason       string                          `json:"reason"`
			Attempt      string                          `json:"attempt_id"`
			Outcome      string                          `json:"outcome"`
			Evidence     []string                        `json:"evidence_ids"`
			BankEvidence *persistence.RefundBankEvidence `json:"bank_evidence"`
		}
		if e := paymentDecode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		if len(in.Evidence) < 1 || len(in.Evidence) > 100 || in.Reason == "" {
			sepayError(s, w, persistence.ErrInput)
			return
		}
		id := chi.URLParam(r, "refund_id")
		if e := s.Payment.CheckRefundAttempt(r.Context(), actor, id, in.Attempt); e != nil {
			sepayError(s, w, e)
			return
		}
		view, e := s.Payment.VerifyRefundEvidenceSet(r.Context(), actor, s.PaymentEvidence, id, in.Attempt, in.Outcome, r.Header.Get("X-Idempotency-Key"), in.Version, in.Reason, in.Evidence, in.BankEvidence)
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 200, view)
	})
	r.Get("/api/v1/admin/payments/reconciliation-cases", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		limit := 25
		if r.URL.Query().Get("limit") != "" {
			limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
		}
		page, e := s.Payment.CaseQueue(r.Context(), actor, r.URL.Query().Get("status"), r.URL.Query().Get("cursor"), limit)
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 200, page)
	})
	r.Get("/api/v1/admin/payments/reconciliation-cases/{case_id}", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		view, e := s.Payment.ReadCase(r.Context(), actor, chi.URLParam(r, "case_id"))
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 200, view)
	})
	r.Post("/api/v1/admin/payments/reconciliation-cases/{case_id}/assign", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		var in struct {
			Version  int64  `json:"expected_version"`
			Reason   string `json:"reason"`
			Assignee string `json:"assignee_id"`
		}
		if e := paymentDecode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		view, e := s.Payment.CaseCommand(r.Context(), actor, s.PaymentCases, chi.URLParam(r, "case_id"), "assign", r.Header.Get("X-Idempotency-Key"), in.Version, in.Reason, in.Assignee, "", "", nil)
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 200, view)
	})
	r.Post("/api/v1/admin/payments/reconciliation-cases/{case_id}/resolve", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		var in struct {
			Version    int64    `json:"expected_version"`
			Reason     string   `json:"reason"`
			Resolution string   `json:"resolution"`
			Decision   string   `json:"decision_reference"`
			Evidence   []string `json:"evidence_ids"`
		}
		if e := paymentDecode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		view, e := s.Payment.CaseCommand(r.Context(), actor, s.PaymentCases, chi.URLParam(r, "case_id"), "resolve", r.Header.Get("X-Idempotency-Key"), in.Version, in.Reason, "", in.Decision, in.Resolution, in.Evidence)
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 200, view)
	})
	r.Post("/api/v1/admin/payments/reconciliation-cases/{case_id}/decisions", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		var in struct {
			Reason     string   `json:"reason"`
			Resolution string   `json:"resolution"`
			Evidence   []string `json:"evidence_ids"`
		}
		if e := paymentDecode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		view, e := s.Payment.CreateCaseDecision(r.Context(), actor, chi.URLParam(r, "case_id"), in.Resolution, in.Reason, in.Evidence)
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 201, view)
	})
	r.Get("/api/v1/admin/payments/reconciliation-cases/{case_id}/decisions", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		items, e := s.Payment.ListCaseDecisions(r.Context(), actor, chi.URLParam(r, "case_id"))
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 200, map[string]any{"items": items})
	})
	r.Post("/api/v1/admin/payments/evidence-uploads", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		var in persistence.UploadRequest
		if e := paymentDecode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		ticket, e := s.Payment.EvidenceUpload(r.Context(), actor, s.PaymentVault, s.PaymentEvidenceHosts, r.Header.Get("X-Idempotency-Key"), in)
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 201, ticket)
	})
	r.Get("/api/v1/admin/payments/evidence/{evidence_id}", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		view, e := s.Payment.EvidenceDownload(r.Context(), actor, s.PaymentVault, s.PaymentEvidenceHosts, chi.URLParam(r, "evidence_id"))
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 200, view)
	})
	// F-42 slice 2: local refund authority endpoints. Approvals are
	// created and withdrawn by approve-tier actors and accepted into
	// refund obligations by a different accept-tier actor; bank-statement
	// certificates are issued by verify-tier actors, attached by the
	// executor and settled against by the verifier.
	r.Post("/api/v1/admin/payments/refund-approvals", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		var in struct {
			Receipt     string `json:"receipt_id"`
			Amount      int64  `json:"amount"`
			Beneficiary string `json:"beneficiary_account"`
			Bank        string `json:"beneficiary_bank"`
			Holder      string `json:"beneficiary_holder"`
			Reason      string `json:"reason"`
		}
		if e := paymentDecode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		view, e := s.Payment.CreateRefundApproval(r.Context(), actor, in.Receipt, in.Amount, in.Beneficiary, in.Bank, in.Holder, in.Reason)
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 201, view)
	})
	r.Get("/api/v1/admin/payments/refund-approvals", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		items, e := s.Payment.ListRefundApprovals(r.Context(), actor, r.URL.Query().Get("receipt_id"))
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 200, map[string]any{"items": items})
	})
	r.Post("/api/v1/admin/payments/refund-approvals/{approval_id}/withdraw", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		var in struct {
			Reference string `json:"withdrawal_reference"`
			Reason    string `json:"reason"`
		}
		if e := paymentDecode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		view, e := s.Payment.WithdrawRefundApproval(r.Context(), actor, chi.URLParam(r, "approval_id"), in.Reference, in.Reason)
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 200, view)
	})
	r.Post("/api/v1/admin/payments/refund-approvals/{approval_id}/accept", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		view, e := s.Payment.AcceptRefundFromApproval(r.Context(), actor, chi.URLParam(r, "approval_id"), r.Header.Get("X-Idempotency-Key"))
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 201, view)
	})
	r.Post("/api/v1/admin/payments/refund-evidence", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		var in persistence.RefundEvidenceInput
		if e := paymentDecode(w, r, &in); e != nil {
			failure(s, w, e)
			return
		}
		record, e := s.Payment.CreateRefundEvidenceRecord(r.Context(), actor, in)
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 201, record)
	})
	r.Get("/api/v1/admin/payments/refund-evidence", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := paymentStaff(s, w, r)
		if !ok {
			return
		}
		items, e := s.Payment.ListRefundEvidenceRecords(r.Context(), actor, r.URL.Query().Get("refund_id"))
		if e != nil {
			sepayError(s, w, e)
			return
		}
		write(w, 200, map[string]any{"items": items})
	})
}
