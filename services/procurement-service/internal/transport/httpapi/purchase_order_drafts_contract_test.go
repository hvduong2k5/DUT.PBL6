package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
	"github.com/omamx/procurement-service/internal/transport/httpapi"
)

func (r *httpRepository) ListPurchaseOrders(_ context.Context, supplier uuid.UUID, status domain.PurchaseOrderStatus, limit, offset int) ([]domain.PurchaseOrder, error) {
	items := []domain.PurchaseOrder{}
	for _, p := range r.orders {
		if (supplier == uuid.Nil || p.SupplierID == supplier) && (status == "" || p.Status == status) {
			items = append(items, p)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID.String() < items[j].ID.String()
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if offset >= len(items) {
		return []domain.PurchaseOrder{}, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], nil
}

type poListRepository struct {
	*httpRepository
	fail, forceNil bool
}

func (r *poListRepository) ListPurchaseOrders(ctx context.Context, supplier uuid.UUID, status domain.PurchaseOrderStatus, limit, offset int) ([]domain.PurchaseOrder, error) {
	if r.fail {
		return nil, errors.New("private list database failure")
	}
	if r.forceNil {
		return nil, nil
	}
	return r.httpRepository.ListPurchaseOrders(ctx, supplier, status, limit, offset)
}

func TestPODraftAndListContractExamples(t *testing.T) {
	ctx := context.Background()
	baseRepo := newHTTPRepository()
	repo := &poListRepository{httpRepository: baseRepo}
	service := application.NewService(repo)
	supplier, _ := domain.NewSupplier("DRAFT-SUP", "Supplier", "COMPANY", "", "", "", "")
	material, _ := domain.NewRawMaterial("DRAFT-RICE", "Rice", "KG", "")
	offer, _ := domain.NewSupplierMaterialOffer(supplier.ID, material.ID, "", "KG", 25000, "VND", "1", 1)
	repo.suppliers[supplier.ID], repo.materials[material.ID] = supplier, material
	repo.offers[supplier.ID] = []domain.SupplierMaterialOffer{offer}
	creator := uuid.NewSHA1(uuid.NameSpaceOID, []byte("buyer")).String()
	p, err := service.CreatePurchaseOrder(ctx, application.PurchaseOrderRequest{SupplierID: supplier.ID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: material.ID, Quantity: "10"}}}, creator)
	if err != nil {
		t.Fatal(err)
	}
	p.Revision = 2
	repo.orders[p.ID] = p
	handler, err := httpapi.NewRouter(service, testIdentityConfig(), repo.Ping)
	if err != nil {
		t.Fatal(err)
	}
	root := "/api/v1/admin/procurement/purchase-orders"
	detail := root + "/" + p.ID.String()
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
			t.Fatalf("%s %s: %d want %d: %s", method, path, w.Code, status, w.Body.String())
		}
		var value map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		if status >= 400 && value["requestId"] != w.Header().Get("X-Request-Id") {
			t.Fatal("missing correlation")
		}
		if strings.Contains(w.Body.String(), "private ") {
			t.Fatal("private failure leaked")
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
	check("GET", root, "", "viewer", "", "", 401)
	check("GET", root, "", "viewer", "WAREHOUSE_VIEWER", "", 403)
	check("GET", root, "", "viewer", "PO_EDITOR", "", 403)
	page := check("GET", root, "", "viewer", "PROCUREMENT_VIEWER", "", 200)
	if len(page["items"].([]any)) != 1 {
		t.Fatal("missing PO list result")
	}
	for _, query := range []string{"?limit=0", "?limit=101", "?offset=-1", "?supplier_id=bad-id", "?supplier_id=" + uuid.Nil.String(), "?status=UNKNOWN"} {
		check("GET", root+query, "", "viewer", "PROCUREMENT_VIEWER", "", 400)
	}
	for _, query := range []string{"?offset=99", "?supplier_id=" + uuid.NewString(), "?status=SENT"} {
		got := check("GET", root+query, "", "viewer", "PROCUREMENT_VIEWER", "", 200)
		if len(got["items"].([]any)) != 0 {
			t.Fatal("empty filtered page")
		}
	}
	got := check("GET", root+"?supplier_id="+supplier.ID.String()+"&status=draft&limit=1", "", "viewer", "PROCUREMENT_VIEWER", "", 200)
	if len(got["items"].([]any)) != 1 {
		t.Fatal("combined filters")
	}
	repo.forceNil = true
	check("GET", root, "", "viewer", "PROCUREMENT_VIEWER", "", 200)
	repo.forceNil = false
	body := fmt.Sprintf(`{"expected_version":%d,"supplier_id":%q,"notes":"Correct draft","lines":[{"material_id":%q,"quantity":"8"}]}`, p.Version, supplier.ID.String(), material.ID.String())
	key := "draft-edit-contract-0001"
	check("PUT", detail, body, "editor", "", key, 401)
	check("PUT", detail, body, "editor", "PROCUREMENT_VIEWER", key, 403)
	check("PUT", detail, body, "editor", "PO_EDITOR", "", 400)
	check("PUT", detail, `{}`, "editor", "PO_EDITOR", key, 400)
	check("PUT", detail, strings.Replace(body, `"expected_version":1`, `"expected_version":0`, 1), "editor", "PO_EDITOR", key, 400)
	check("PUT", detail, strings.Replace(body, `"quantity":"8"`, `"quantity":"0"`, 1), "editor", "PO_EDITOR", key, 400)
	check("PUT", detail, strings.Replace(body, `"notes":"Correct draft"`, `"reason":"unknown"`, 1), "editor", "PO_EDITOR", key, 400)
	check("PUT", root+"/bad-id", body, "editor", "PO_EDITOR", key, 400)
	check("PUT", root+"/"+uuid.NewString(), body, "editor", "PO_EDITOR", key, 404)
	first := check("PUT", detail, body, "editor", "PO_EDITOR", key, 200)
	retry := check("PUT", detail, body, "editor", "PO_EDITOR", key, 200)
	if !reflect.DeepEqual(first, retry) || first["revision"] != float64(2) || first["created_by"] != creator {
		t.Fatal("edit changed revision/creator or replay")
	}
	check("PUT", detail, strings.Replace(body, "Correct draft", "Changed", 1), "editor", "PO_EDITOR", key, 409)
	check("PUT", detail, body, "editor", "PO_EDITOR", key+"-stale", 409)
	current, err := service.SubmitPurchaseOrder(ctx, p.ID, creator)
	if err != nil {
		t.Fatal(err)
	}
	repo.offers[supplier.ID] = nil
	if got := check("PUT", detail, body, "editor", "PO_EDITOR", key, 200); !reflect.DeepEqual(first, got) {
		t.Fatal("replay rebuilt draft after submit")
	}
	check("PUT", detail, strings.Replace(body, `"expected_version":1`, fmt.Sprintf(`"expected_version":%d`, current.Version), 1), "editor", "PO_EDITOR", key+"-submitted", 409)
	repo.fail = true
	check("GET", root, "", "viewer", "PROCUREMENT_VIEWER", "", 500)
	repo.fail = false
	handler, err = httpapi.NewRouter(application.NewService(failedPOActionLookup{baseRepo}), testIdentityConfig(), baseRepo.Ping)
	if err != nil {
		t.Fatal(err)
	}
	check("PUT", detail, body, "editor", "PO_EDITOR", key+"-failure", 500)
}
