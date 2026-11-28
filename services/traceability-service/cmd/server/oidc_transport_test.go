package main

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestOIDCTransportAcceptsOnlyTrustedTLSAndDoesNotFollowRedirect(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "https://untrusted.example.test", 302)
			return
		}
		w.WriteHeader(200)
	}))
	defer server.Close()
	ca := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	client, err := oidcHTTPClient(ca)
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseIdleConnections()
	response, err := client.Get(server.URL + "/redirect")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 302 {
		t.Fatal("followed issuer redirect")
	}
	// Configure a fresh transport before its first request: mutating a transport
	// already used by HTTP can retain an established connection on newer Go releases.
	wrongHost, err := oidcHTTPClient(ca)
	if err != nil {
		t.Fatal(err)
	}
	defer wrongHost.CloseIdleConnections()
	wrongHost.Transport.(*http.Transport).TLSClientConfig.ServerName = "different-issuer.invalid"
	if response, err = wrongHost.Get(server.URL); err == nil {
		response.Body.Close()
		t.Fatal("hostname verification disabled")
	}
	// A malformed private CA cannot silently fall back to a different trust mode.
	if err = os.WriteFile(ca, []byte("not a certificate"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = oidcHTTPClient(ca); err == nil {
		t.Fatal("invalid CA accepted")
	}
	if _, err = oidcHTTPClient(filepath.Join(t.TempDir(), "missing.pem")); err == nil {
		t.Fatal("missing CA accepted")
	}
}
