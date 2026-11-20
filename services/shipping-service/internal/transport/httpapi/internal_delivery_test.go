package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/shipping-service/internal/application"
	"github.com/omamx/shipping-service/internal/carrier"
	"github.com/omamx/shipping-service/internal/domain"
	identityclient "github.com/omamx/shipping-service/internal/infrastructure/identity"
	"github.com/omamx/shipping-service/internal/repository/postgres"
	"github.com/omamx/shipping-service/migrations"
	"github.com/rs/zerolog"
)

type assignmentAuthFixture struct {
	key             *rsa.PrivateKey
	issuer          string
	auth            *jwtStaffAuthenticator
	directory       *identityclient.Directory
	actorID         uuid.UUID
	driverID        uuid.UUID
	actorStatus     atomic.Value
	driverStatus    atomic.Value
	directoryStatus atomic.Int64
}

func newAssignmentAuthFixture(t *testing.T) *assignmentAuthFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &assignmentAuthFixture{key: key, actorID: uuid.New(), driverID: uuid.New()}
	f.actorStatus.Store("ACTIVE")
	f.driverStatus.Store("ACTIVE")
	f.directoryStatus.Store(200)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/certs" {
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "test-key", "use": "sig", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
			return
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ey") || r.Header.Get("X-Internal-Token") != "" || r.Header.Get("X-Actor-Principal-Id") != "" {
			t.Error("Identity lookup did not forward only employee bearer")
		}
		if status := int(f.directoryStatus.Load()); status != 200 {
			w.WriteHeader(status)
			return
		}
		id, status := f.actorID, f.actorStatus.Load().(string)
		if strings.HasPrefix(r.URL.Path, "/api/v1/admin/users/") {
			if r.URL.Path != "/api/v1/admin/users/"+f.driverID.String() {
				w.WriteHeader(404)
				return
			}
			id, status = f.driverID, f.driverStatus.Load().(string)
		} else if r.URL.Path != "/api/v1/users/me" {
			w.WriteHeader(404)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"principalId": id.String(), "idpSubject": "employee-subject", "principalType": "EMPLOYEE", "provisioningStatus": status}})
	}))
	t.Cleanup(server.Close)
	f.issuer = server.URL + "/realm"
	f.auth, err = NewJWTStaffAuthenticator(f.issuer, "omama-api", server.URL+"/certs")
	if err != nil {
		t.Fatal(err)
	}
	f.directory, err = identityclient.NewDirectory(server.URL, true)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *assignmentAuthFixture) token(t *testing.T, grants ...string) string {
	return "Bearer " + signStaffToken(t, f.key, map[string]any{"iss": f.issuer, "sub": "employee-subject", "aud": []string{"omama-api"}, "exp": time.Now().Add(5 * time.Minute).Unix(), "iat": time.Now().Add(-time.Second).Unix(), "typ": "Bearer", "principal_type": "EMPLOYEE", "user_id": f.actorID.String(), "permission_grants": grants})
}

