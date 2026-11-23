package persistence

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// F-42 slice 3: the local evidence vault carries a file from ticket to
// verified storage behind capability-signed URLs, and the wrapper's
// resource rules (executor-only upload, read-tier download) hold.

type vaultContent interface {
	PutContent(context.Context, string, int64, string, io.Reader) error
	GetContent(context.Context, string, int64, string) (string, []byte, error)
}

func ticketParams(t *testing.T, raw string) (int64, string) {
	t.Helper()
	u, e := url.Parse(raw)
	if e != nil {
		t.Fatal(e)
	}
	exp, e := strconv.ParseInt(u.Query().Get("exp"), 10, 64)
	if e != nil {
		t.Fatal(e)
	}
	return exp, u.Query().Get("sig")
}

func TestLocalEvidenceVaultFlow(t *testing.T) {
	h, p, _, tr := paymentFixture(t)
	receipt := f56Receipt(t, h, p, tr)
	reader := p.LocalRefundApproval()
	approver := f56Actor(p, "payment.refund.approve", "payment.refund.accept", "payment.refund.read")
	acceptor := f56Actor(p, "payment.refund.accept", "payment.refund.read")
	executor := f56Actor(p, "payment.refund.execute", "payment.evidence.write", "payment.refund.read")
	issuer := f56Actor(p, "payment.evidence.issue", "payment.refund.verify", "payment.refund.read")
	approval, e := p.CreateRefundApproval(h.ctx, approver, receipt, tr.In, "000012345678", "VCB", "LOCAL TEST", "vault proof refund")
	if e != nil {
		t.Fatal(e)
	}
	refund, e := p.AcceptRefundFromApproval(h.ctx, acceptor, approval.ID, "f57-accept-00000001")
	if e != nil {
		t.Fatal(e)
	}
	claimed, e := p.RefundCommand(h.ctx, executor, reader, refund.ID, "claim", "f57-claim-00000001", refund.Version, "taking it")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := p.RefundCommand(h.ctx, executor, reader, refund.ID, "start", "f57-start-00000001", claimed.Version, "sending"); e != nil {
		t.Fatal(e)
	}

	vault := p.LocalEvidenceVault("https://api-pbl6.hvduong.qzz.io")
	hosts := []string{"api-pbl6.hvduong.qzz.io"}
	cv, ok := vault.(vaultContent)
	if !ok {
		t.Fatal("local vault does not expose content endpoints")
	}
	content := []byte("%PDF-1.4 f57 bank statement bytes")
	req := UploadRequest{Type: "REFUND", Resource: refund.ID, Media: "application/pdf", Size: int64(len(content)), SHA256: hash(content), Reason: "statement for the vault proof"}

	// Only the executor may upload evidence for the refund.
	if _, e := p.EvidenceUpload(h.ctx, issuer, vault, hosts, "f57-upload-issuer-01", req); !errors.Is(e, ErrForbidden) {
		t.Fatal("non-executor uploaded refund evidence", e)
	}
	ticket, e := p.EvidenceUpload(h.ctx, executor, vault, hosts, "f57-upload-00000001", req)
	if e != nil {
		t.Fatal(e)
	}
	if !validID(ticket.ID) || !strings.Contains(ticket.URL, "/evidence-content/"+ticket.ID) {
		t.Fatal(ticket)
	}

	// Ticket creation is idempotent on the stable key, and a conflicting
	// replay is refused.
	stable := p.guestCredential("staff:" + executor.ID + ":PAYMENT_EVIDENCE_UPLOAD:REFUND:" + refund.ID + ":f57-upload-00000001")
	again, e := vault.CreateUpload(h.ctx, executor.ID, stable, req)
	if e != nil || again.ID != ticket.ID || !again.Expires.Equal(ticket.Expires) {
		t.Fatal(again, e)
	}
	conflict := req
	conflict.SHA256 = hash([]byte("other bytes"))
	if _, e := vault.CreateUpload(h.ctx, executor.ID, stable, conflict); !errors.Is(e, ErrConflict) {
		t.Fatal("conflicting ticket replay accepted", e)
	}

	// A bad signature moves nothing; the right bytes verify CLEAN, and a
	// repeat PUT of the same bytes is a no-op.
	exp, sig := ticketParams(t, ticket.URL)
	if e := cv.PutContent(h.ctx, ticket.ID, exp, "deadbeef", bytes.NewReader(content)); !errors.Is(e, ErrForbidden) {
		t.Fatal("unsigned content accepted", e)
	}
	if e := cv.PutContent(h.ctx, ticket.ID, exp, sig, bytes.NewReader(content)); e != nil {
		t.Fatal(e)
	}
	if e := cv.PutContent(h.ctx, ticket.ID, exp, sig, bytes.NewReader(content)); e != nil {
		t.Fatal("idempotent re-PUT failed", e)
	}

	// The read tier can describe the object and mint a download; the
	// bytes round-trip unchanged.
	viewer := f56Actor(p, "payment.refund.read", "payment.evidence.read")
	view, e := p.EvidenceDownload(h.ctx, viewer, vault, hosts, ticket.ID)
	if e != nil {
		t.Fatal(e)
	}
	if view.Status != "CLEAN" || view.URL == nil || view.SHA256 != req.SHA256 {
		t.Fatal(view)
	}
	gexp, gsig := ticketParams(t, *view.URL)
	media, raw, e := cv.GetContent(h.ctx, ticket.ID, gexp, gsig)
	if e != nil || media != "application/pdf" || !bytes.Equal(raw, content) {
		t.Fatal(media, raw, e)
	}

	// Bytes that do not match the ticket are rejected, permanently.
	other := []byte("%PDF-1.4 different bytes entirely")
	badReq := UploadRequest{Type: "REFUND", Resource: refund.ID, Media: "application/pdf", Size: int64(len(other)), SHA256: hash(other), Reason: "mismatched upload"}
	badTicket, e := p.EvidenceUpload(h.ctx, executor, vault, hosts, "f57-upload-00000002", badReq)
	if e != nil {
		t.Fatal(e)
	}
	bexp, bsig := ticketParams(t, badTicket.URL)
	if e := cv.PutContent(h.ctx, badTicket.ID, bexp, bsig, bytes.NewReader(content)); !errors.Is(e, ErrInput) {
		t.Fatal("mismatched bytes accepted", e)
	}
	badView, e := p.EvidenceDownload(h.ctx, viewer, vault, hosts, badTicket.ID)
	if e != nil || badView.Status != "REJECTED" || badView.URL != nil {
		t.Fatal(badView, e)
	}
}
