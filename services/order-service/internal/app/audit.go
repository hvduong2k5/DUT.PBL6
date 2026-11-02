package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

// A single SELECT observes a consistent PostgreSQL statement snapshot.
func (s *Service) VerifyAudit(ctx context.Context) error {
	rows, e := s.DB.Query(ctx, "SELECT id,COALESCE(order_id::text,''),actor,action,details,prev_hash,hash FROM audit_records ORDER BY sequence")
	if e != nil {
		return e
	}
	defer rows.Close()
	previous := ""
	for rows.Next() {
		var id, order, actor, action, prev, hash string
		var raw []byte
		if e = rows.Scan(&id, &order, &actor, &action, &raw, &prev, &hash); e != nil {
			return e
		}
		if prev != previous {
			return errors.New("audit chain discontinuity")
		}
		var details any
		if e = json.Unmarshal(raw, &details); e != nil {
			return e
		}
		canonical, e := json.Marshal(details)
		if e != nil {
			return e
		}
		mac := hmac.New(sha256.New, []byte(s.C.CallbackSecret))
		fmt.Fprintf(mac, "%s|%s|%s|%s|%s|%s", prev, id, order, actor, action, canonical)
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(hash), []byte(expected)) {
			return errors.New("audit integrity mismatch")
		}
		previous = hash
	}
	return rows.Err()
}
