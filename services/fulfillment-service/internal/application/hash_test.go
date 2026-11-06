package application

import (
	"encoding/json"
	f "github.com/omamx/fulfillment-service/internal/gen/fulfillment/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
	"testing"
)

func TestGoldenGrpcHashes(t *testing.T) {
	raw, e := os.ReadFile("../../../../docs/03_api_specs/fulfillment-service/canonical_hash_vectors.json")
	if e != nil {
		t.Fatal(e)
	}
	var v struct {
		Vectors []struct {
			Name  string          `json:"name"`
			Model json.RawMessage `json:"canonical_model"`
			SHA   string          `json:"sha256"`
		} `json:"vectors"`
	}
	if e = json.Unmarshal(raw, &v); e != nil {
		t.Fatal(e)
	}
	for _, x := range v.Vectors {
		var got string
		if x.Name == "grpc_cancel_task" {
			var m f.CancelTaskRequest
			if e = protojson.Unmarshal(x.Model, &m); e != nil {
				t.Fatal(e)
			}
			got, e = ProtoHash(&m)
		} else if x.Name == "grpc_accept_task" {
			var m map[string]json.RawMessage
			_ = json.Unmarshal(x.Model, &m)
			var permit map[string]any
			_ = json.Unmarshal(m["permit"], &permit)
			permit["expires_at"] = "2026-10-30T07:00:30Z"
			m["permit"], _ = json.Marshal(permit)
			b, _ := json.Marshal(m)
			var req f.AcceptTaskRequest
			if e = protojson.Unmarshal(b, &req); e != nil {
				t.Fatal(e)
			}
			got, e = ProtoHash(&req)
		} else {
			continue
		}
		if e != nil || got != x.SHA {
			t.Fatalf("%s: %s != %s (%v)", x.Name, got, x.SHA, e)
		}
	}
}
