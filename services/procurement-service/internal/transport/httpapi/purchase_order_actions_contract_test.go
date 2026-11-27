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

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
	"github.com/omamx/procurement-service/internal/transport/httpapi"
)

type failedPOActionLookup struct{ *httpRepository }

func (r failedPOActionLookup) GetPurchaseOrderActionResult(context.Context, string, string) (domain.PurchaseOrderActionResult, error) {
	return domain.PurchaseOrderActionResult{}, errors.New("private database error")
}

func TestPOActionsContractExamples(t *testing.T) {
	repo := newHTTPRepository()
	supplier, _ := domain.NewSupplier("PO-SUP", "Supplier", "COMPANY", "", "", "", "")
	material, _ := domain.NewRawMaterial("RICE", "Rice", "KG", "")
	offer, _ := domain.NewSupplierMaterialOffer(supplier.ID, material.ID, "", "KG", 25000, "VND", "1", 1)
	repo.suppliers[supplier.ID], repo.materials[material.ID] = supplier, material
	repo.offers[supplier.ID] = []domain.SupplierMaterialOffer{offer}
	service := application.NewService(repo)
	actor := uuid.NewSHA1(uuid.NameSpaceOID, []byte("buyer")).String()
	create := func() domain.PurchaseOrder {
		t.Helper()
		po, err := service.CreatePurchaseOrder(context.Background(), application.PurchaseOrderRequest{SupplierID: supplier.ID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: material.ID, Quantity: "10"}}}, actor)
		if err != nil {
			t.Fatal(err)
		}
		return po
	}
	handler := newHandler(repo)
	base := "/api/v1/admin/procurement/purchase-orders/"
	check := func(path, body, user, role, key string, status int) map[string]any {
		t.Helper()
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		if role != "" {
			r.Header.Set("Authorization", "Bearer "+employeeToken(user, role))
		}
		r.Header.Set("Idempotency-Key", key)
		r.Header.Set("X-User-Role", "ADMIN")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s: got %d want %d: %s", path, w.Code, status, w.Body.String())
		}
		var value map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		if status >= 400 && value["requestId"] != w.Header().Get("X-Request-Id") {
			t.Fatal("missing correlation")
		}
		if strings.Contains(w.Body.String(), "private database") {
			t.Fatal("dependency detail leaked")
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
	po := create()
	var submitted map[string]any
	var submitBody string
	for _, tc := range []struct{ action, user, role, denied string }{
		{"submit", "buyer", "SUPPLY_MANAGER", "PO_APPROVER"},
		{"approve", "checker", "PO_APPROVER", "SUPPLY_MANAGER"},
		{"send", "buyer", "SUPPLY_MANAGER", "PO_APPROVER"},
	} {
		path, key := base+po.ID.String()+"/"+tc.action, "po-command-"+tc.action+"-0001"
		before := repo.orders[po.ID].Version
		body := fmt.Sprintf(`{"expected_version":%d}`, before)
		check(path, body, tc.user, "", key, 401)
		check(path, body, tc.user, tc.denied, key, 403)
		check(path, body, tc.user, tc.role, "", 400)
		check(path, `{}`, tc.user, tc.role, key, 400)
		check(path, `{`, tc.user, tc.role, key, 400)
		check(path, fmt.Sprintf(`{"expected_version":%d,"reason":""}`, before), tc.user, tc.role, key, 400)
		check(path, fmt.Sprintf(`{"expected_version":%d,"unexpected":true}`, before), tc.user, tc.role, key, 400)
		check(base+"bad-id/"+tc.action, body, tc.user, tc.role, key, 400)
		check(base+uuid.NewString()+"/"+tc.action, body, tc.user, tc.role, key+"-missing", 404)
		if tc.action == "approve" {
			check(path, body, "buyer", "PO_APPROVER", "po-self-approval-0001", 409)
		}
		first := check(path, body, tc.user, tc.role, key, 200)
		retry := check(path, body, tc.user, tc.role, key, 200)
		if !reflect.DeepEqual(first, retry) || repo.orders[po.ID].Version != before+1 {
			t.Fatal("action replay changed snapshot/state")
		}
		check(path, fmt.Sprintf(`{"expected_version":%d}`, before+1), tc.user, tc.role, key, 409)
		check(path, body, tc.user, tc.role, key+"-stale", 409)
		if tc.action == "submit" {
			submitted, submitBody = first, body
		}
	}
	old := check(base+po.ID.String()+"/submit", submitBody, "buyer", "SUPPLY_MANAGER", "po-command-submit-0001", 200)
	if !reflect.DeepEqual(submitted, old) || repo.orders[po.ID].Status != domain.PurchaseOrderSent {
		t.Fatal("old retry affected later lifecycle")
	}
	check(base+po.ID.String()+"/send", fmt.Sprintf(`{"expected_version":%d}`, repo.orders[po.ID].Version), "buyer", "SUPPLY_MANAGER", "fresh-send-after-sent-0001", 409)
	other := create()
	check(base+other.ID.String()+"/submit", submitBody, "buyer", "SUPPLY_MANAGER", "po-command-submit-0001", 409)
	other, err := service.SubmitPurchaseOrder(context.Background(), other.ID, actor)
	if err != nil {
		t.Fatal(err)
	}
	rejectPath, key := base+other.ID.String()+"/reject", "po-command-reject-0001"
	body := fmt.Sprintf(`{"expected_version":%d,"reason":"Needs correction"}`, other.Version)
	check(rejectPath, body, "checker", "", key, 401)
	check(rejectPath, body, "checker", "SUPPLY_MANAGER", key, 403)
	check(rejectPath, body, "checker", "PO_APPROVER", "", 400)
	check(rejectPath, fmt.Sprintf(`{"expected_version":%d}`, other.Version), "checker", "PO_APPROVER", key, 400)
	check(rejectPath, body, "buyer", "PO_APPROVER", "po-self-rejection-0001", 409)
	first := check(rejectPath, body, "checker", "PO_APPROVER", key, 200)
	retry := check(rejectPath, body, "checker", "PO_APPROVER", key, 200)
	if !reflect.DeepEqual(first, retry) {
		t.Fatal("reject replay changed snapshot")
	}
	check(rejectPath, strings.Replace(body, "Needs correction", "Different reason", 1), "checker", "PO_APPROVER", key, 409)
	// New terminal and supplier-evidence endpoints use the same durable namespace.
	sentPO := func() domain.PurchaseOrder {
		t.Helper()
		p := create()
		var err error
		if p, err = service.SubmitPurchaseOrder(context.Background(), p.ID, actor); err != nil {
			t.Fatal(err)
		}
		if p, err = service.ApprovePurchaseOrder(context.Background(), p.ID, "checker"); err != nil {
			t.Fatal(err)
		}
		if p, err = service.MarkPurchaseOrderSent(context.Background(), p.ID, actor); err != nil {
			t.Fatal(err)
		}
		return p
	}
	for _, action := range []string{"supplier-response", "cancel", "close"} {
		p := create()
		if action != "cancel" {
			p = sentPO()
		}
		path, key := base+p.ID.String()+"/"+action, "terminal-po-"+action+"-0001"
		body := fmt.Sprintf(`{"expected_version":%d,"reason":"No more required"}`, p.Version)
		if action == "supplier-response" {
			body = fmt.Sprintf(`{"expected_version":%d,"response":"ACKNOWLEDGED","note":"Confirmed"}`, p.Version)
		}
		check(path, body, "buyer", "", key, 401)
		check(path, body, "buyer", "PO_APPROVER", key, 403)
		check(path, body, "buyer", "SUPPLY_MANAGER", "", 400)
		check(path, `{}`, "buyer", "SUPPLY_MANAGER", key, 400)
		check(base+"bad-id/"+action, body, "buyer", "SUPPLY_MANAGER", key, 400)
		check(base+uuid.NewString()+"/"+action, body, "buyer", "SUPPLY_MANAGER", key+"-missing", 404)
		first := check(path, body, "buyer", "SUPPLY_MANAGER", key, 200)
		retry := check(path, body, "buyer", "SUPPLY_MANAGER", key, 200)
		if !reflect.DeepEqual(first, retry) || repo.orders[p.ID].Version != p.Version+1 {
			t.Fatal("terminal replay changed state")
		}
		check(path, strings.Replace(body, "No more required", "Changed", 1)+" ", "buyer", "SUPPLY_MANAGER", key+"-stale", 409)
		changed := strings.Replace(body, fmt.Sprintf(`"expected_version":%d`, p.Version), fmt.Sprintf(`"expected_version":%d`, p.Version+1), 1)
		check(path, changed, "buyer", "SUPPLY_MANAGER", key, 409)
		if action == "supplier-response" {
			_, err := service.ExecutePurchaseOrderAction(context.Background(), p.ID, actor, "close-after-ack-0001", "CLOSE", p.Version+1, "Stop remaining")
			if err != nil {
				t.Fatal(err)
			}
			if got := check(path, body, "buyer", "SUPPLY_MANAGER", key, 200); !reflect.DeepEqual(first, got) {
				t.Fatal("old acknowledgement was not frozen")
			}
		}
	}
	for _, response := range []string{"CHANGE_REQUESTED", "DECLINED"} {
		p := sentPO()
		path, key := base+p.ID.String()+"/supplier-response", "supplier-"+response+"-0001"
		check(path, fmt.Sprintf(`{"expected_version":%d,"response":%q}`, p.Version, response), "buyer", "SUPPLY_MANAGER", key, 400)
		check(path, fmt.Sprintf(`{"expected_version":%d,"response":"UNKNOWN"}`, p.Version), "buyer", "SUPPLY_MANAGER", key, 400)
		body := fmt.Sprintf(`{"expected_version":%d,"response":%q,"note":" Delivery changed "}`, p.Version, response)
		first := check(path, body, "buyer", "SUPPLY_MANAGER", key, 200)
		if got := check(path, strings.Replace(body, " Delivery changed ", "Delivery changed", 1), "buyer", "SUPPLY_MANAGER", key, 200); !reflect.DeepEqual(first, got) {
			t.Fatal("normalized note did not replay")
		}
		check(path, strings.Replace(body, " Delivery changed ", "Other note", 1), "buyer", "SUPPLY_MANAGER", key, 409)
	}
	partial := sentPO()
	if err := partial.RecordReceived(partial.Lines[0].ID, "2"); err != nil {
		t.Fatal(err)
	}
	repo.orders[partial.ID] = partial
	varianceReceiptID := uuid.New()
	repo.receipts[varianceReceiptID] = domain.GoodsReceipt{ID: varianceReceiptID, PurchaseOrderID: partial.ID, Variances: []domain.GoodsReceiptVariance{{Status: "OPEN"}}}
	path := base + partial.ID.String() + "/close"
	body = fmt.Sprintf(`{"expected_version":%d,"reason":"Stop remainder"}`, partial.Version)
	check(path, body, "buyer", "SUPPLY_MANAGER", "partial-close-evidence-0001", 409)
	delete(repo.receipts, varianceReceiptID)
	closed := check(path, body, "buyer", "SUPPLY_MANAGER", "partial-close-evidence-0001", 200)
	if closed["lines"].([]any)[0].(map[string]any)["received_quantity"] != partial.Lines[0].ReceivedQuantity {
		t.Fatal("close erased received quantity")
	}
	check(base+partial.ID.String()+"/cancel", fmt.Sprintf(`{"expected_version":%d,"reason":"Invalid"}`, partial.Version+1), "buyer", "SUPPLY_MANAGER", "cancel-closed-po-0001", 409)
	handler, err = httpapi.NewRouter(application.NewService(failedPOActionLookup{repo}), testIdentityConfig(), repo.Ping)
	if err != nil {
		t.Fatal(err)
	}
	check(base+other.ID.String()+"/submit", `{"expected_version":1}`, "buyer", "SUPPLY_MANAGER", "po-command-db-failure-0001", 500)
	for _, action := range []string{"cancel", "close", "supplier-response"} {
		body := `{"expected_version":1,"reason":"Failure"}`
		if action == "supplier-response" {
			body = `{"expected_version":1,"response":"ACKNOWLEDGED"}`
		}
		check(base+other.ID.String()+"/"+action, body, "buyer", "SUPPLY_MANAGER", "po-dependency-failure-"+action, 500)
	}
}
