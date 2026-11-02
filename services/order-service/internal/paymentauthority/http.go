// Package paymentauthority reads decisions from authenticated owning services.
// It deliberately has no fallback to caller-supplied approvals or bank evidence.
package paymentauthority

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/omamx/order-service/internal/persistence"
	"github.com/omamx/order-service/internal/sepay"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTP struct {
	base, token string
	client      *http.Client
}

func New(base, token string) (*HTTP, error) {
	u, e := url.Parse(base)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || len(token) < 16 {
		return nil, errors.New("invalid authority configuration")
	}
	return &HTTP{strings.TrimRight(base, "/"), token, &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (h *HTTP) get(ctx context.Context, path, ref string, out any) error {
	if ref == "" || len(ref) > 256 {
		return persistence.ErrInput
	}
	request, e := http.NewRequestWithContext(ctx, "GET", h.base+path+url.PathEscape(ref), nil)
	if e != nil {
		return persistence.ErrCapability
	}
	request.Header.Set("X-Internal-Token", h.token)
	request.Header.Set("Accept", "application/json")
	response, e := h.client.Do(request)
	if e != nil {
		return persistence.ErrCapability
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return persistence.ErrCapability
	}
	raw, e := io.ReadAll(io.LimitReader(response.Body, 65537))
	if e != nil || len(raw) > 65536 {
		return persistence.ErrCapability
	}
	if sepay.Decode(raw, out) != nil {
		return persistence.ErrCapability
	}
	return nil
}
func (h *HTTP) RefundApproval(ctx context.Context, ref string) (out persistence.RefundApproval, e error) {
	e = h.get(ctx, "/internal/payments/refund-approvals/", ref, &out)
	return
}
func (h *HTTP) RefundEvidence(ctx context.Context, ref string) (out persistence.RefundEvidence, e error) {
	e = h.get(ctx, "/internal/payments/refund-evidence/", ref, &out)
	return
}
func (h *HTTP) CaseDecision(ctx context.Context, ref string) (out persistence.CaseDecision, e error) {
	e = h.get(ctx, "/internal/payments/reconciliation-decisions/", ref, &out)
	return
}
func (h *HTTP) CanAssign(ctx context.Context, receiver, actor string) (bool, error) {
	var out struct {
		Receiver string `json:"receiver_profile_id"`
		Actor    string `json:"actor_id"`
		Allowed  bool   `json:"allowed"`
	}
	e := h.get(ctx, "/internal/payments/case-assignees/"+url.PathEscape(receiver)+"/", actor, &out)
	return e == nil && out.Allowed && out.Receiver == receiver && out.Actor == actor, e
}
func (h *HTTP) CreateUpload(ctx context.Context, actor, key string, in persistence.UploadRequest) (out persistence.UploadTicket, e error) {
	raw, _ := json.Marshal(map[string]any{"actor_id": actor, "request": in})
	request, e := http.NewRequestWithContext(ctx, "POST", h.base+"/internal/payments/evidence-uploads", bytes.NewReader(raw))
	if e != nil {
		return out, persistence.ErrCapability
	}
	request.Header.Set("X-Internal-Token", h.token)
	request.Header.Set("X-Idempotency-Key", key)
	request.Header.Set("Content-Type", "application/json")
	response, e := h.client.Do(request)
	if e != nil {
		return out, persistence.ErrCapability
	}
	defer response.Body.Close()
	if response.StatusCode != 200 && response.StatusCode != 201 {
		return out, persistence.ErrCapability
	}
	raw, e = io.ReadAll(io.LimitReader(response.Body, 65537))
	if e != nil || len(raw) > 65536 || sepay.Decode(raw, &out) != nil {
		return out, persistence.ErrCapability
	}
	return out, nil
}
func (h *HTTP) Download(ctx context.Context, actor, id string) (out persistence.EvidenceView, e error) {
	e = h.get(ctx, "/internal/payments/evidence/"+url.PathEscape(actor)+"/", id, &out)
	return
}
