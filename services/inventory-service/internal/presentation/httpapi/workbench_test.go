package httpapi

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dut-pbl6/inventory-service/internal/testdb"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestWorkbenchUsesPermissionScopeIntersection(t *testing.T) {
	db := testdb.Open(t)
	a, b := uuid.NewString(), uuid.NewString()
	actor := uuid.New()
	if _, err := db.Exec(`INSERT INTO inventory_items(sku,warehouse_id) VALUES('SKU-A',$1),('SKU-B',$2)`, a, b); err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	base := ReceiptHandler{DB: db, Key: &key.PublicKey, Issuer: "https://issuer.example", Audience: "omama-api"}
	handler := WorkbenchHandler{ReceiptHandler: base}
	var revision int64
	if err := db.QueryRow(`SELECT inventory_version FROM inventory_items WHERE sku='SKU-A'`).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	received, err := base.receive(t.Context(), actor, false, map[string]bool{a: true}, "workbench-receipt-test", receiptRequest{
		SKU: "SKU-A", BatchMode: "NEW", BatchCode: "SOURCE-LOT", SupplierID: uuid.NewString(), Reference: "WORKBENCH-FIXTURE", Reason: "TEST",
		Manufactured: time.Now().AddDate(0, 0, -1).Format(time.DateOnly), Expires: time.Now().AddDate(0, 2, 0).Format(time.DateOnly), Quantity: 5, Revision: &revision,
	})
	if err != nil {
		t.Fatal(err)
	}
	sign := func(grants []string) string {
		claims := jwt.MapClaims{"iss": base.Issuer, "aud": base.Audience, "sub": "fixture-staff", "user_id": actor.String(), "principal_type": "EMPLOYEE", "iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(time.Minute).Unix(), "permission_grants": grants}
		token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	for _, tc := range []struct {
		name          string
		grants        []string
		status, count int
	}{
		{"warehouse A", []string{"INVENTORY_VIEW#WAREHOUSE#" + a, "WAREHOUSE_WORKBENCH_VIEW#WAREHOUSE#" + a}, 200, 1},
		{"mixed global and warehouse", []string{"INVENTORY_VIEW#GLOBAL", "WAREHOUSE_WORKBENCH_VIEW#WAREHOUSE#" + a}, 200, 1},
		{"global", []string{"INVENTORY_VIEW#GLOBAL", "WAREHOUSE_WORKBENCH_VIEW#GLOBAL"}, 200, 2},
		{"cross product forbidden", []string{"INVENTORY_VIEW#WAREHOUSE#" + a, "WAREHOUSE_WORKBENCH_VIEW#WAREHOUSE#" + b}, 403, 0},
		{"missing view", []string{"WAREHOUSE_WORKBENCH_VIEW#GLOBAL"}, 403, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", WorkbenchPath, nil)
			r.Header.Set("Authorization", "Bearer "+sign(tc.grants))
			r.Header.Set("X-User-Role", "ADMIN")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tc.status || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal(w.Code, w.Body.String())
			}
			if tc.status == 200 {
				var result struct {
					SKUs []workbenchSKU `json:"skus"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || len(result.SKUs) != tc.count {
					t.Fatal("wrong warehouse result", err, w.Body.String())
				}
				if result.SKUs[0].ID != "SKU-A" || len(result.SKUs[0].Batches) != 1 || result.SKUs[0].Batches[0].ID != received.BatchID || result.SKUs[0].Balance.OnHand != 5 {
					t.Fatal("receipt batch context lost", w.Body.String())
				}
				if tc.name == "warehouse A" {
					if dir := os.Getenv("INVENTORY_WORKBENCH_ARTIFACT_DIR"); dir != "" {
						if err := os.WriteFile(filepath.Join(dir, "workbench.json"), w.Body.Bytes(), 0600); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
		})
	}
}
