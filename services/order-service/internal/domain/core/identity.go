package core

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/google/uuid"
)

// Different purposes remain different types even when the normalized SKU and
// quantity inputs yield the same digest. Neither is a full command request hash.
type SelectionHash string
type ManifestHash string
type PayloadHash string

func validID(id string) bool {
	u, err := uuid.Parse(id)
	return err == nil && u != uuid.Nil && u.String() == id
}

func validHash(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func digest(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

type SelectionLine struct {
	Quantity int64  `json:"quantity"`
	SKU      string `json:"sku_code"`
}

// HashSelection accepts the already-consolidated checkout manifest. Inventory
// adapters must consolidate raw RPC lines before invoking their wire profile.
func HashSelection(lines []SelectionLine) (SelectionHash, error) {
	if len(lines) == 0 || len(lines) > 100 {
		return "", ErrSnapshot
	}
	copyLines := append([]SelectionLine(nil), lines...)
	sort.Slice(copyLines, func(i, j int) bool { return copyLines[i].SKU < copyLines[j].SKU })
	for i, line := range copyLines {
		if !skuPattern.MatchString(line.SKU) || line.Quantity < 1 || line.Quantity > 999 ||
			(i > 0 && copyLines[i-1].SKU == line.SKU) {
			return "", ErrSnapshot
		}
	}
	b, err := canonicalJSON(copyLines)
	if err != nil {
		return "", err
	}
	return SelectionHash(digest(b)), nil
}

// HashManifest follows packages/contracts/order-3.2.md: warehouse is checked
// separately, not silently included in the provider items_digest.
func HashManifest(lines []SelectionLine) (ManifestHash, error) {
	h, err := HashSelection(lines)
	return ManifestHash(h), err
}

type CommandIdentity struct {
	scope, action, orderID, key string
	hash                        PayloadHash
}

// NewCommandIdentity hashes the local business payload using order-json-v1.
// This is NOT the Inventory protobuf request_hash profile (which omits defaults
// and transport fields); that mapping belongs to the typed provider adapter.
func NewCommandIdentity(scope, action, orderID, key string, businessJSON []byte) (CommandIdentity, error) {
	if !validToken(scope, 255) || !validToken(action, 128) || !validID(orderID) || !validKey(key) {
		return CommandIdentity{}, ErrIdentity
	}
	v, err := parseJSON(businessJSON)
	if err != nil {
		return CommandIdentity{}, ErrIdentity
	}
	if _, ok := v.(map[string]any); !ok {
		return CommandIdentity{}, ErrIdentity
	}
	b, err := canonicalValue(v)
	if err != nil {
		return CommandIdentity{}, ErrIdentity
	}
	return CommandIdentity{scope, action, orderID, key, PayloadHash(digest(b))}, nil
}

func validKey(key string) bool {
	return len(key) >= 16 && validToken(key, 128)
}

func validToken(key string, maxLength int) bool {
	if len(key) == 0 || len(key) > maxLength {
		return false
	}
	for _, c := range key {
		if c < 0x21 || c > 0x7e {
			return false
		}
	}
	return true
}

// EffectIdentity preserves an adapter's already-pinned remote key/hash through
// retries. It never derives a new key from an attempt, epoch or worker instance.
type EffectIdentity struct {
	provider, scope, action, plannedOrderID, key, requestHash string
}

func NewEffectIdentity(provider, scope, action, plannedOrderID, key, requestHash string) (EffectIdentity, error) {
	if !validToken(provider, 128) || !validToken(scope, 255) || !validToken(action, 128) ||
		!validID(plannedOrderID) || !validKey(key) || !validHash(requestHash) {
		return EffectIdentity{}, ErrIdentity
	}
	return EffectIdentity{provider, scope, action, plannedOrderID, key, requestHash}, nil
}

func (e EffectIdentity) Provider() string       { return e.provider }
func (e EffectIdentity) Scope() string          { return e.scope }
func (e EffectIdentity) Action() string         { return e.action }
func (e EffectIdentity) PlannedOrderID() string { return e.plannedOrderID }
func (e EffectIdentity) Key() string            { return e.key }
func (e EffectIdentity) RequestHash() string    { return e.requestHash }
func (e EffectIdentity) CheckReplay(incoming EffectIdentity) error {
	if e == (EffectIdentity{}) || e != incoming {
		return ErrIdentity
	}
	return nil
}

func (c CommandIdentity) Scope() string            { return c.scope }
func (c CommandIdentity) Action() string           { return c.action }
func (c CommandIdentity) OrderID() string          { return c.orderID }
func (c CommandIdentity) Key() string              { return c.key }
func (c CommandIdentity) RequestHash() PayloadHash { return c.hash }

// CheckReplay is a domain comparison only. Authorization precedes registry
// replay; SQL serialization/uniqueness and retention are persistence duties.
func (c CommandIdentity) CheckReplay(incoming CommandIdentity) error {
	if c == (CommandIdentity{}) || c != incoming {
		return ErrIdentity
	}
	return nil
}
