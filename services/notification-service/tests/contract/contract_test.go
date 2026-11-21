package contract

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/omamx/notification-service/internal/domain"
	"os"
	"testing"
)

func TestPinnedContracts(t *testing.T) {
	b, e := os.ReadFile("../../contracts/examples/intake.json")
	if e != nil {
		t.Fatal(e)
	}
	var c domain.Command
	if e = json.Unmarshal(b, &c); e != nil {
		t.Fatal(e)
	}
	if e = c.Validate(); e != nil {
		t.Fatal(e)
	}
	b, e = os.ReadFile("../../contracts/events/manifest.json")
	if e != nil {
		t.Fatal(e)
	}
	var manifest struct {
		Enabled bool
		Events  []struct{ Schema, SHA256 string }
	}
	if e = json.Unmarshal(b, &manifest); e != nil {
		t.Fatal(e)
	}
	if manifest.Enabled || len(manifest.Events) != 1 {
		t.Fatal("unapproved subscription")
	}
	for _, entry := range manifest.Events {
		schema, e := os.ReadFile("../../contracts/events/" + entry.Schema)
		if e != nil {
			t.Fatal(e)
		}
		if fmt.Sprintf("%x", sha256.Sum256(schema)) != entry.SHA256 {
			t.Fatal("schema pin mismatch")
		}
	}
	b, e = os.ReadFile("../../contracts/notification.openapi.json")
	if e != nil {
		t.Fatal(e)
	}
	var api struct {
		OpenAPI string
		Paths   map[string]json.RawMessage
	}
	if e = json.Unmarshal(b, &api); e != nil || api.OpenAPI != "3.1.0" || len(api.Paths) != 2 {
		t.Fatal("invalid local contract", e)
	}
}
