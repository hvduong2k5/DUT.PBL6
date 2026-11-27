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

type failedPORevisionList struct{ *httpRepository }

func (r failedPORevisionList) ListPurchaseOrderRevisions(context.Context, uuid.UUID, int, int) ([]domain.PurchaseOrderRevision, error) {
	return nil, errors.New("private revision database failure")
}

func TestPORevisionContractExamples(t *testing.T) {
	ctx := context.Background()
	repo := newHTTPRepository()
	supplier, _ := domain.NewSupplier("REV-SUP", "Supplier", "COMPANY", "", "", "", "")
	material, _ := domain.NewRawMaterial("REV-RICE", "Rice", "KG", "")
	offer, _ := domain.NewSupplierMaterialOffer(supplier.ID, material.ID, "", "KG", 25000, "VND", "1", 1)
	repo.suppliers[supplier.ID], repo.materials[material.ID] = supplier, material
	repo.offers[supplier.ID] = []domain.SupplierMaterialOffer{offer}
	service := application.NewService(repo)
	creator := uuid.NewSHA1(uuid.NameSpaceOID, []byte("buyer")).String()
	po, err := service.CreatePurchaseOrder(ctx, application.PurchaseOrderRequest{SupplierID: supplier.ID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: material.ID, Quantity: "10"}}}, creator)
	if err != nil {
		t.Fatal(err)
	}
	handler := newHandler(repo)
	path := "/api/v1/admin/procurement/purchase-orders/" + po.ID.String() + "/revisions"
	check := func(method, path, body, user, role, key string, status int) map[string]any {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if role != "" {
			r.Header.Set("Authorization", "Bearer "+employeeToken(user, role))
		}
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, w.Code, status, w.Body.String())
		}
		var value map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		if status >= 400 && value["requestId"] != w.Header().Get("X-Request-Id") {
			t.Fatal("missing request correlation")
		}
		if strings.Contains(w.Body.String(), "private revision") || strings.Contains(w.Body.String(), "private database") {
			t.Fatal("database detail leaked")
		}
		fixture := map[string]any{"method": method, "path": path, "status": status, "body": value}
		var input any
		if json.Unmarshal([]byte(body), &input) == nil {
			fixture["request"] = input
		}
		encoded, _ := json.Marshal(fixture)
		fmt.Println("PROCUREMENT_HTTP=" + string(encoded))
		return value
	}
	check("GET", path, "", "viewer", "", "", 401)
	check("GET", path, "", "viewer", "WAREHOUSE_VIEWER", "", 403)
	empty := check("GET", path, "", "viewer", "PROCUREMENT_VIEWER", "", 200)
	if len(empty["items"].([]any)) != 0 {
		t.Fatal("history must exclude live draft")
	}
	check("GET", path+"?limit=bad", "", "viewer", "PROCUREMENT_VIEWER", "", 400)
	check("GET", strings.Replace(path, po.ID.String(), "bad-id", 1), "", "viewer", "PROCUREMENT_VIEWER", "", 400)
	check("GET", strings.Replace(path, po.ID.String(), uuid.NewString(), 1), "", "viewer", "PROCUREMENT_VIEWER", "", 404)
	if po, err = service.SubmitPurchaseOrder(ctx, po.ID, creator); err != nil {
		t.Fatal(err)
	}
	if po, err = service.ApprovePurchaseOrder(ctx, po.ID, "checker"); err != nil {
		t.Fatal(err)
	}
	if po, err = service.MarkPurchaseOrderSent(ctx, po.ID, creator); err != nil {
		t.Fatal(err)
	}
	if po, err = service.RecordSupplierResponse(ctx, po.ID, creator, domain.SupplierPOChangeRequested, "Need less"); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"expected_version":%d,"supplier_id":%q,"reason":"Need less","notes":"Revision two","lines":[{"material_id":%q,"quantity":"8"}]}`, po.Version, supplier.ID.String(), material.ID.String())
	key := "http-revision-command-0001"
	check("POST", path, body, "reviser", "", key, 401)
	check("POST", path, body, "reviser", "PROCUREMENT_VIEWER", key, 403)
	check("POST", path, body, "reviser", "SUPPLY_MANAGER", "", 400)
	check("POST", path, `{}`, "reviser", "SUPPLY_MANAGER", key, 400)
	check("POST", path, strings.Replace(body, `"reason":"Need less"`, `"reason":" "`, 1), "reviser", "SUPPLY_MANAGER", key, 400)
	check("POST", path, strings.Replace(body, fmt.Sprintf(`"expected_version":%d`, po.Version), `"expected_version":0`, 1), "reviser", "SUPPLY_MANAGER", key, 400)
	check("POST", path, strings.Replace(body, supplier.ID.String(), uuid.NewString(), 1), "reviser", "SUPPLY_MANAGER", key, 400)
	check("POST", path, strings.Replace(body, `"quantity":"8"`, `"quantity":"0"`, 1), "reviser", "SUPPLY_MANAGER", key, 400)
	check("POST", path, strings.Replace(body, `"notes":"Revision two"`, `"notes":"Revision two","unknown":true`, 1), "reviser", "SUPPLY_MANAGER", key, 400)
	check("POST", strings.Replace(path, po.ID.String(), uuid.NewString(), 1), body, "reviser", "SUPPLY_MANAGER", key, 404)
	first := check("POST", path, body, "reviser", "SUPPLY_MANAGER", key, 200)
	retry := check("POST", path, body, "reviser", "SUPPLY_MANAGER", key, 200)
	if !reflect.DeepEqual(first, retry) || first["revision"] != float64(2) || repo.orders[po.ID].Version != po.Version+1 {
		t.Fatal("revision replay duplicated draft")
	}
	history := check("GET", path, "", "viewer", "PROCUREMENT_VIEWER", "", 200)
	if len(history["items"].([]any)) != 1 || history["items"].([]any)[0].(map[string]any)["snapshot"].(map[string]any)["revision"] != float64(1) {
		t.Fatal("missing original archive")
	}
	check("GET", path+"?offset=99", "", "viewer", "PROCUREMENT_VIEWER", "", 200)
	check("POST", path, strings.Replace(body, "Revision two", "Changed", 1), "reviser", "SUPPLY_MANAGER", key, 409)
	check("POST", path, body, "reviser", "SUPPLY_MANAGER", key+"-stale", 409)
	// Advancing the PO and removing offers must not alter replayed draft prices.
	current := repo.orders[po.ID]
	if current, err = service.SubmitPurchaseOrder(ctx, current.ID, current.CreatedBy); err != nil {
		t.Fatal(err)
	}
	repo.offers[supplier.ID] = nil
	if got := check("POST", path, body, "reviser", "SUPPLY_MANAGER", key, 200); !reflect.DeepEqual(first, got) {
		t.Fatal("replay repriced or advanced revision")
	}
	if _, err := service.ApprovePurchaseOrder(ctx, current.ID, current.CreatedBy); !errors.Is(err, domain.ErrConflict) {
		t.Fatal("new reviser self-approved")
	}
	handler, err = httpapi.NewRouter(application.NewService(failedPOActionLookup{repo}), testIdentityConfig(), repo.Ping)
	if err != nil {
		t.Fatal(err)
	}
	check("POST", path, body, "reviser", "SUPPLY_MANAGER", key+"-failure", 500)
	handler, err = httpapi.NewRouter(application.NewService(failedPORevisionList{repo}), testIdentityConfig(), repo.Ping)
	if err != nil {
		t.Fatal(err)
	}
	check("GET", path, "", "viewer", "PROCUREMENT_VIEWER", "", 500)
}
