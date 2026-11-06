package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"net/url"
	"strings"
	"time"
)

type UploadRequest struct {
	Type     string `json:"resource_type"`
	Resource string `json:"resource_id"`
	Media    string `json:"media_type"`
	Size     int64  `json:"byte_size"`
	SHA256   string `json:"sha256"`
	Reason   string `json:"reason"`
}
type UploadTicket struct {
	ID      string            `json:"evidence_id"`
	URL     string            `json:"upload_url"`
	Expires time.Time         `json:"expires_at"`
	Headers map[string]string `json:"required_headers"`
}
type EvidenceView struct {
	ID       string     `json:"evidence_id"`
	Type     string     `json:"resource_type"`
	Resource string     `json:"resource_id"`
	Status   string     `json:"status"`
	SHA256   string     `json:"sha256"`
	URL      *string    `json:"download_url"`
	Expires  *time.Time `json:"expires_at"`
}
type EvidenceVault interface {
	CreateUpload(context.Context, string, string, UploadRequest) (UploadTicket, error)
	Download(context.Context, string, string) (EvidenceView, error)
}

func (p *Payment) evidenceResource(ctx context.Context, actor PaymentActor, kind, id string, write bool) error {
	permission := "payment.evidence.read"
	if write {
		permission = "payment.evidence.write"
	}
	if !actor.allows(p, permission, false) {
		return ErrForbidden
	}
	switch kind {
	case "REFUND":
		v, e := p.refundView(ctx, p.Store.db, id)
		if e != nil {
			return e
		}
		if write && (v.Executor == nil || *v.Executor != actor.ID || (v.Execution != "STARTED" && v.Execution != "EVIDENCE_PENDING" && v.Execution != "UNKNOWN")) {
			return ErrForbidden
		}
		if !write && (v.Executor == nil || *v.Executor != actor.ID) && !actor.Permissions["payment.refund.read"] && !actor.Permissions["payment.refund.verify"] {
			return ErrForbidden
		}
		return nil
	case "RECONCILIATION_CASE":
		v, e := p.caseView(ctx, p.Store.db, id)
		if e != nil {
			return e
		}
		if (v.Assignee == nil || *v.Assignee != actor.ID) && !actor.Permissions["payment.reconciliation.manage"] {
			return ErrForbidden
		}
		return nil
	default:
		return ErrInput
	}
}
func trustedEvidenceURL(value string, expires time.Time, hosts []string) bool {
	u, e := url.Parse(value)
	if e != nil || u.Scheme != "https" || u.User != nil || u.Hostname() == "" || !expires.After(time.Now()) || expires.After(time.Now().Add(10*time.Minute)) {
		return false
	}
	for _, host := range hosts {
		if strings.EqualFold(u.Host, host) {
			return true
		}
	}
	return false
}
func (p *Payment) EvidenceUpload(ctx context.Context, actor PaymentActor, vault EvidenceVault, hosts []string, key string, in UploadRequest) (UploadTicket, error) {
	if !commandKey(key) || !validID(in.Resource) || !validHash(in.SHA256) || in.Size < 1 || in.Size > 10485760 || in.Reason == "" || len(in.Reason) > 1000 {
		return UploadTicket{}, ErrInput
	}
	if in.Media != "application/pdf" && in.Media != "image/png" && in.Media != "image/jpeg" {
		return UploadTicket{}, ErrInput
	}
	if e := p.evidenceResource(ctx, actor, in.Type, in.Resource, true); e != nil {
		return UploadTicket{}, e
	}
	if vault == nil || len(hosts) == 0 {
		return UploadTicket{}, ErrCapability
	}
	raw, _ := json.Marshal(in)
	requestHash := hash(raw)
	scope := "staff:" + actor.ID
	kind := "PAYMENT_EVIDENCE_UPLOAD:" + in.Type + ":" + in.Resource
	var priorHash string
	var prior []byte
	e := p.Store.db.QueryRow(ctx, "SELECT request_hash,response FROM command_registry WHERE scope=$1 AND kind=$2 AND key=$3", scope, kind, key).Scan(&priorHash, &prior)
	if e == nil && priorHash != requestHash {
		return UploadTicket{}, ErrConflict
	}
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return UploadTicket{}, e
	}
	// The vault owns idempotent object-ticket creation; an uncertain HTTP result
	// is recovered by the same principal/action/target/key, never a fresh object.
	stableKey := p.guestCredential(scope + ":" + kind + ":" + key)
	ticket, e := vault.CreateUpload(ctx, actor.ID, stableKey, in)
	if e != nil {
		return UploadTicket{}, e
	}
	if !validID(ticket.ID) || !trustedEvidenceURL(ticket.URL, ticket.Expires, hosts) {
		return UploadTicket{}, ErrIntegrity
	}
	for header := range ticket.Headers {
		if strings.EqualFold(header, "Authorization") || strings.EqualFold(header, "X-Internal-Token") {
			return UploadTicket{}, ErrIntegrity
		}
	}
	metadata, _ := json.Marshal(struct {
		ID      string
		Expires time.Time
	}{ticket.ID, ticket.Expires})
	e = p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", scope+":"+kind+":"+key); e != nil {
			return e
		}
		var oldHash string
		var old []byte
		e := tx.QueryRow(ctx, "SELECT request_hash,response FROM command_registry WHERE scope=$1 AND kind=$2 AND key=$3", scope, kind, key).Scan(&oldHash, &old)
		if e == nil {
			var pinned struct {
				ID      string
				Expires time.Time
			}
			if oldHash != requestHash || json.Unmarshal(old, &pinned) != nil || pinned.ID != ticket.ID || !pinned.Expires.Equal(ticket.Expires) {
				return ErrConflict
			}
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO payment_evidence_uploads(id,resource_type,resource_id,uploader,sha256,media_type,byte_size,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, ticket.ID, in.Type, in.Resource, actor.ID, in.SHA256, in.Media, in.Size, ticket.Expires); e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO command_registry(scope,kind,key,request_hash,order_id,response) VALUES($1,$2,$3,$4,NULL,$5)`, scope, kind, key, requestHash, metadata)
		return e
	})
	if e == nil {
		e = p.evidenceResource(ctx, actor, in.Type, in.Resource, true)
	}
	if ticket.Headers == nil {
		ticket.Headers = map[string]string{}
	}
	return ticket, e
}
func (p *Payment) EvidenceDownload(ctx context.Context, actor PaymentActor, vault EvidenceVault, hosts []string, id string) (EvidenceView, error) {
	if !actor.allows(p, "payment.evidence.read", false) {
		return EvidenceView{}, ErrForbidden
	}
	if !validID(id) {
		return EvidenceView{}, ErrInput
	}
	if vault == nil || len(hosts) == 0 {
		return EvidenceView{}, ErrCapability
	}
	var kind, resource, digest string
	if e := p.Store.db.QueryRow(ctx, "SELECT resource_type,resource_id,sha256 FROM payment_evidence_uploads WHERE id=$1", id).Scan(&kind, &resource, &digest); e != nil {
		return EvidenceView{}, e
	}
	if e := p.evidenceResource(ctx, actor, kind, resource, false); e != nil {
		return EvidenceView{}, e
	}
	view, e := vault.Download(ctx, actor.ID, id)
	if e != nil {
		return view, e
	}
	if view.ID != id || view.Type != kind || view.Resource != resource || view.SHA256 != digest {
		return EvidenceView{}, ErrIntegrity
	}
	switch view.Status {
	case "AWAITING_UPLOAD", "SCANNING", "CLEAN", "REJECTED":
	default:
		return EvidenceView{}, ErrIntegrity
	}
	if view.URL != nil {
		if view.Status != "CLEAN" || view.Expires == nil || !trustedEvidenceURL(*view.URL, *view.Expires, hosts) {
			return EvidenceView{}, ErrIntegrity
		}
	} else {
		view.Expires = nil
	}
	if e = p.evidenceResource(ctx, actor, kind, resource, false); e != nil {
		return EvidenceView{}, e
	}
	return view, nil
}
