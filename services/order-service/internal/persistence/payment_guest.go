package persistence

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"math/big"
	"strings"
	"time"
)

func (p *Payment) guestKey() []byte {
	h := hmac.New(sha256.New, p.Store.auditKey)
	h.Write([]byte("guest-payment-grant-v1"))
	return h.Sum(nil)
}
func (p *Payment) guestCredential(token string) string {
	h := hmac.New(sha256.New, p.guestKey())
	h.Write([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func (p *Payment) CreateGuestSession(ctx context.Context, sid, credential string, issued, expires time.Time) error {
	if !validID(sid) || credential == "" || !expires.After(issued) || expires.Sub(issued) > 24*time.Hour {
		return ErrInput
	}
	_, e := p.Store.db.Exec(ctx, "INSERT INTO guest_checkout_sessions(sid,credential_hash,issued_at,expires_at) VALUES($1,$2,$3,$4)", sid, p.guestCredential(credential), issued, expires)
	return e
}
func (p *Payment) RevokeGuestSession(ctx context.Context, sid, credential string) error {
	tag, e := p.Store.db.Exec(ctx, `UPDATE guest_checkout_sessions SET revoked_at=COALESCE(revoked_at,clock_timestamp()) WHERE sid=$1 AND credential_hash=$2 AND expires_at>clock_timestamp()`, sid, p.guestCredential(credential))
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return ErrScope
	}
	return nil
}
func(p *Payment)CheckGuestSession(ctx context.Context,sid string)error{var valid bool;e:=p.Store.db.QueryRow(ctx,"SELECT EXISTS(SELECT 1 FROM guest_checkout_sessions WHERE sid=$1 AND revoked_at IS NULL AND expires_at>clock_timestamp())",sid).Scan(&valid);if e!=nil{return e};if !valid{return ErrScope};return nil}

type guestClaims struct {
	Issuer         string `json:"iss"`
	Audience       string `json:"aud"`
	SID            string `json:"sid"`
	Order          string `json:"order_id"`
	JTI            string `json:"jti"`
	Version        int64  `json:"grant_version"`
	Expires        int64  `json:"exp"`
	SessionExpires int64  `json:"session_exp"`
	Permission     string `json:"scope"`
}
type GuestGrant struct {
	Token          string    `json:"access_token"`
	Type           string    `json:"token_type"`
	Order          string    `json:"order_id"`
	Expires        time.Time `json:"expires_at"`
	SessionExpires time.Time `json:"session_expires_at"`
	Version        int64     `json:"grant_version"`
}

func (p *Payment) signGuest(c guestClaims) (string, error) {
	if p.GrantKey == nil || p.GrantKey.Curve != elliptic.P256() || p.GrantKeyID == "" {
		return "", ErrCapability
	}
	head, _ := json.Marshal(map[string]string{"alg": "ES256", "typ": "JWT", "kid": p.GrantKeyID})
	raw, _ := json.Marshal(c)
	body := base64.RawURLEncoding.EncodeToString(head) + "." + base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(body))
	r, s, e := ecdsa.Sign(rand.Reader, p.GrantKey, digest[:])
	if e != nil {
		return "", e
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])
	return body + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}
