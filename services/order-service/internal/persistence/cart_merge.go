package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
)

var ErrMergeConflict = errors.New("CART_MERGE_CONFLICT")

func LockCartScopes(ctx context.Context, tx pgx.Tx, scopes ...string) error {
	scopes = append([]string(nil), scopes...)
	sort.Strings(scopes)
	for _, scope := range scopes {
		if _, e := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", "CART:"+scope); e != nil {
			return e
		}
	}
	return nil
}

func (p *Payment) CartMergeResult(ctx context.Context, customerScope, key string) (out domain.CartMergeResult, err error) {
	var raw []byte
	err = p.Store.db.QueryRow(ctx, "SELECT response FROM guest_cart_merges WHERE customer_scope=$1 AND merge_key=$2", customerScope, key).Scan(&raw)
	if err == nil {
		err = json.Unmarshal(raw, &out)
	}
	return
}

// The callback reads legacy Redis carts and Catalog only while both SQL cart
// locks are held. Every keycloak cart mutation/cleanup uses these same locks.
// Cart writes, immutable outcome and revocation commit together in PostgreSQL.
func (p *Payment) MergeGuestCart(ctx context.Context, customerScope, sid, credential, key string, prepare func(pgx.Tx) (domain.CartMergeResult, error)) (out domain.CartMergeResult, err error) {
	if !strings.HasPrefix(customerScope, "customer:") || !validID(strings.TrimPrefix(customerScope, "customer:")) || !validID(sid) || !commandKey(key) {
		return out, ErrInput
	}
	guestScope := "guest:" + sid
	tx, e := p.Store.db.Begin(ctx)
	if e != nil {
		return out, e
	}
	defer tx.Rollback(ctx)
	if e = LockCartScopes(ctx, tx, customerScope, guestScope); e != nil {
		return out, e
	}
	var source string
	var previous []byte
	e = tx.QueryRow(ctx, "SELECT guest_scope,response FROM guest_cart_merges WHERE customer_scope=$1 AND merge_key=$2", customerScope, key).Scan(&source, &previous)
	if e == nil {
		if source != guestScope {
			return out, ErrMergeConflict
		}
		if e = json.Unmarshal(previous, &out); e != nil {
			return out, e
		}
		return out, nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return out, e
	}
	var active bool
	e = tx.QueryRow(ctx, `SELECT revoked_at IS NULL AND expires_at>clock_timestamp() AND credential_hash=$2 FROM guest_checkout_sessions WHERE sid=$1 FOR UPDATE`, sid, p.guestCredential(credential)).Scan(&active)
	if errors.Is(e, pgx.ErrNoRows) {
		return out, ErrScope
	}
	if e != nil {
		return out, e
	}
	if !active {
		return out, ErrScope
	}
	var merged bool
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM guest_cart_merges WHERE guest_scope=$1)", guestScope).Scan(&merged); e != nil {
		return out, e
	}
	if merged {
		return out, ErrMergeConflict
	}
	out, e = prepare(tx)
	if e != nil {
		return out, e
	}
	out.Key = key
	if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&out.MergedAt); e != nil {
		return out, e
	}
	raw, e := json.Marshal(out)
	if e != nil {
		return out, e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO guest_cart_merges(customer_scope,merge_key,guest_scope,response) VALUES($1,$2,$3,$4)", customerScope, key, guestScope, raw); e != nil {
		return out, e
	}
	tag, e := tx.Exec(ctx, "UPDATE guest_checkout_sessions SET revoked_at=clock_timestamp() WHERE sid=$1 AND revoked_at IS NULL", sid)
	if e != nil {
		return out, e
	}
	if tag.RowsAffected() != 1 {
		return out, ErrScope
	}
	if e = tx.Commit(ctx); e != nil {
		// Recover a lost commit acknowledgement from the persisted customer result.
		lookup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		var recovered []byte
		if read := p.Store.db.QueryRow(lookup, "SELECT response FROM guest_cart_merges WHERE customer_scope=$1 AND merge_key=$2 AND guest_scope=$3", customerScope, key, guestScope).Scan(&recovered); read == nil {
			if json.Unmarshal(recovered, &out) == nil {
				return out, nil
			}
		}
		return out, ErrCommitUnknown
	}
	return out, nil
}

func WriteStorefrontCart(ctx context.Context, tx pgx.Tx, scope string, cart domain.Cart, cartID string) error {
	if cartID == "" {
		cartID = uuid.NewString()
	}
	raw, e := json.Marshal(cart.Items)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO storefront_carts(scope,cart_id,revision,items) VALUES($1,$2,$3,$4)
 ON CONFLICT(scope) DO UPDATE SET cart_id=EXCLUDED.cart_id,revision=EXCLUDED.revision,items=EXCLUDED.items,expires_at=clock_timestamp()+interval '30 days',updated_at=clock_timestamp()`, scope, cartID, cart.Revision, raw)
	return e
}