func assignmentHTTPCall(h http.Handler, method, path, auth, key, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Authorization", auth)
	r.Header.Set("Idempotency-Key", key)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestAssignmentRoutesAreDisabledByDefaultAndRequireOwnPermission(t *testing.T) {
	f := newAssignmentAuthFixture(t)
	path := "/api/v2/admin/shipping/shipments/" + uuid.NewString() + "/assignment"
	defaultRouter := NewRouter(nil, "internal-token", "fake-secret", "", zerolog.Nop(), f.auth)
	if w := assignmentHTTPCall(defaultRouter, "POST", path, f.token(t, "SHIPMENT_ASSIGN#GLOBAL", "USER_VIEW#GLOBAL"), "assignment-key-0001", `{}`); w.Code != 404 {
		t.Fatal("disabled assignment route exposed")
	}
	h := NewRouterWithAssignments(nil, "internal-token", "fake-secret", "", zerolog.Nop(), f.auth, f.directory)
	for _, tc := range []struct {
		name, auth string
		status     int
	}{
		{"missing", "", 401}, {"internal token", "Bearer internal-token", 401},
		{"view only", f.token(t, "SHIPMENT_VIEW#GLOBAL"), 403},
		{"warehouse", f.token(t, "SHIPMENT_ASSIGN#WAREHOUSE#WH-1", "USER_VIEW#GLOBAL"), 403},
		{"missing directory grant", f.token(t, "SHIPMENT_ASSIGN#GLOBAL"), 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := assignmentHTTPCall(h, "POST", path, tc.auth, "assignment-key-0001", `{}`)
			var e errorBody
			if w.Code != tc.status || json.Unmarshal(w.Body.Bytes(), &e) != nil || e.RequestID == "" {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}
		})
	}
	full := f.token(t, "SHIPMENT_ASSIGN#GLOBAL", "USER_VIEW#GLOBAL")
	f.actorStatus.Store("SUSPENDED")
	if w := assignmentHTTPCall(h, "POST", path, full, "assignment-key-0001", `{}`); w.Code != 403 {
		t.Fatalf("suspended actor: %d", w.Code)
	}
	f.actorStatus.Store("ACTIVE")
	f.directoryStatus.Store(503)
	if w := assignmentHTTPCall(h, "GET", path, full, "", ""); w.Code != 503 {
		t.Fatalf("directory outage: %d", w.Code)
	}
}

