package persistence

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestGuestPaymentGrantAuthorityReplayAndRevocation(t *testing.T) {
	h := database(t, false)
	_, place := prepared(t, h, 2)
	a, e := h.store.Place(h.ctx, place)
	if e != nil {
		t.Fatal(e)
	}
	grantKey, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	p := &Payment{Store: h.store, GrantKey: grantKey, GrantKeyID: "test-key-v1"}
	if e = h.db.QueryRow(h.ctx, "SELECT connection_id,receiver_profile_id FROM payment_intents WHERE order_id=$1", a.ID()).Scan(&p.Connection, &p.Receiver); e != nil {
		t.Fatal(e)
	}
	sid := a.Owner().PrincipalID
	credential := "verified-guest-checkout-session-credential"
	now := time.Now()
	if e = p.CreateGuestSession(h.ctx, sid, credential, now, now.Add(24*time.Hour)); e != nil {
		t.Fatal(e)
	}
	firstKey := uuid.NewString()
	grant, e := p.GuestAccess(h.ctx, sid, credential, a.ID(), firstKey, false)
	if e != nil {
		t.Fatal(e)
	}
	again, e := p.GuestAccess(h.ctx, sid, credential, a.ID(), firstKey, false)
	if e != nil || again.Token != grant.Token || !again.Expires.Equal(grant.Expires) {
		t.Fatal("replay extended/replaced token", e)
	}
	scope, e := p.AuthorizeGuest(h.ctx, "Bearer "+grant.Token, a.ID())
	if e != nil || scope != ownerScope(a.Owner()) {
		t.Fatal(scope, e)
	}
	if _, e = p.AuthorizeGuest(h.ctx, "Bearer "+grant.Token, uuid.NewString()); e == nil {
		t.Fatal("cross-order grant accepted")
	}
	newer, e := p.GuestAccess(h.ctx, sid, credential, a.ID(), uuid.NewString(), false)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.AuthorizeGuest(h.ctx, "Bearer "+grant.Token, a.ID()); e == nil {
		t.Fatal("superseded grant accepted")
	}
	if _, e = p.GuestAccess(h.ctx, sid, credential, a.ID(), firstKey, false); !errors.Is(e, ErrConflict) {
		t.Fatal("superseded mint replay resurrected", e)
	}
	revokeKey := uuid.NewString()
	if _, e = p.GuestAccess(h.ctx, sid, credential, a.ID(), revokeKey, true); e != nil {
		t.Fatal(e)
	}
	if _, e = p.AuthorizeGuest(h.ctx, "Bearer "+newer.Token, a.ID()); e == nil {
		t.Fatal("revoked grant accepted")
	}
	latest, e := p.GuestAccess(h.ctx, sid, credential, a.ID(), uuid.NewString(), false)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.GuestAccess(h.ctx, sid, credential, a.ID(), revokeKey, true); e != nil {
		t.Fatal(e)
	}
	if _, e = p.AuthorizeGuest(h.ctx, "Bearer "+latest.Token, a.ID()); e != nil {
		t.Fatal("replay revoke removed later grant", e)
	}
	if e = p.RevokeGuestSession(h.ctx, sid, credential); e != nil {
		t.Fatal(e)
	}
	if _, e = p.AuthorizeGuest(h.ctx, "Bearer "+latest.Token, a.ID()); e == nil {
		t.Fatal("logout did not revoke grants")
	}
}

func TestGuestGrantCurrentOwnerAndCredential(t *testing.T) {
	h := database(t, false)
	_, place := prepared(t, h, 2)
	a, e := h.store.Place(h.ctx, place)
	if e != nil {
		t.Fatal(e)
	}
	grantKey, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	p := &Payment{Store: h.store, GrantKey: grantKey, GrantKeyID: "test-key-v1"}
	if e = h.db.QueryRow(h.ctx, "SELECT connection_id,receiver_profile_id FROM payment_intents WHERE order_id=$1", a.ID()).Scan(&p.Connection, &p.Receiver); e != nil {
		t.Fatal(e)
	}
	sid := a.Owner().PrincipalID
	now := time.Now()
	credential := "verified-session"
	if e = p.CreateGuestSession(h.ctx, sid, credential, now, now.Add(time.Hour)); e != nil {
		t.Fatal(e)
	}
	if _, e = p.GuestAccess(h.ctx, sid, "forged", a.ID(), uuid.NewString(), false); e == nil {
		t.Fatal("caller-supplied SID authorized without credential")
	}
	grant, e := p.GuestAccess(h.ctx, sid, credential, a.ID(), uuid.NewString(), false)
	if e != nil {
		t.Fatal(e)
	}
	// Mimic the ownership projection effect; no generic owner-change API is exposed.
	if _, e = h.db.Exec(h.ctx, "UPDATE orders SET scope=$2 WHERE id=$1", a.ID(), "customer:"+uuid.NewString()); e != nil {
		t.Fatal(e)
	}
	if _, e = p.AuthorizeGuest(h.ctx, "Bearer "+grant.Token, a.ID()); e == nil {
		t.Fatal("owner claim left Guest access live")
	}
}
