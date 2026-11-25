package transport

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Opt-in actual repository Kong config -> actual Order handler -> PostgreSQL.
// No Keycloak server, identity fixture, customer JWT or session is configured.
func TestGuestAnonymousGatewayIssuanceAUTH11(t *testing.T) {
	if os.Getenv("ORDER_GUEST_TEST_KONG") != "true" {
		t.Skip("ORDER_GUEST_TEST_KONG=true and local kong:3.9 image required")
	}
	s := guestService(t)
	server := httptest.NewUnstartedServer(Handler(s))
	listener, e := net.Listen("tcp", "0.0.0.0:0")
	if e != nil {
		t.Fatal(e)
	}
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	source, e := os.ReadFile(filepath.Join("..", "..", "..", "api-gateway", "config", "kong.yml"))
	if e != nil {
		t.Fatal(e)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	config := strings.ReplaceAll(string(source), "http://order-service:8004", fmt.Sprintf("http://host.docker.internal:%d", port))
	folder := t.TempDir()
	if e = os.WriteFile(filepath.Join(folder, "kong.yml"), []byte(config), 0600); e != nil {
		t.Fatal(e)
	}
	spare, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	gatewayPort := spare.Addr().(*net.TCPAddr).Port
	spare.Close()
	name := "f53-guest-gateway-" + uuid.NewString()
	t.Cleanup(func() { exec.Command("docker", "rm", "-f", name).Run() })
	args := []string{"run", "-d", "--name", name, "--add-host", "host.docker.internal:host-gateway", "-p", fmt.Sprintf("127.0.0.1:%d:8000", gatewayPort), "-e", "KONG_DATABASE=off", "-e", "KONG_DECLARATIVE_CONFIG=/config/kong.yml", "-e", "KONG_NGINX_WORKER_PROCESSES=1", "--mount", "type=bind,source=" + folder + ",target=/config,readonly", "kong:3.9"}
	if raw, e := exec.Command("docker", args...).CombinedOutput(); e != nil {
		t.Fatalf("Kong start: %v %s", e, raw)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	base := fmt.Sprintf("http://127.0.0.1:%d", gatewayPort)
	ready := false
	for deadline := time.Now().Add(45 * time.Second); time.Now().Before(deadline); time.Sleep(200 * time.Millisecond) {
		response, e := client.Get(base + "/f53-ready-no-route")
		if e == nil {
			response.Body.Close()
			if response.StatusCode == 404 {
				ready = true
				break
			}
		}
	}
	if !ready {
		t.Fatal("Kong did not become ready")
	}
	request, _ := http.NewRequest("POST", base+"/api/v1/guest-sessions", strings.NewReader("{}"))
	request.Header.Set("Content-Type", "application/json")
	response, e := client.Do(request)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	if response.StatusCode != 201 {
		t.Fatalf("anonymous gateway issuance HTTP %d", response.StatusCode)
	}
	var n int
	if e = s.DB.QueryRow(context.Background(), "SELECT count(*) FROM guest_checkout_sessions").Scan(&n); e != nil || n != 1 {
		t.Fatal("AUTH-11 persisted count", n, e)
	}
	if response.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("session response may be cached")
	}
}
