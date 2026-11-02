package paymentauthority

import (
	"context"
	"github.com/omamx/order-service/internal/persistence"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthorityHTTPFailureBoundaries(t *testing.T) {
	for _, mode := range []string{"ok", "unauthorized", "redirect", "duplicate", "oversize"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Internal-Token") != "synthetic-internal-token" {
					t.Error("missing internal authentication")
				}
				switch mode {
				case "unauthorized":
					w.WriteHeader(401)
				case "redirect":
					w.Header().Set("Location", "https://example.invalid/")
					w.WriteHeader(302)
				case "duplicate":
					w.Write([]byte(`{"approval_id":"a","approval_id":"b"}`))
				case "oversize":
					w.Write([]byte(strings.Repeat("a", 65537)))
				default:
					w.Write([]byte(`{"approval_id":"trusted","state":"APPROVED"}`))
				}
			}))
			defer server.Close()
			client, e := New(server.URL, "synthetic-internal-token")
			if e != nil {
				t.Fatal(e)
			}
			approval, e := client.RefundApproval(context.Background(), "trusted")
			if mode == "ok" {
				if e != nil || approval.ID != "trusted" {
					t.Fatal(approval, e)
				}
			} else if e == nil {
				t.Fatal("accepted untrusted authority response")
			}
		})
	}
}
func TestUploadUsesStableKeyAndAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/internal/payments/evidence-uploads" || r.Header.Get("X-Idempotency-Key") != "stable-key" || r.Header.Get("X-Internal-Token") != "synthetic-internal-token" {
			t.Error("upload contract")
		}
		w.WriteHeader(201)
		w.Write([]byte(`{"evidence_id":"trusted-object"}`))
	}))
	defer server.Close()
	c, _ := New(server.URL, "synthetic-internal-token")
	ticket, e := c.CreateUpload(context.Background(), "actor", "stable-key", persistence.UploadRequest{})
	if e != nil || ticket.ID != "trusted-object" {
		t.Fatal(ticket, e)
	}
}
