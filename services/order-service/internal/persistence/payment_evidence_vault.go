package persistence

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// F-42 slice 3: the local evidence vault. Evidence files (bank statements,
// receipts) are stored in this service's own database behind the existing
// EvidenceVault contract: CreateUpload tickets an object and returns a
// short-lived capability-signed PUT URL served by this same service; the
// bytes are accepted only when their length and SHA-256 match the ticket
// exactly. Download mints a fresh capability-signed GET URL, and only for
// objects whose bytes verified CLEAN. The signing key derives from the
// payment webhook secret, so rotating that secret invalidates outstanding
// tickets (they live at most ten minutes).

const evidenceVaultMaxBytes = 10485760

type localEvidenceVault struct {
	p    *Payment
	base string
}

func (p *Payment) LocalEvidenceVault(base string) EvidenceVault {
	return &localEvidenceVault{p: p, base: strings.TrimRight(base, "/")}
}

func (v *localEvidenceVault) secret() []byte {
	sum := sha256.Sum256([]byte("evidence-vault-v1:" + v.p.Secret))
	return sum[:]
}

func (v *localEvidenceVault) sign(method, id string, exp time.Time) string {
	mac := hmac.New(sha256.New, v.secret())
	fmt.Fprintf(mac, "%s\n%s\n%d", method, id, exp.Unix())
	return hex.EncodeToString(mac.Sum(nil))
}

func (v *localEvidenceVault) contentURL(method, id string, exp time.Time) string {
	return fmt.Sprintf("%s/api/v1/admin/payments/evidence-content/%s?exp=%d&sig=%s", v.base, id, exp.Unix(), v.sign(method, id, exp))
}

func (v *localEvidenceVault) verify(method, id string, expUnix int64, sig string) error {
	exp := time.Unix(expUnix, 0)
	if !exp.After(time.Now()) {
		return ErrForbidden
	}
	if subtle.ConstantTimeCompare([]byte(v.sign(method, id, exp)), []byte(sig)) != 1 {
		return ErrForbidden
	}
	return nil
}

func (v *localEvidenceVault) CreateUpload(ctx context.Context, actor, stableKey string, in UploadRequest) (UploadTicket, error) {
	if stableKey == "" || len(stableKey) > 256 {
		return UploadTicket{}, ErrInput
	}
	expires := time.Now().Add(10 * time.Minute).UTC().Truncate(time.Second)
	var id string
	e := v.p.Store.db.QueryRow(ctx, `INSERT INTO payment_evidence_objects(evidence_id,stable_key,resource_type,resource_id,uploader,sha256,media_type,byte_size,ticket_expires_at)
		VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (stable_key) DO NOTHING
		RETURNING evidence_id::text`, stableKey, in.Type, in.Resource, actor, in.SHA256, in.Media, in.Size, expires).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		// Idempotent replay of the same stable key: the pinned ticket is
		// returned unchanged, and only for the identical request.
		var sha, media, rtype, rsrc string
		e = v.p.Store.db.QueryRow(ctx, `SELECT evidence_id::text,ticket_expires_at,sha256,media_type,resource_type,resource_id::text FROM payment_evidence_objects WHERE stable_key=$1`, stableKey).Scan(&id, &expires, &sha, &media, &rtype, &rsrc)
		if e != nil {
			return UploadTicket{}, e
		}
		if sha != in.SHA256 || media != in.Media || rtype != in.Type || rsrc != in.Resource {
			return UploadTicket{}, ErrConflict
		}
	} else if e != nil {
		return UploadTicket{}, e
	}
	return UploadTicket{ID: id, URL: v.contentURL("PUT", id, expires), Expires: expires, Headers: map[string]string{"Content-Type": in.Media}}, nil
}

func (v *localEvidenceVault) Download(ctx context.Context, actor, id string) (EvidenceView, error) {
	var rtype, rsrc, sha, state string
	e := v.p.Store.db.QueryRow(ctx, `SELECT resource_type,resource_id::text,sha256,state FROM payment_evidence_objects WHERE evidence_id=$1`, id).Scan(&rtype, &rsrc, &sha, &state)
	if errors.Is(e, pgx.ErrNoRows) {
		return EvidenceView{}, ErrCapability
	}
	if e != nil {
		return EvidenceView{}, e
	}
	view := EvidenceView{ID: id, Type: rtype, Resource: rsrc, Status: state, SHA256: sha}
	if state == "CLEAN" {
		exp := time.Now().Add(5 * time.Minute).UTC().Truncate(time.Second)
		url := v.contentURL("GET", id, exp)
		view.URL = &url
		view.Expires = &exp
	}
	return view, nil
}

// PutContent stores the bytes behind an upload ticket. It is reached only
// through the capability-signed URL CreateUpload minted; the transport
// layer performs no staff authentication on it by design.
func (v *localEvidenceVault) PutContent(ctx context.Context, id string, expUnix int64, sig string, body io.Reader) error {
	if e := v.verify("PUT", id, expUnix, sig); e != nil {
		return e
	}
	raw, e := io.ReadAll(io.LimitReader(body, evidenceVaultMaxBytes+1))
	if e != nil || len(raw) > evidenceVaultMaxBytes {
		return ErrInput
	}
	// The state transitions are single statements on purpose: marking an
	// object REJECTED must commit even though the PUT itself fails, which
	// a surrounding transaction would roll back.
	var size int64
	var want, state string
	e = v.p.Store.db.QueryRow(ctx, `SELECT byte_size,sha256,state FROM payment_evidence_objects WHERE evidence_id=$1`, id).Scan(&size, &want, &state)
	if errors.Is(e, pgx.ErrNoRows) {
		return ErrCapability
	}
	if e != nil {
		return e
	}
	if state == "CLEAN" {
		var stored []byte
		if e := v.p.Store.db.QueryRow(ctx, `SELECT content FROM payment_evidence_objects WHERE evidence_id=$1`, id).Scan(&stored); e != nil {
			return e
		}
		if int64(len(raw)) == size && hash(raw) == want && string(stored) == string(raw) {
			return nil
		}
		return ErrConflict
	}
	if state == "REJECTED" {
		return ErrInput
	}
	if int64(len(raw)) != size || hash(raw) != want {
		if _, e := v.p.Store.db.Exec(ctx, `UPDATE payment_evidence_objects SET state='REJECTED' WHERE evidence_id=$1 AND state='AWAITING_UPLOAD'`, id); e != nil {
			return e
		}
		return ErrInput
	}
	_, e = v.p.Store.db.Exec(ctx, `UPDATE payment_evidence_objects SET content=$2,state='CLEAN',uploaded_at=clock_timestamp() WHERE evidence_id=$1 AND state='AWAITING_UPLOAD'`, id, raw)
	return e
}

// GetContent returns the stored bytes behind a download ticket, again only
// through the capability-signed URL Download minted.
func (v *localEvidenceVault) GetContent(ctx context.Context, id string, expUnix int64, sig string) (string, []byte, error) {
	if e := v.verify("GET", id, expUnix, sig); e != nil {
		return "", nil, e
	}
	var media, state string
	var content []byte
	e := v.p.Store.db.QueryRow(ctx, `SELECT media_type,state,content FROM payment_evidence_objects WHERE evidence_id=$1`, id).Scan(&media, &state, &content)
	if errors.Is(e, pgx.ErrNoRows) {
		return "", nil, ErrCapability
	}
	if e != nil {
		return "", nil, e
	}
	if state != "CLEAN" || content == nil {
		return "", nil, ErrCapability
	}
	return media, content, nil
}
