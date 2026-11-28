package core

import (
	"encoding/json"
	"reflect"
	"time"
)

type historicalCatalogEvidence struct {
	SnapshotID string    `json:"snapshot_id"`
	Version    string    `json:"version"`
	ValidUntil time.Time `json:"valid_until"`
}

// DecodePersistedSnapshot supports already-frozen pre-pricing Catalog evidence.
// Only the repository reader uses this compatibility path. New snapshots still
// use DecodeSnapshot and the current writer schema. No stored bytes, logical
// hash, item price, or lifecycle state are migrated or rewritten.
func DecodePersistedSnapshot(data []byte) (Snapshot, error) {
	if current, err := DecodeSnapshot(data); err == nil {
		return current, nil
	}
	value, err := parseJSON(data)
	if err != nil {
		return Snapshot{}, ErrSnapshot
	}
	root, ok := value.(map[string]any)
	if !ok || requireShape(root["catalog"], reflect.TypeOf(historicalCatalogEvidence{})) != nil {
		return Snapshot{}, ErrSnapshot
	}
	catalogJSON, err := json.Marshal(root["catalog"])
	var old historicalCatalogEvidence
	if err != nil || json.Unmarshal(catalogJSON, &old) != nil || !text(old.SnapshotID) || !text(old.Version) {
		return Snapshot{}, ErrSnapshot
	}
	provenance, ok := root["provenance"].(map[string]any)
	if !ok {
		return Snapshot{}, ErrSnapshot
	}
	// Validate all shared fields with today's strict decoder. The evaluation
	// instant is projected from immutable historical provenance, never live data.
	root["catalog"] = map[string]any{"catalog_version": old.Version,
		"evaluated_at": provenance["canonicalized_at"], "channel": root["channel"]}
	projected, err := canonicalValue(root)
	if err != nil {
		return Snapshot{}, ErrSnapshot
	}
	current, err := DecodeSnapshot(projected)
	if err != nil || !current.Data().PlacedAt.Before(old.ValidUntil) {
		return Snapshot{}, ErrSnapshot
	}
	view := current.Data().Catalog
	// The current decoder canonicalizes shared timestamps and SKU ordering.
	// Reinsert the original, typed Catalog evidence before computing its hash.
	canonicalRoot, err := parseJSON(current.Bytes())
	if err != nil {
		return Snapshot{}, ErrSnapshot
	}
	catalogJSON, err = json.Marshal(old)
	if err != nil {
		return Snapshot{}, ErrSnapshot
	}
	originalCatalog, err := parseJSON(catalogJSON)
	if err != nil {
		return Snapshot{}, ErrSnapshot
	}
	canonicalRoot.(map[string]any)["catalog"] = originalCatalog
	frozen, err := canonicalValue(canonicalRoot)
	if err != nil {
		return Snapshot{}, ErrSnapshot
	}
	return Snapshot{canonical: string(frozen), hash: PayloadHash(digest(frozen)), legacyCatalog: &view}, nil
}
