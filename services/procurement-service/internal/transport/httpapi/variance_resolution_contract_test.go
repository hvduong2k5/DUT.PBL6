package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
	"github.com/omamx/procurement-service/internal/transport/httpapi"
)

type failedVarianceLookup struct{ *httpRepository }

func (r failedVarianceLookup) GetGoodsReceiptResolutionResult(context.Context, string, string) (domain.VarianceResolutionResult, error) {
	return domain.VarianceResolutionResult{}, errors.New("private variance database error")
}

func TestVarianceResolutionContractExamples(t *testing.T) {
	repo := newHTTPRepository()
	now := time.Now().UTC()
	id, lineID, firstID, secondID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	receipt := domain.GoodsReceipt{ID: id, Number: "GR-TEST", PurchaseOrderID: uuid.New(), PurchaseOrderNumber: "PO-TEST", SupplierID: uuid.New(), SupplierName: "Supplier", WarehouseCode: "WH-01", Status: domain.GoodsReceiptInspected, ReceivedBy: "receiver", Version: 2, CreatedAt: now, UpdatedAt: now,
		Lines:     []domain.GoodsReceiptLine{{ID: lineID, PurchaseOrderLineID: uuid.New(), MaterialID: uuid.New(), MaterialCode: "RICE", MaterialName: "Rice", Unit: "KG", ExpectedOpenQuantity: "1", QuantityReceived: "2", QuantityAccepted: "1", QuantityRejected: "1", QualityStatus: domain.GoodsReceiptQualityStatus("PARTIALLY_ACCEPTED")}},
		Variances: []domain.GoodsReceiptVariance{{ID: firstID, GoodsReceiptLineID: lineID, Type: domain.GoodsReceiptVarianceOverReceived, ExpectedQuantity: "1", ActualQuantity: "2", DifferenceQuantity: "1", Reason: "Over delivery", Status: "OPEN"}, {ID: secondID, GoodsReceiptLineID: lineID, Type: domain.GoodsReceiptVarianceQualityReject, ExpectedQuantity: "2", ActualQuantity: "1", DifferenceQuantity: "1", Reason: "Quality rejection", Status: "OPEN"}}}
	repo.receipts[id] = receipt
	handler := newHandler(repo)
	path := func(r, v uuid.UUID) string {
		return "/api/v1/admin/procurement/goods-receipts/" + r.String() + "/variances/" + v.String() + "/resolve"
	}
	check := func(path, body, role, key string, status int) map[string]any {
		t.Helper()
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		if role != "" {
			r.Header.Set("Authorization", "Bearer "+employeeToken("operator", role))
		}
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s got %d want %d: %s", path, w.Code, status, w.Body.String())
		}
		var value map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		if status >= 400 && value["requestId"] != w.Header().Get("X-Request-Id") {
			t.Fatal("missing request correlation")
		}
		if strings.Contains(w.Body.String(), "private variance") {
			t.Fatal("database detail leaked")
		}
		fixture := map[string]any{"method": "POST", "path": path, "status": status, "body": value}
		var input any
		if json.Unmarshal([]byte(body), &input) == nil {
			fixture["request"] = input
		}
		encoded, _ := json.Marshal(fixture)
		fmt.Println("PROCUREMENT_HTTP=" + string(encoded))
		return value
	}
	body := `{"expected_version":2,"decision":"ACCEPT_VARIANCE","note":"Documented exception"}`
	key := "variance-http-key-0001"
	check(path(id, firstID), body, "", key, 401)
	for _, role := range []string{"SUPPLY_MANAGER", "WAREHOUSE_STAFF", "VARIANCE_FOREIGN"} {
		check(path(id, firstID), body, role, key, 403)
	}
	check(path(id, firstID), body, "VARIANCE_STAFF", "", 400)
	for _, invalid := range []string{`{}`, `{`, `{"expected_version":0,"decision":"ACCEPT_VARIANCE","note":"Evidence"}`, `{"expected_version":2,"decision":"UNKNOWN","note":"Evidence"}`, `{"expected_version":2,"decision":"ACCEPT_VARIANCE","note":" "}`, `{"expected_version":2,"decision":"ACCEPT_VARIANCE","note":"Evidence","unknown":true}`} {
		check(path(id, firstID), invalid, "VARIANCE_STAFF", key, 400)
	}
	check(strings.Replace(path(id, firstID), id.String(), "bad-id", 1), body, "VARIANCE_STAFF", key, 400)
	check(strings.Replace(path(id, firstID), firstID.String(), "bad-id", 1), body, "VARIANCE_STAFF", key, 400)
	check(path(uuid.New(), firstID), body, "VARIANCE_STAFF", key, 404)
	check(path(id, uuid.New()), body, "VARIANCE_STAFF", key, 404)
	first := check(path(id, firstID), body, "VARIANCE_STAFF", key, 200)
	if got := check(path(id, firstID), body, "VARIANCE_STAFF", key, 200); !reflect.DeepEqual(first, got) {
		t.Fatal("duplicate resolution changed evidence")
	}
	check(path(id, firstID), strings.Replace(body, "ACCEPT_VARIANCE", "REJECT_VARIANCE", 1), "VARIANCE_STAFF", key, 409)
	check(path(id, firstID), strings.Replace(body, "Documented exception", "Changed note", 1), "VARIANCE_STAFF", key, 409)
	check(path(id, secondID), body, "VARIANCE_STAFF", key, 409)
	check(path(id, secondID), body, "VARIANCE_STAFF", key+"-stale", 409)
	secondBody := `{"expected_version":3,"decision":"REJECT_VARIANCE","note":"Keep quarantined"}`
	check(path(id, secondID), secondBody, "PROCUREMENT_MANAGER", "variance-http-key-0002", 200)
	if got := check(path(id, firstID), body, "VARIANCE_STAFF", key, 200); !reflect.DeepEqual(first, got) {
		t.Fatal("old result was not frozen")
	}
	check(path(id, firstID), body, "VARIANCE_FOREIGN", key, 403)
	check(path(id, firstID), `{"expected_version":4,"decision":"ACCEPT_VARIANCE","note":"Documented exception"}`, "VARIANCE_STAFF", key+"-resolved", 409)
	if !reflect.DeepEqual(repo.receipts[id].Lines, receipt.Lines) || repo.receipts[id].Version != 4 {
		t.Fatal("decision altered stock quantities or duplicated version")
	}
	handler, err := httpapi.NewRouter(application.NewService(failedVarianceLookup{repo}), testIdentityConfig(), repo.Ping)
	if err != nil {
		t.Fatal(err)
	}
	check(path(id, secondID), secondBody, "VARIANCE_STAFF", "variance-http-failure-0001", 500)
}
