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
	handler, err = httpapi.NewRouter(application.NewService(failedPOActionLookup{repo}), testIdentityConfig(), repo.Ping)
	if err != nil {
		t.Fatal(err)
	}
	check(base+other.ID.String()+"/submit", `{"expected_version":1}`, "buyer", "SUPPLY_MANAGER", "po-command-db-failure-0001", 500)
}