// Real PostgreSQL + HTTP + signed JWT/JWKS; Identity is an explicit HTTP fixture,
// not a claim that the deployed Identity/Keycloak grant provisioning was accepted.
func TestPostgresAssignmentHTTPAndSignedJWT(t *testing.T) {
	dsn := os.Getenv("SHIPPING_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SHIPPING_TEST_DATABASE_URL required; CI supplies PostgreSQL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "shipping_assignment_http_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		clean, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := admin.Exec(clean, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	}()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err = migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	repo := postgres.New(pool)
	internal, _ := carrier.NewInternal(25000, time.Hour)
	svc := application.NewService(repo, carrier.NewRegistry(internal, carrier.NewFake()))
	create := func(code string) *domain.Shipment {
		t.Helper()
		id := uuid.NewString()
		s, err := svc.CreateShipment(ctx, domain.CreateShipmentCommand{IdempotencyKey: "create-" + id, OrderID: id, PackageID: id, CarrierCode: code, ServiceCode: "STANDARD", Recipient: domain.Recipient{Name: "Fixture", Phone: "0900000001", Street: "Test", Ward: "Ward", Province: "Province", CountryCode: "VN"}, Parcel: domain.Parcel{WeightGrams: 500, LengthCM: 20, WidthCM: 15, HeightCM: 10}})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	s := create(carrier.InternalCode)
	f := newAssignmentAuthFixture(t)
	h := NewRouterWithAssignments(svc, "internal-token", "fake-secret", "", zerolog.Nop(), f.auth, f.directory)
	path := "/api/v2/admin/shipping/shipments/" + s.ID.String() + "/assignment"
	full := f.token(t, "SHIPMENT_ASSIGN#GLOBAL", "USER_VIEW#GLOBAL", "traceability.read_internal#GLOBAL")
	read := f.token(t, "SHIPMENT_ASSIGN#GLOBAL")
	body := `{"driver_id":"` + f.driverID.String() + `","expected_version":0,"reason":"Initial dispatch"}`
	call := func(method, path, token, key, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		w := assignmentHTTPCall(h, method, path, token, key, body)
		if w.Code != want {
			t.Fatalf("%s %s: got %d want %d body=%s", method, path, w.Code, want, w.Body)
		}
		return w
	}
	call("GET", path, read, "", "", 404)
	w := call("GET", path+"/history", read, "", "", 200)
	if !strings.Contains(w.Body.String(), `"items":[]`) || !strings.Contains(w.Body.String(), `"total":0`) {
		t.Fatal("empty history contract")
	}
	call("POST", path, full, "", body, 400)
	call("POST", path, full, "assignment-key-0001", strings.TrimSuffix(body, "}")+`,"actor_id":"`+uuid.NewString()+`"}`, 400)
	call("POST", path, full, "assignment-key-0001", `{"driver_id":"`+f.driverID.String()+`","reason":"Dispatch"}`, 400)
	f.driverStatus.Store("SUSPENDED")
	call("POST", path, full, "assignment-key-0001", body, 422)
	f.driverStatus.Store("ACTIVE")
	first := call("POST", path, full, "assignment-key-0001", body, 200)
	var assigned domain.InternalDeliveryAssignment
	if err := json.Unmarshal(first.Body.Bytes(), &assigned); err != nil || assigned.ActorID != f.actorID || assigned.DriverID != f.driverID || assigned.Version != 1 {
		t.Fatalf("assignment not JWT-bound: %+v %v", assigned, err)
	}
	replay := call("POST", path, full, "assignment-key-0001", body, 200)
	if replay.Body.String() != first.Body.String() {
		t.Fatal("replay changed original snapshot")
	}
	call("GET", path, read, "", "", 200)
	call("GET", path+"/history", read, "", "", 200)
	call("POST", path, full, "assignment-key-0001", strings.Replace(body, "Initial dispatch", "Changed reason", 1), 409)
	call("POST", path, full, "assignment-stale-0001", body, 409)
	fake := create(carrier.FakeCode)
	call("POST", "/api/v2/admin/shipping/shipments/"+fake.ID.String()+"/assignment", full, "assignment-fake-0001", body, 422)
	call("POST", strings.Replace(path, s.ID.String(), uuid.NewString(), 1), full, "assignment-missing-001", body, 404)
	call("POST", strings.Replace(path, s.ID.String(), "invalid", 1), full, "assignment-invalid-001", body, 400)
	f.directoryStatus.Store(503)
	call("POST", path, full, "assignment-offline-001", body, 503)
	f.directoryStatus.Store(200)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM internal_delivery_assignment_events WHERE shipment_id=$1`, s.ID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("denied requests wrote audit: %d %v", n, err)
	}
	if _, err := svc.ConfirmHandover(ctx, s.ID, "handover-after-assign-001"); err != nil {
		t.Fatal(err)
	}
	call("POST", path, full, "assignment-key-0001", body, 200)
	call("POST", path, full, "assignment-late-0001", strings.Replace(body, `"expected_version":0`, `"expected_version":1`, 1), 409)

	t.Run("driver acceptance JWT ownership and retry", func(t *testing.T) {
		s := create(carrier.InternalCode)
		acceptPath := "/api/v2/admin/shipping/shipments/" + s.ID.String() + "/accept"
		token := f.token(t, "SHIPMENT_EXECUTE#GLOBAL")
		workbenchPath := "/api/v2/admin/shipping/workbench/shipments"
		call("GET", workbenchPath, "", "", "", 401)
		call("GET", workbenchPath, full, "", "", 403)
		empty := call("GET", workbenchPath, token, "", "", 200)
		if !strings.Contains(empty.Body.String(), `"items":[]`) || !strings.Contains(empty.Body.String(), `"total":0`) {
			t.Fatal("empty workbench contract")
		}
		reassigned := create(carrier.InternalCode)
		reassign := domain.AssignInternalDelivery{ShipmentID: reassigned.ID, DriverID: f.actorID, ActorID: f.actorID, ExpectedVersion: 0, Reason: "Ownership fixture", IdempotencyKey: "read-reassign-0001"}
		if _, err := svc.AssignInternalDelivery(ctx, reassign); err != nil {
			t.Fatal(err)
		}
		call("GET", workbenchPath+"/"+reassigned.ID.String(), token, "", "", 200)
		reassign.DriverID = f.driverID
		reassign.ExpectedVersion = 1
		reassign.IdempotencyKey = "read-reassign-0002"
		if _, err := svc.AssignInternalDelivery(ctx, reassign); err != nil {
			t.Fatal(err)
		}
		denied := call("GET", workbenchPath+"/"+reassigned.ID.String(), token, "", "", 404)
		if strings.Contains(denied.Body.String(), "0900000001") {
			t.Fatal("reassignment leaked recipient")
		}
		call("GET", workbenchPath+"/"+fake.ID.String(), token, "", "", 404)
		call("GET", workbenchPath+"?driver_id="+f.driverID.String(), token, "", "", 400)
		call("GET", workbenchPath+"?limit=0", token, "", "", 400)
		call("GET", workbenchPath+"?offset=-1", token, "", "", 400)
		call("GET", workbenchPath+"?limit=1&limit=2", token, "", "", 400)
		call("GET", workbenchPath+"/"+s.ID.String(), token, "", "", 404)
		defaultRouter := NewRouter(svc, "internal-token", "fake-secret", "", zerolog.Nop(), f.auth)
		if w := assignmentHTTPCall(defaultRouter, "GET", workbenchPath, token, "", ""); w.Code != 404 {
			t.Fatal("disabled workbench exposed")
		}
		f.actorStatus.Store("SUSPENDED")
		call("GET", workbenchPath, token, "", "", 403)
		f.actorStatus.Store("ACTIVE")
		f.directoryStatus.Store(503)
		call("GET", workbenchPath+"/"+s.ID.String(), token, "", "", 503)
		f.directoryStatus.Store(200)
		if w := assignmentHTTPCall(defaultRouter, "POST", acceptPath, token, "acceptance-key-0001", `{"expected_version":1}`); w.Code != 404 {
			t.Fatal("disabled acceptance exposed")
		}
		call("POST", acceptPath, "", "acceptance-key-0001", `{"expected_version":1}`, 401)
		call("POST", acceptPath, full, "acceptance-key-0001", `{"expected_version":1}`, 403)
		call("POST", acceptPath, f.token(t, "SHIPMENT_EXECUTE#WAREHOUSE#WH-1"), "acceptance-key-0001", `{"expected_version":1}`, 403)
		call("POST", acceptPath, token, "acceptance-key-0001", `{"expected_version":1}`, 404)
		a := domain.AssignInternalDelivery{ShipmentID: s.ID, DriverID: f.driverID, ActorID: f.actorID, ExpectedVersion: 0, Reason: "Dispatch", IdempotencyKey: "dispatch-for-accept-001"}
		if _, err := svc.AssignInternalDelivery(ctx, a); err != nil {
			t.Fatal(err)
		}
		call("POST", acceptPath, token, "acceptance-key-0001", `{"expected_version":1}`, 403)
		call("GET", workbenchPath+"/"+s.ID.String(), token, "", "", 404)
		a.DriverID = f.actorID
		a.ExpectedVersion = 1
		a.IdempotencyKey = "dispatch-for-accept-002"
		if _, err := svc.AssignInternalDelivery(ctx, a); err != nil {
			t.Fatal(err)
		}
		list := call("GET", workbenchPath+"?limit=1&offset=0", token, "", "", 200)
		var driverList domain.DriverShipmentList
		if err := json.Unmarshal(list.Body.Bytes(), &driverList); err != nil || driverList.Total != 1 || len(driverList.Items) != 1 || driverList.Items[0].Accepted || driverList.Items[0].AssignmentVersion != 2 {
			t.Fatalf("workbench list: %s %v", list.Body, err)
		}
		if strings.Contains(list.Body.String(), `"recipient"`) || strings.Contains(list.Body.String(), `"phone"`) {
			t.Fatal("list exposed recipient")
		}
		ownDetail := call("GET", workbenchPath+"/"+s.ID.String(), token, "", "", 200)
		if ownDetail.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("workbench PII cache policy missing")
		}
		var driverDetail domain.DriverShipmentDetail
		if err := json.Unmarshal(ownDetail.Body.Bytes(), &driverDetail); err != nil || driverDetail.Recipient.Phone != "0900000001" || driverDetail.AssignmentVersion != 2 {
			t.Fatalf("owned detail: %s %v", ownDetail.Body, err)
		}
		call("GET", workbenchPath+"?offset=1", token, "", "", 200)
		call("GET", workbenchPath+"/"+s.ID.String()+"?actor_id="+f.actorID.String(), token, "", "", 400)
		call("POST", acceptPath, token, "", `{"expected_version":2}`, 400)
		call("POST", acceptPath, token, "acceptance-key-0001", `{}`, 400)
		call("POST", acceptPath, token, "acceptance-key-0001", `{"expected_version":2,"actor_id":"`+f.actorID.String()+`"}`, 400)
		call("POST", acceptPath, token, "acceptance-key-0001", `{"expected_version":0}`, 422)
		call("POST", acceptPath, token, "acceptance-key-0001", `{"expected_version":1}`, 409)
		f.actorStatus.Store("SUSPENDED")
		call("POST", acceptPath, token, "acceptance-key-0001", `{"expected_version":2}`, 403)
		f.actorStatus.Store("ACTIVE")
		f.directoryStatus.Store(503)
		call("POST", acceptPath, token, "acceptance-key-0001", `{"expected_version":2}`, 503)
		f.directoryStatus.Store(200)
		first := call("POST", acceptPath, token, "acceptance-key-0001", `{"expected_version":2}`, 200)
		acceptedDetail := call("GET", workbenchPath+"/"+s.ID.String(), token, "", "", 200)
		if !strings.Contains(acceptedDetail.Body.String(), `"accepted":true`) {
			t.Fatal("acceptance missing from workbench")
		}
		var receipt domain.InternalDeliveryAcceptance
		if err := json.Unmarshal(first.Body.Bytes(), &receipt); err != nil || receipt.DriverID != f.actorID || receipt.AssignmentVersion != 2 {
			t.Fatalf("invalid receipt: %+v %v", receipt, err)
		}
		if replay := call("POST", acceptPath, token, "acceptance-key-0001", `{"expected_version":2}`, 200); replay.Body.String() != first.Body.String() {
			t.Fatal("replay changed receipt")
		}
		call("POST", acceptPath, token, "acceptance-key-0001", `{"expected_version":1}`, 409)
		call("POST", strings.Replace(acceptPath, s.ID.String(), fake.ID.String(), 1), token, "acceptance-fake-0001", `{"expected_version":1}`, 422)
		if _, err := svc.ConfirmHandover(ctx, s.ID, "acceptance-handover-001"); err != nil {
			t.Fatal(err)
		}
		call("POST", acceptPath, token, "acceptance-key-0001", `{"expected_version":2}`, 200)
		call("POST", acceptPath, token, "acceptance-late-0001", `{"expected_version":2}`, 409)
		progressPath := strings.TrimSuffix(acceptPath, "/accept") + "/progress"
		shipment, _, err := svc.GetShipment(ctx, s.ID)
		if err != nil {
			t.Fatal(err)
		}
		payload := `{"assignment_version":2,"expected_revision":` + fmt.Sprint(shipment.Revision) + `,"status":"IN_TRANSIT","reason":"Started route"}`
		call("POST", progressPath, "", "progress-key-0001", payload, 401)
		call("POST", progressPath, full, "progress-key-0001", payload, 403)
		call("POST", progressPath, token, "", payload, 400)
		call("POST", progressPath, token, "progress-key-0001", strings.TrimSuffix(payload, "}")+`,"actor_id":"`+f.actorID.String()+`"}`, 400)
		call("POST", progressPath, token, "progress-key-0001", strings.Replace(payload, "IN_TRANSIT", "DELIVERED", 1), 409)
		f.actorStatus.Store("SUSPENDED")
		call("POST", progressPath, token, "progress-key-0001", payload, 403)
		f.actorStatus.Store("ACTIVE")
		f.directoryStatus.Store(503)
		call("POST", progressPath, token, "progress-key-0001", payload, 503)
		f.directoryStatus.Store(200)
		progress := call("POST", progressPath, token, "progress-key-0001", payload, 200)
		if retry := call("POST", progressPath, token, "progress-key-0001", payload, 200); retry.Body.String() != progress.Body.String() {
			t.Fatal("progress replay changed")
		}
		call("POST", progressPath, token, "progress-stale-0001", payload, 409)
		call("POST", progressPath, token, "progress-key-0001", strings.Replace(payload, "Started route", "Changed reason", 1), 409)
	})
}