func commandKey(key string) bool {
	if len(key) < 16 || len(key) > 128 {
		return false
	}
	for _, r := range key {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}
func (p *Payment) GuestAccess(ctx context.Context, sid, credential, order, key string, revoke bool) (out GuestGrant, err error) {
	if !validID(sid) || !validID(order) || !commandKey(key) {
		return out, ErrInput
	}
	action := "GUEST_PAYMENT_MINT:" + order
	if revoke {
		action = "GUEST_PAYMENT_REVOKE:" + order
	}
	var claim guestClaims
	var token string
	err = p.Store.runner.Do(ctx, func(tx pgx.Tx) error {
		var owner string
		if e := tx.QueryRow(ctx, "SELECT o.scope FROM orders o JOIN payment_intents i ON i.order_id=o.id WHERE o.id=$1 AND o.cohort=$2 AND i.connection_id=$3 AND i.receiver_profile_id=$4 FOR UPDATE OF o", order, Cohort, p.Connection, p.Receiver).Scan(&owner); e != nil {
			return e
		}
		if owner != "guest:"+sid {
			return pgx.ErrNoRows
		}
		var sessionExpiry, now time.Time
		if e := tx.QueryRow(ctx, `SELECT expires_at,clock_timestamp() FROM guest_checkout_sessions WHERE sid=$1 AND credential_hash=$2 AND revoked_at IS NULL AND expires_at>clock_timestamp() FOR SHARE`, sid, p.guestCredential(credential)).Scan(&sessionExpiry, &now); e != nil {
			return e
		}
		var previous []byte
		e := tx.QueryRow(ctx, "SELECT response FROM command_registry WHERE scope=$1 AND kind=$2 AND key=$3", "guest:"+sid, action, key).Scan(&previous)
		if e == nil {
			if revoke {
				return nil
			}
			if json.Unmarshal(previous, &claim) != nil {
				return ErrIntegrity
			}
			if e := p.activeGuest(ctx, tx, claim); e != nil {
				return ErrConflict
			}
			var encrypted []byte
			if e = tx.QueryRow(ctx, "SELECT token_cipher FROM guest_payment_grants WHERE jti=$1", claim.JTI).Scan(&encrypted); e != nil {
				return e
			}
			clear, e := p.Store.cipher.Open("guest-grant:"+claim.JTI, encrypted)
			if e != nil {
				return e
			}
			token = string(clear)
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		var version int64
		if e = tx.QueryRow(ctx, `INSERT INTO guest_payment_grant_versions(sid,order_id,version) VALUES($1,$2,1) ON CONFLICT(sid,order_id) DO UPDATE SET version=guest_payment_grant_versions.version+1 RETURNING version`, sid, order).Scan(&version); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE guest_payment_grants SET revoked_at=COALESCE(revoked_at,clock_timestamp()) WHERE sid=$1 AND order_id=$2", sid, order); e != nil {
			return e
		}
		expires := now.Add(10 * time.Minute)
		if sessionExpiry.Before(expires) {
			expires = sessionExpiry
		}
		expires = expires.Truncate(time.Second)
		claim = guestClaims{"order-service-guest", "order-payment", sid, order, uuid.NewString(), version, expires.Unix(), sessionExpiry.Unix(), "payment:read"}
		if !revoke {
			token, e = p.signGuest(claim)
			if e != nil {
				return e
			}
			encrypted, e := p.Store.cipher.Seal("guest-grant:"+claim.JTI, []byte(token))
			if e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO guest_payment_grants(jti,sid,order_id,grant_version,expires_at,session_expires_at,token_cipher) VALUES($1,$2,$3,$4,$5,$6,$7)`, claim.JTI, sid, order, version, expires, sessionExpiry, encrypted); e != nil {
				return e
			}
		}
		raw, _ := json.Marshal(claim)
		_, e = tx.Exec(ctx, `INSERT INTO command_registry(scope,kind,key,request_hash,order_id,response) VALUES($1,$2,$3,$4,$5,$6)`, "guest:"+sid, action, key, hash([]byte(action)), order, raw)
		return e
	})
	if err != nil {
		return out, err
	}
	if revoke {
		return out, nil
	}
	return GuestGrant{token, "Bearer", order, time.Unix(claim.Expires, 0).UTC(), time.Unix(claim.SessionExpires, 0).UTC(), claim.Version}, nil
}
func (p *Payment) activeGuest(ctx context.Context, q rowReader, c guestClaims) error {
	var active bool
	e := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM guest_payment_grants g JOIN guest_checkout_sessions s ON s.sid=g.sid JOIN guest_payment_grant_versions v ON v.sid=g.sid AND v.order_id=g.order_id JOIN orders o ON o.id=g.order_id WHERE g.jti=$1 AND g.sid=$2 AND g.order_id=$3 AND g.grant_version=$4 AND v.version=g.grant_version AND g.revoked_at IS NULL AND s.revoked_at IS NULL AND g.expires_at>clock_timestamp() AND s.expires_at>clock_timestamp() AND o.scope='guest:'||g.sid::text AND o.cohort=$5)`, c.JTI, c.SID, c.Order, c.Version, Cohort).Scan(&active)
	if e != nil {
		return e
	}
	if !active {
		return ErrReplayExpired
	}
	return nil
}
func (p *Payment) AuthorizeGuest(ctx context.Context, bearer, order string) (string, error) {
	if p.GrantKey == nil || p.GrantKey.Curve != elliptic.P256() || p.GrantKeyID == "" || !strings.HasPrefix(bearer, "Bearer ") {
		return "", ErrScope
	}
	token := strings.TrimPrefix(bearer, "Bearer ")
	parts := strings.Split(token, ".")
	if len(parts) != 3 || len(token) > 8192 {
		return "", ErrScope
	}
	header, e := base64.RawURLEncoding.DecodeString(parts[0])
	if e != nil {
		return "", ErrScope
	}
	var h struct {
		Alg  string `json:"alg"`
		Type string `json:"typ"`
		Key  string `json:"kid"`
	}
	if json.Unmarshal(header, &h) != nil || h.Alg != "ES256" || h.Type != "JWT" || h.Key != p.GrantKeyID {
		return "", ErrScope
	}
	signature, e := base64.RawURLEncoding.DecodeString(parts[2])
	if e != nil || len(signature) != 64 {
		return "", ErrScope
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(&p.GrantKey.PublicKey, digest[:], new(big.Int).SetBytes(signature[:32]), new(big.Int).SetBytes(signature[32:])) {
		return "", ErrScope
	}
	raw, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		return "", ErrScope
	}
	var c guestClaims
	if json.Unmarshal(raw, &c) != nil || c.Issuer != "order-service-guest" || c.Audience != "order-payment" || c.Permission != "payment:read" || c.Order != order || !validID(c.SID) || !validID(c.JTI) || c.Expires <= time.Now().Unix() {
		return "", ErrScope
	}
	if e = p.activeGuest(ctx, p.Store.db, c); e != nil {
		return "", e
	}
	return "guest:" + c.SID, nil
}
