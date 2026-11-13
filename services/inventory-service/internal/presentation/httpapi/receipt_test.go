package httpapi

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestReceiptAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("INVENTORY_RECEIPT_TEST_DATABASE")
	if dsn == "" {
		t.Skip("isolated Inventory PostgreSQL fixture required")
	}
	database, err := url.Parse(dsn)
	if err != nil || !strings.HasPrefix(database.Path, "/pbl_inventory_receipt") {
		t.Fatal("receipt tests require a disposable pbl_inventory_receipt database")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	files, err := filepath.Glob("../../../migrations/*.up.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("migrations: %v %v", files, err)
	}
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(raw)); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}

	const warehouse = "a1111111-1111-1111-1111-111111111111"
	if _, err = db.Exec("INSERT INTO inventory_items(sku,warehouse_id) VALUES('MX-F18',$1)", warehouse); err != nil {
		t.Fatal(err)
	}
	var revision int64
	if err = db.QueryRow("SELECT inventory_version FROM inventory_items WHERE sku='MX-F18'").Scan(&revision); err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	claims := jwt.MapClaims{
		"iss": "https://test-issuer.example", "sub": "receipt-staff", "aud": "omama-api",
		"exp": time.Now().Add(time.Minute).Unix(), "principal_type": "EMPLOYEE",
		"user_id":           "e5f9ea69-bdc3-4ee7-9076-96c327335410",
		"permission_grants": []string{"INVENTORY_RECEIPT_CREATE#WAREHOUSE#" + warehouse},
	}
	sign := func() string {
		raw, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return "Bearer " + raw
	}
	handler := ReceiptHandler{DB: db, Key: &key.PublicKey, Issuer: "https://test-issuer.example", Audience: "omama-api"}
	body := receiptRequest{SKU: "MX-F18", BatchMode: "NEW", BatchCode: "F18-LOT",
		SupplierID:   "b2222222-2222-2222-2222-222222222222",
		Manufactured: time.Now().AddDate(0, 0, -1).Format(time.DateOnly), Expires: time.Now().AddDate(0, 3, 0).Format(time.DateOnly),
		Reference: "PRODUCTION-F18-001", Quantity: 25, Revision: &revision}
	call := func(token string, req receiptRequest, key string, expected int) string {
		raw, err := json.Marshal(req)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest("POST", ReceiptPath, bytes.NewReader(raw))
		request.Header.Set("Idempotency-Key", key)
		request.Header.Set("Authorization", token)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != expected {
			t.Fatalf("got %d want %d: %s", recorder.Code, expected, recorder.Body)
		}
		return recorder.Body.String()
	}
	const idem = "receipt-integration-0001"
	call("", body, idem, 401)
	claims["permission_grants"] = []string{"INVENTORY_RECEIPT_CREATE#WAREHOUSE#wrong"}
	call(sign(), body, idem, 403)
	claims["permission_grants"] = []string{"INVENTORY_RECEIPT_CREATE#WAREHOUSE#" + warehouse}
	token := sign()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan *httptest.ResponseRecorder, 2)
	for range 2 {
		go func() {
			req := httptest.NewRequest("POST", ReceiptPath, bytes.NewReader(payload))
			req.Header.Set("Authorization", token)
			req.Header.Set("Idempotency-Key", idem)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			results <- recorder
		}()
	}
	first, second := <-results, <-results
	if first.Code != 200 || second.Code != 200 || first.Body.String() != second.Body.String() {
		t.Fatalf("concurrent receipt replay differs: %d %s / %d %s", first.Code, first.Body, second.Code, second.Body)
	}
	body.Quantity++
	call(token, body, idem, 409)
	call(token, body, "receipt-integration-0002", 409)
	var itemQty, batchQty, sellable int
	err = db.QueryRow(`SELECT i.physical_qty,b.physical_qty,inventory_sellable_quantity(i.sku,now())
		FROM inventory_items i JOIN batches b ON b.sku=i.sku WHERE i.sku='MX-F18'`).Scan(&itemQty, &batchQty, &sellable)
	if err != nil || itemQty != 25 || batchQty != 25 || sellable != 25 {
		t.Fatalf("balances: %d %d %d %v", itemQty, batchQty, sellable, err)
	}
	var projected int
	err = db.QueryRow(`SELECT (payload->>'sellable_quantity')::int FROM outbox_events
		WHERE aggregate_id='MX-F18' ORDER BY (payload->>'inventory_version')::int DESC LIMIT 1`).Scan(&projected)
	if err != nil || projected != 25 {
		t.Fatalf("projection: %d %v", projected, err)
	}
}
