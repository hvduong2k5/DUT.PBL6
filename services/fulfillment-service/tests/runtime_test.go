package tests

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/fulfillment-service/internal/application"
	d "github.com/omamx/fulfillment-service/internal/domain"
	"github.com/omamx/fulfillment-service/internal/persistence"
	"github.com/omamx/fulfillment-service/internal/transport"
	"github.com/omamx/fulfillment-service/migrations"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	dsn := os.Getenv("FULFILLMENT_TEST_DATABASE")
	if dsn != "" {
		if !strings.Contains(dsn, "/fulfillment_test") {
			panic("tests require an isolated fulfillment_test database")
		}
		var e error
		pool, e = pgxpool.New(context.Background(), dsn)
		if e != nil {
			panic(e)
		}
		if e = migrations.Apply(context.Background(), pool); e != nil {
			panic(e)
		}
	}
	code := m.Run()
	if pool != nil {
		pool.Close()
	}
	os.Exit(code)
}
func fixture(t *testing.T) (*persistence.Store, d.AdmissionInput) {
	t.Helper()
	if pool == nil {
		t.Skip("FULFILLMENT_TEST_DATABASE not set; real PostgreSQL test not run")
	}
	b, e := os.ReadFile("../../../docs/03_api_specs/fulfillment-service/examples.json")
	if e != nil {
		t.Fatal(e)
	}
	var x map[string]json.RawMessage
	if e = json.Unmarshal(b, &x); e != nil {
		t.Fatal(e)
	}
	var i d.Inputs
	var stock d.Stock
	if e = json.Unmarshal(x["packing_snapshot"], &i); e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(x["committed_allocation"], &stock); e != nil {
		t.Fatal(e)
	}
	now := time.Now().UTC()
	i.OrderID = uuid.NewString()
	i.ReservationID = uuid.NewString()
	i.Warehouse = "WH-" + uuid.NewString()
	i.Policy.Evidence = "OPTIONAL"
	due := now.Add(time.Hour)
	i.Due = &due
	lineMap := map[string]string{}
	for n := range i.Lines {
		old := i.Lines[n].ID
		i.Lines[n].ID = uuid.NewString()
		lineMap[old] = i.Lines[n].ID
	}
	stock.OrderID = i.OrderID
	stock.ReservationID = i.ReservationID
	stock.Warehouse = i.Warehouse
	stock.Observed = now
	for n := range stock.Slots {
		stock.Slots[n].ID = uuid.NewString()
		stock.Slots[n].BatchID = uuid.NewString()
		stock.Slots[n].LineID = lineMap[stock.Slots[n].LineID]
		stock.Slots[n].Until = now.Add(24 * time.Hour)
		stock.Slots[n].Version = stock.Version
	}
	c := d.Command{Caller: "service:fulfillment-service", Action: "AcceptTask", OrderID: i.OrderID, Generation: 1, Key: "accept-" + uuid.NewString(), Hash: d.Hash(i)}
	permit := d.Permit{ID: uuid.NewString(), OrderID: i.OrderID, Digest: i.Digest, Warehouse: i.Warehouse, Key: c.Key, Audience: "fulfillment-service", Policy: i.Policy.Version, Generation: 1, OrderVersion: 8, Expires: now.Add(time.Minute), Allowed: true}
	return &persistence.Store{DB: pool}, d.AdmissionInput{Command: c, Permit: permit, Inputs: i, Stock: stock}
}
func mustTask(t *testing.T, s *persistence.Store, in d.AdmissionInput) d.Task {
	t.Helper()
	v, e := s.Accept(context.Background(), in)
	if e != nil || v.State != "ACCEPTED" {
		t.Fatalf("accept: %+v %v", v, e)
	}
	task, e := s.Task(context.Background(), *v.TaskID)
	if e != nil {
		t.Fatal(e)
	}
	return task
}
func principal(t d.Task) d.Principal {
	return d.Principal{ID: uuid.NewString(), Warehouses: []string{t.Warehouse}, Permissions: []string{"FULFILLMENT_TASK_READ", "FULFILLMENT_TASK_READ_HISTORY", "FULFILLMENT_TASK_WORK", "FULFILLMENT_TASK_ASSIGN_SELF", "FULFILLMENT_TASK_SEAL", "FULFILLMENT_TASK_BLOCK", "FULFILLMENT_TASK_RESOLVE_BLOCK", "FULFILLMENT_MEDIA_UPLOAD", "FULFILLMENT_MEDIA_READ_METADATA", "FULFILLMENT_MEDIA_VIEW", "FULFILLMENT_TASK_REOPEN"}}
}
func change(t *testing.T, s *persistence.Store, p d.Principal, task d.Task, m d.Mutation) d.Task {
	t.Helper()
	m.Expected = task.Version
	v, e := s.Mutate(context.Background(), p, task.ID, "mutation-"+uuid.NewString(), d.Hash(m), m, nil)
	if e != nil {
		t.Fatalf("%s: %v", m.Action, e)
	}
	return v
}
func TestAdmissionRaceReplayAndGeneration(t *testing.T) {
	for n := 0; n < 12; n++ {
		s, in := fixture(t)
		var a, c d.Decision
		var ae, ce error
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); <-start; a, ae = s.Accept(context.Background(), in) }()
		go func() {
			defer wg.Done()
			<-start
			c, ce = s.Cancel(context.Background(), d.Command{Caller: "service:order-service", Action: "CancelTask", OrderID: in.Inputs.OrderID, Generation: 1, Key: "cancel-" + uuid.NewString(), Hash: d.Hash("cancel")}, "test race")
		}()
		close(start)
		wg.Wait()
		if ae != nil || ce != nil {
			t.Fatalf("race: %v %v", ae, ce)
		}
		if a.State == "ACCEPTED" {
			if c.State != "ACCEPTED" || c.Command.Result != "REJECTED" {
				t.Fatal("accepted task cancelled")
			}
		} else if c.State != "CANCELLED" || a.Command.Result != "REJECTED" {
			t.Fatal("cancel tombstone reopened")
		}
		original := a.Command.ID
		in.Permit.Expires = time.Now().Add(-time.Hour)
		replay, e := s.Accept(context.Background(), in)
		if e != nil || replay.Command.ID != original {
			t.Fatal("original outcome lost", e)
		}
		var count int
		_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM fulfillment.tasks WHERE order_id=$1`, in.Inputs.OrderID).Scan(&count)
		if count > 1 {
			t.Fatal("duplicate task")
		}
		in.Command.Hash = d.Hash("changed")
		if _, e = s.Accept(context.Background(), in); e == nil {
			t.Fatal("hash conflict accepted")
		}
	}
}
func TestCancelBeforeAbsentTaskAndEverAccepted(t *testing.T) {
	s, in := fixture(t)
	ctx := context.Background()
	c := d.Command{Caller: "service:order-service", Action: "CancelTask", OrderID: in.Inputs.OrderID, Generation: 1, Key: "cancel-" + uuid.NewString(), Hash: d.Hash("cancel")}
	cancel, e := s.Cancel(ctx, c, "hold")
	if e != nil || cancel.Reference == nil {
		t.Fatal(e)
	}
	a, e := s.Accept(ctx, in)
	if e != nil || a.State != "CANCELLED" {
		t.Fatal("late accept", e)
	}
	filtered, e := s.Decision(ctx, c.OrderID, 1, c.Key, c.Action)
	if e != nil || filtered.Command.ID != cancel.Command.ID {
		t.Fatal("original command not recoverable", e)
	}
	in.Command.Generation = 2
	in.Command.Key = "accept-" + uuid.NewString()
	in.Inputs.Generation = 2
	in.Stock.Generation = 2
	in.Permit.Generation = 2
	in.Permit.Key = in.Command.Key
	in.Command.Hash = d.Hash(in.Inputs)
	task := mustTask(t, s, in)
	c.Generation = 3
	c.Key = "cancel-" + uuid.NewString()
	if _, e = s.Cancel(ctx, c, "new generation"); e == nil {
		t.Fatal("fabricated stopped proof after accepted")
	}
	if _, e = s.Decision(ctx, c.OrderID, 3, "", ""); e == nil {
		t.Fatal("unexpected generation3 tombstone")
	}
	if task.Generation != 2 {
		t.Fatal("wrong generation")
	}
}
func TestWorkflowAssignmentsBlocksAndSealAtomicity(t *testing.T) {
	s, in := fixture(t)
	task := mustTask(t, s, in)
	p := principal(task)
	task = change(t, s, p, task, d.Mutation{Action: "AssignTask"})
	task = change(t, s, p, task, d.Mutation{Action: "StartTask"})
	task = change(t, s, p, task, d.Mutation{Action: "BlockTask", ReasonCode: "BOX_DAMAGED", Reason: "replace packaging"})
	first := task.Blocks[0].ID
	task = change(t, s, p, task, d.Mutation{Action: "BlockTask", ReasonCode: "CHECK_REQUIRED", Reason: "second independent block"})
	task = change(t, s, p, task, d.Mutation{Action: "ResolveTaskBlock", Child: first, Reason: "box replaced"})
	if task.State != "BLOCKED" {
		t.Fatal("cleared unrelated block")
	}
	task = change(t, s, p, task, d.Mutation{Action: "ResolveTaskBlock", Child: task.Blocks[0].ID, Reason: "checked"})
	for _, a := range task.Allocations {
		m := d.Mutation{Action: "SetPickedQuantity", Expected: task.Version, Child: a.ID, Counted: a.Allocated, AllocationVersion: a.Version}
		key := "pick-" + uuid.NewString()
		got, e := s.Mutate(context.Background(), p, task.ID, key, d.Hash(m), m, nil)
		if e != nil {
			t.Fatal(e)
		}
		replay, e := s.Mutate(context.Background(), p, task.ID, key, d.Hash(m), m, nil)
		if e != nil || replay.Version != got.Version {
			t.Fatal("retry incremented pick", e)
		}
		task = got
	}
	proof := validation(task)
	m := d.Mutation{Action: "SealTask", Expected: task.Version, PackageID: task.Package.ID, Seal: "SEAL-TEST-" + strings.ToUpper(uuid.NewString()), Parcel: &d.Parcel{Weight: 1000, Length: 20, Width: 20, Height: 20, Fragile: true}}
	if _, e := s.Mutate(context.Background(), p, task.ID, "seal-"+uuid.NewString(), d.Hash(m), m, proof); e == nil {
		t.Fatal("missing checklist sealed")
	}
	for _, c := range task.Checklist {
		task = change(t, s, p, task, d.Mutation{Action: "ConfirmChecklist", Child: c.Code, Result: "PASS"})
	}
	m.Expected = task.Version
	proof = validation(task)
	key := "seal-" + uuid.NewString()
	out, e := s.Mutate(context.Background(), p, task.ID, key, d.Hash(m), m, proof)
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.Mutate(context.Background(), p, task.ID, key, d.Hash(m), m, nil)
	if e != nil || again.Version != out.Version {
		t.Fatal("lost seal response not recoverable", e)
	}
	rev, e := s.Revision(context.Background(), out.ID, out.Version)
	if e != nil || rev.Kind != "PACKING_SEALED" {
		t.Fatal(e)
	}
	old, e := s.Revision(context.Background(), out.ID, 1)
	if e != nil || old.Kind != "PACKING_ACCEPTED" || old.Task.State != "READY" {
		t.Fatal("latest substituted for exact proof", e)
	}
	for table, want := range map[string]int{"packages": 1, "task_revisions": int(out.Version), "task_history": int(out.Version)} {
		var n int
		if e = pool.QueryRow(context.Background(), fmt.Sprintf("SELECT count(*) FROM fulfillment.%s WHERE task_id=$1", table), out.ID).Scan(&n); e != nil || n != want {
			t.Fatalf("atomic %s: %d != %d %v", table, n, want, e)
		}
	}
	if _, e = pool.Exec(context.Background(), `UPDATE fulfillment.task_revisions SET snapshot='{}' WHERE task_id=$1`, out.ID); e == nil {
		t.Fatal("immutable proof modified")
	}
}
func validation(t d.Task) *d.Validation {
	return &d.Validation{Allowed: true, ID: uuid.NewString(), Contract: d.Contract, OrderID: t.OrderID, TaskID: t.ID, ReservationID: t.Inputs.ReservationID, Version: t.Stock.Version, Generation: t.Generation, Digest: t.Digest, Warehouse: t.Warehouse, Observed: time.Now().UTC(), Until: time.Now().UTC().Add(5 * time.Second), Slots: t.Stock.Slots}
}
func TestWrongProofAndScopeFailClosed(t *testing.T) {
	s, in := fixture(t)
	in.Stock.Digest = d.Hash("wrong")
	if _, e := s.Accept(context.Background(), in); e == nil {
		t.Fatal("wrong digest accepted")
	}
	var count int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM fulfillment.tasks WHERE order_id=$1`, in.Inputs.OrderID).Scan(&count)
	if count != 0 {
		t.Fatal("failed admission leaked task")
	}
	in.Stock.Digest = in.Inputs.Digest
	task := mustTask(t, s, in)
	p := principal(task)
	p.Warehouses = []string{"OTHER"}
	_, e := s.Mutate(context.Background(), p, task.ID, "assign-"+uuid.NewString(), d.Hash("assign"), d.Mutation{Action: "AssignTask", Expected: 1}, nil)
	var de *d.Error
	if !errors.As(e, &de) || de.HTTP != 403 {
		t.Fatal("cross warehouse mutation", e)
	}
}
func TestHTTPJWTVersionAndPayloadIsolation(t *testing.T) {
	s, in := fixture(t)
	task := mustTask(t, s, in)
	p := principal(task)
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	svc := &application.Service{Store: s}
	router := (transport.HTTP{Service: svc, Auth: transport.Auth{Key: &key.PublicKey, Issuer: "test-issuer"}}).Router()
	grants := make([]string, 0, len(p.Permissions))
	for _, permission := range p.Permissions {
		if p.Global {
			grants = append(grants, permission+"#GLOBAL")
		}
		for _, warehouse := range p.Warehouses {
			grants = append(grants, permission+"#WAREHOUSE#"+warehouse)
		}
	}
	claims := transport.Claims{UserID: p.ID, PermissionGrants: grants, RegisteredClaims: jwt.RegisteredClaims{Subject: p.ID, Issuer: "test-issuer", Audience: jwt.ClaimStrings{"omama-api"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
	token, e := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if e != nil {
		t.Fatal(e)
	}
	call := func(method, path, body, auth, etag string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", auth)
		req.Header.Set("If-Match", etag)
		req.Header.Set("Idempotency-Key", "http-command-0000001")
		req.Header.Set("X-User-ID", uuid.NewString())
		req.Header.Set("X-Role", "ADMIN")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	path := "/api/v1/admin/fulfillment/tasks/" + task.ID
	if w := call("GET", path, "", "", ""); w.Code != 401 {
		t.Fatal("forged headers accepted", w.Code)
	}
	if w := call("POST", path+"/assignments", `{}`, "Bearer "+token, ""); w.Code != 428 {
		t.Fatal("missing precondition", w.Body.String())
	}
	if w := call("POST", path+"/assignments", `{"role":"ADMIN"}`, "Bearer "+token, `"1"`); w.Code != 400 {
		t.Fatal("body identity injected", w.Code)
	}
	if w := call("POST", path+"/assignments", `{}`, "Bearer "+token, `"1"`); w.Code != 200 {
		t.Fatal("assign", w.Code, w.Body.String())
	}
	w := call("GET", path, "", "Bearer "+token, "")
	if w.Code != 200 || w.Header().Get("ETag") != `"2"` {
		t.Fatal("GET ETag", w.Code, w.Body.String())
	}
	_ = os.MkdirAll("results", 0755)
	_ = os.WriteFile("results/task.json", w.Body.Bytes(), 0644)
	var payload map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &payload)
	if payload["input_snapshot"] != nil || payload["stock"] != nil {
		t.Fatal("private snapshot leaked")
	}
	claims.Audience = jwt.ClaimStrings{"order-api"}
	bad, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if w := call("GET", path, "", "Bearer "+bad, ""); w.Code != 401 {
		t.Fatal("wrong audience accepted")
	}
}
