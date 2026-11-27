package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/domain"
)

func TestReceivingContractExamples(t *testing.T) {
	repo := newHTTPRepository()
	supplier, _ := domain.NewSupplier("SUP-01", "Supplier", "COMPANY", "", "", "", "")
	material, _ := domain.NewRawMaterial("RICE", "Rice", "KG", "")
	supplier.CreatedAt, supplier.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	material.CreatedAt, material.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	offer, _ := domain.NewSupplierMaterialOffer(supplier.ID, material.ID, "", "KG", 25000, "VND", "1", 1)
	repo.suppliers[supplier.ID], repo.materials[material.ID] = supplier, material
	repo.offers[supplier.ID] = []domain.SupplierMaterialOffer{offer}
	handler := newHandler(repo)
	base := "/api/v1/admin/procurement/"
	check := func(method, path, body, user, role, key string, status int) map[string]any {
		t.Helper()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		if role != "" {
			request.Header.Set("Authorization", "Bearer "+employeeToken(user, role))
		}
		request.Header.Set("Idempotency-Key", key)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != status {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, recorder.Code, status, recorder.Body.String())
		}
		var value map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		if status >= 400 && (value["requestId"] == nil || value["requestId"] != recorder.Header().Get("X-Request-Id")) {
			t.Fatalf("error lacks correlation: %v", value)
		}
		fixture := map[string]any{"method": method, "path": path, "status": status, "body": value}
		if body != "" {
			var input any
			if json.Unmarshal([]byte(body), &input) == nil {
				fixture["request"] = input
			}
		}
		encoded, _ := json.Marshal(fixture)
		fmt.Println("PROCUREMENT_HTTP=" + string(encoded))
		return value
	}
	poBody := fmt.Sprintf(`{"supplier_id":%q,"lines":[{"material_id":%q,"quantity":"10"}]}`, supplier.ID, material.ID)
	poPath, poKey := base+"purchase-orders", "purchase-order-create-0001"
	check("POST", poPath, poBody, "buyer", "", poKey, 401)
	check("POST", poPath, poBody, "buyer", "PROCUREMENT_VIEWER", poKey, 403)
	check("POST", poPath, poBody, "buyer", "ADMIN", "", 400)
	check("POST", poPath, `{`, "buyer", "ADMIN", poKey, 400)
	created := check("POST", poPath, poBody, "buyer", "ADMIN", poKey, 201)
	replayed := check("POST", poPath, poBody, "buyer", "ADMIN", poKey, 201)
	if created["id"] != replayed["id"] || len(repo.orders) != 1 {
		t.Fatal("duplicate purchase order")
	}
	check("POST", poPath, strings.Replace(poBody, `"10"`, `"11"`, 1), "buyer", "ADMIN", poKey, 409)
	poID := uuid.MustParse(created["id"].(string))
	poDetail := poPath + "/" + poID.String()
	check("GET", poDetail, "", "viewer", "PROCUREMENT_VIEWER", "", 200)
	check("GET", poDetail, "", "viewer", "", "", 401)
	check("GET", poDetail, "", "viewer", "WAREHOUSE_STAFF", "", 403)
	check("GET", poPath+"/bad-id", "", "viewer", "PROCUREMENT_VIEWER", "", 400)
	check("GET", poPath+"/"+uuid.NewString(), "", "viewer", "PROCUREMENT_VIEWER", "", 404)
	// Lifecycle approval remains private in this slice; prepare a sent fixture.
	po := repo.orders[poID]
	unsentReceipt := fmt.Sprintf(`{"purchase_order_id":%q,"warehouse_code":"WH-01","lines":[{"purchase_order_line_id":%q,"quantity_received":"2"}]}`, poID, po.Lines[0].ID)
	check("POST", base+"goods-receipts", unsentReceipt, "receiver", "WAREHOUSE_STAFF", "unsent-receipt-create-0001", 409)
	po.Status = domain.PurchaseOrderSent
	repo.orders[poID] = po
	receiptBody := fmt.Sprintf(`{"purchase_order_id":%q,"warehouse_code":"WH-01","lines":[{"purchase_order_line_id":%q,"quantity_received":"2"}]}`, poID, po.Lines[0].ID)
	receiptPath, receiptKey := base+"goods-receipts", "goods-receipt-create-0001"
	check("POST", receiptPath, receiptBody, "receiver", "", receiptKey, 401)
	check("POST", receiptPath, receiptBody, "receiver", "PROCUREMENT_VIEWER", receiptKey, 403)
	check("POST", receiptPath, receiptBody, "receiver", "WAREHOUSE_STAFF", "", 400)
	check("POST", receiptPath, strings.Replace(receiptBody, "WH-01", "WH-02", 1), "receiver", "WAREHOUSE_STAFF", receiptKey, 403)
	rc := check("POST", receiptPath, receiptBody, "receiver", "WAREHOUSE_STAFF", receiptKey, 201)
	rr := check("POST", receiptPath, receiptBody, "receiver", "WAREHOUSE_STAFF", receiptKey, 201)
	if rc["id"] != rr["id"] || len(repo.receipts) != 1 {
		t.Fatal("duplicate receipt")
	}
	check("POST", receiptPath, strings.Replace(receiptBody, `"2"`, `"3"`, 1), "receiver", "WAREHOUSE_STAFF", receiptKey, 409)
	rid := uuid.MustParse(rc["id"].(string))
	detail := receiptPath + "/" + rid.String()
	check("GET", detail, "", "receiver", "WAREHOUSE_STAFF", "", 200)
	check("GET", detail, "", "viewer", "PROCUREMENT_VIEWER", "", 200)
	check("GET", detail, "", "receiver", "", "", 401)
	check("GET", detail, "", "receiver", "UNASSIGNED", "", 403)
	check("GET", receiptPath+"/bad-id", "", "receiver", "WAREHOUSE_STAFF", "", 400)
	check("GET", receiptPath+"/"+uuid.NewString(), "", "receiver", "WAREHOUSE_STAFF", "", 404)
	foreign := repo.receipts[rid]
	foreign.ID, foreign.WarehouseCode = uuid.New(), "WH-02"
	repo.receipts[foreign.ID] = foreign
	check("GET", receiptPath+"/"+foreign.ID.String(), "", "receiver", "WAREHOUSE_MIXED", "", 403)
	lineID := repo.receipts[rid].Lines[0].ID
	inspection := fmt.Sprintf(`{"lines":[{"goods_receipt_line_id":%q,"quantity_accepted":"1.5","quantity_rejected":"0.5","inspection_notes":"Damaged portion"}]}`, lineID)
	inspectPath, inspectKey := detail+"/inspect", "receipt-inspection-0001"
	check("POST", inspectPath, inspection, "receiver", "", inspectKey, 401)
	check("POST", inspectPath, inspection, "receiver", "PROCUREMENT_VIEWER", inspectKey, 403)
	check("POST", inspectPath, inspection, "receiver", "WAREHOUSE_STAFF", "", 400)
	check("POST", receiptPath+"/bad-id/inspect", inspection, "receiver", "WAREHOUSE_STAFF", inspectKey, 400)
	check("POST", receiptPath+"/"+uuid.NewString()+"/inspect", inspection, "receiver", "WAREHOUSE_STAFF", inspectKey, 404)
	check("POST", receiptPath+"/"+foreign.ID.String()+"/inspect", inspection, "receiver", "WAREHOUSE_MIXED", inspectKey, 403)
	first := check("POST", inspectPath, inspection, "receiver", "WAREHOUSE_STAFF", inspectKey, 200)
	retry := check("POST", inspectPath, inspection, "receiver", "WAREHOUSE_STAFF", inspectKey, 200)
	if first["version"] != retry["version"] {
		t.Fatal("retry updated receipt")
	}
	check("POST", inspectPath, strings.Replace(inspection, "Damaged portion", "Changed evidence", 1), "receiver", "WAREHOUSE_STAFF", inspectKey, 409)
	check("POST", inspectPath, inspection, "other-inspector", "WAREHOUSE_STAFF", inspectKey, 409)
	check("POST", inspectPath, inspection, "receiver", "WAREHOUSE_STAFF", "receipt-inspection-0002", 409)
	check("POST", receiptPath, receiptBody, "receiver", "WAREHOUSE_STAFF", receiptKey, 201)
}
