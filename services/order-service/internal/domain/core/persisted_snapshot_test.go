package core

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Golden vectors are copied from dev@ac5395c5 before PR176. Keep their original
// expected hashes; regenerating them with the new writer would hide regressions.
func historicalSnapshots(t *testing.T) []fixture {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "order-snapshot.pre-pricing.examples.json"))
	if err != nil {
		t.Fatal(err)
	}
	var examples struct {
		Cases []fixture `json:"cases"`
	}
	if err = json.Unmarshal(b, &examples); err != nil || len(examples.Cases) != 2 {
		t.Fatal("missing historical snapshot vectors", err)
	}
	return examples.Cases
}

func TestPersistedHistoricalSnapshotKeepsOriginalHashAndLifecycle(t *testing.T) {
	for _, example := range historicalSnapshots(t) {
		t.Run(example.Name, func(t *testing.T) {
			if _, err := DecodeSnapshot(example.Snapshot); err == nil {
				t.Fatal("new snapshot decoder accepted a retired writer schema")
			}
			snapshot, err := DecodePersistedSnapshot(example.Snapshot)
			if err != nil || snapshot.Hash() != example.ExpectedHash {
				t.Fatal("historical logical hash changed", snapshot.Hash(), err)
			}
			data := snapshot.Data()
			if data.Catalog.CatalogVersion != data.Items[0].CatalogVersion || !data.Catalog.EvaluatedAt.Equal(data.Provenance.CanonicalizedAt) {
				t.Fatal("historical read metadata changed the commercial binding")
			}
			proof := ResourceProof{OrderID: data.OrderID, ReservationID: "historical-reservation", WarehouseID: data.WarehouseID,
				ItemsDigest: data.Provenance.ItemsDigest, State: "RESERVED", ResourceVersion: 1, DecisionID: "historical-decision",
				DecisionAt: data.PlacedAt.Add(-time.Second), ExpiresAt: data.PlacedAt.Add(15 * time.Minute)}
			if data.Payment.Method == "COD" {
				proof.State = "COMMITTED"
			}
			aggregate, err := Place(snapshot, data.OrderID, proof, data.PlacedAt)
			if err != nil {
				t.Fatal(err)
			}
			restored, err := Restore(snapshot, aggregate.Record())
			if err != nil || restored.Snapshot().Hash() != example.ExpectedHash || restored.Version() != aggregate.Version() {
				t.Fatal("historical state failed to restore", err)
			}
			original := snapshot.Bytes()
			data.Items[0].UnitPrice++
			data.Catalog.CatalogVersion = "mutated"
			if !bytes.Equal(original, snapshot.Bytes()) || snapshot.Data().Catalog.CatalogVersion == "mutated" {
				t.Fatal("read projection mutated the immutable snapshot")
			}
			again, err := DecodePersistedSnapshot(snapshot.Bytes())
			if err != nil || !bytes.Equal(original, again.Bytes()) || again.Hash() != example.ExpectedHash {
				t.Fatal("historical read round trip changed canonical bytes", err)
			}
		})
	}
}

func TestPersistedSnapshotRetainsCurrentWriterEncoding(t *testing.T) {
	for _, example := range fixtures(t) {
		current, err := DecodeSnapshot(example.Snapshot)
		if err != nil {
			t.Fatal(err)
		}
		read, err := DecodePersistedSnapshot(example.Snapshot)
		if err != nil || !bytes.Equal(current.Bytes(), read.Bytes()) || current.Hash() != read.Hash() {
			t.Fatal("current writer encoding changed", err)
		}
	}
}

func TestPersistedHistoricalSnapshotRejectsMalformedEvidence(t *testing.T) {
	mutations := map[string]func(map[string]any){
		"extra root":                func(root map[string]any) { root["extra"] = true },
		"mixed catalog schemas":     func(root map[string]any) { root["catalog"].(map[string]any)["evaluated_at"] = root["placed_at"] },
		"missing snapshot evidence": func(root map[string]any) { delete(root["catalog"].(map[string]any), "snapshot_id") },
		"blank snapshot evidence":   func(root map[string]any) { root["catalog"].(map[string]any)["snapshot_id"] = " " },
		"mismatched item version":   func(root map[string]any) { root["catalog"].(map[string]any)["version"] = "wrong-version" },
		"expired at placement":      func(root map[string]any) { root["catalog"].(map[string]any)["valid_until"] = root["placed_at"] },
		"bad expiry":                func(root map[string]any) { root["catalog"].(map[string]any)["valid_until"] = "invalid" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			var root map[string]any
			if err := json.Unmarshal(historicalSnapshots(t)[0].Snapshot, &root); err != nil {
				t.Fatal(err)
			}
			mutate(root)
			data, _ := json.Marshal(root)
			if _, err := DecodePersistedSnapshot(data); err == nil {
				t.Fatal("invalid historical evidence accepted")
			}
		})
	}
	duplicate := bytes.Replace(historicalSnapshots(t)[0].Snapshot, []byte(`"catalog":`), []byte(`"catalog":{},"catalog":`), 1)
	if _, err := DecodePersistedSnapshot(duplicate); err == nil {
		t.Fatal("duplicate keys accepted")
	}
}
