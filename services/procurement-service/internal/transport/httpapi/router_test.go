package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
	"github.com/omamx/procurement-service/internal/transport/httpapi"
)

type httpRepository struct {
	application.Repository
	suppliers map[uuid.UUID]domain.Supplier
	materials map[uuid.UUID]domain.RawMaterial
	offers    map[uuid.UUID][]domain.SupplierMaterialOffer
	orders    map[uuid.UUID]domain.PurchaseOrder
	receipts  map[uuid.UUID]domain.GoodsReceipt
}

func newHTTPRepository() *httpRepository {
	return &httpRepository{
		suppliers: make(map[uuid.UUID]domain.Supplier), materials: make(map[uuid.UUID]domain.RawMaterial),
		offers: make(map[uuid.UUID][]domain.SupplierMaterialOffer), orders: make(map[uuid.UUID]domain.PurchaseOrder), receipts: make(map[uuid.UUID]domain.GoodsReceipt),
	}
}

func (r *httpRepository) Ping(context.Context) error { return nil }
func (r *httpRepository) CreateSupplier(_ context.Context, supplier domain.Supplier) (domain.Supplier, error) {
	r.suppliers[supplier.ID] = supplier
	return supplier, nil
}
func (r *httpRepository) GetSupplier(_ context.Context, id uuid.UUID) (domain.Supplier, error) {
	item, ok := r.suppliers[id]
	if !ok {
		return domain.Supplier{}, domain.ErrNotFound
	}
	return item, nil
}
func (r *httpRepository) GetMaterial(_ context.Context, id uuid.UUID) (domain.RawMaterial, error) {
	item, ok := r.materials[id]
	if !ok {
		return domain.RawMaterial{}, domain.ErrNotFound
	}
	return item, nil
}
func (r *httpRepository) ListSupplierOffers(_ context.Context, id uuid.UUID) ([]domain.SupplierMaterialOffer, error) {
	return r.offers[id], nil
}
func (r *httpRepository) CreatePurchaseOrder(_ context.Context, order domain.PurchaseOrder) (domain.PurchaseOrder, error) {
	r.orders[order.ID] = order
	return order, nil
}
func (r *httpRepository) GetPurchaseOrder(_ context.Context, id uuid.UUID) (domain.PurchaseOrder, error) {
	order, ok := r.orders[id]
	if !ok {
		return domain.PurchaseOrder{}, domain.ErrNotFound
	}
	return order, nil
}
func (r *httpRepository) SavePurchaseOrder(_ context.Context, order domain.PurchaseOrder, expectedVersion int64, _ domain.PurchaseOrderStatus, _, _, _ string) (domain.PurchaseOrder, error) {
	current, ok := r.orders[order.ID]
	if !ok {
		return domain.PurchaseOrder{}, domain.ErrNotFound
	}
	if current.Version != expectedVersion {
		return domain.PurchaseOrder{}, domain.ErrConflict
	}
	r.orders[order.ID] = order
	return order, nil
}
func (r *httpRepository) CreateGoodsReceipt(_ context.Context, receipt domain.GoodsReceipt) (domain.GoodsReceipt, error) {
	r.receipts[receipt.ID] = receipt
	return receipt, nil
}
func (r *httpRepository) GetGoodsReceipt(_ context.Context, id uuid.UUID) (domain.GoodsReceipt, error) {
	receipt, ok := r.receipts[id]
	if !ok {
		return domain.GoodsReceipt{}, domain.ErrNotFound
	}
	return receipt, nil
}
func (r *httpRepository) InspectGoodsReceipt(_ context.Context, receipt domain.GoodsReceipt, expected int64, _ string) (domain.GoodsReceipt, error) {
	current, ok := r.receipts[receipt.ID]
	if !ok {
		return domain.GoodsReceipt{}, domain.ErrNotFound
	}
	if current.Version != expected {
		return domain.GoodsReceipt{}, domain.ErrConflict
	}
	r.receipts[receipt.ID] = receipt
	return receipt, nil
}

func newHandler(repo *httpRepository) http.Handler {
	return httpapi.NewRouter(application.NewService(repo), "test-token", repo.Ping)
}

func performRequest(handler http.Handler, method, path, body, user, role string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if user != "" {
		request.Header.Set("X-User-ID", user)
	}
	if role != "" {
		request.Header.Set("X-User-Role", role)
	}
	request.Header.Set("X-Internal-Token", "test-token")
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestAdminSupplierAPIRequiresInternalStaffContext(t *testing.T) {
	repo := newHTTPRepository()
	handler := newHandler(repo)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/procurement/suppliers", strings.NewReader(`{"code":"SUP-1","name":"Supplier","supplier_type":"COMPANY"}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}

	denied := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/suppliers", `{"code":"SUP-1","name":"Supplier","supplier_type":"COMPANY"}`, "staff", "WAREHOUSE_STAFF")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("warehouse staff status = %d, want 403", denied.Code)
	}
}

func TestCreateSupplierAndStrictJSONValidation(t *testing.T) {
	repo := newHTTPRepository()
	handler := newHandler(repo)
	created := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/suppliers", `{"code":"SUP-1","name":"Supplier","supplier_type":"COMPANY"}`, "staff", "SUPPLY_MANAGER")
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", created.Code, created.Body.String())
	}
	var supplier domain.Supplier
	if err := json.Unmarshal(created.Body.Bytes(), &supplier); err != nil {
		t.Fatal(err)
	}
	if supplier.Code != "SUP-1" || supplier.Status != domain.SupplierActive {
		t.Fatalf("unexpected response: %+v", supplier)
	}
	invalid := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/suppliers", `{"code":"SUP-2","name":"Supplier","supplier_type":"COMPANY","unexpected":true}`, "staff", "SUPPLY_MANAGER")
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d, want 400", invalid.Code)
	}
}

func TestPurchaseOrderHTTPWorkflowAndSelfApprovalGuard(t *testing.T) {
	repo := newHTTPRepository()
	supplier, err := domain.NewSupplier("SUP-1", "Supplier", "COOPERATIVE", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	material, err := domain.NewRawMaterial("SESAME", "Sesame", "KG", "")
	if err != nil {
		t.Fatal(err)
	}
	offer, err := domain.NewSupplierMaterialOffer(supplier.ID, material.ID, "", "KG", 10000, "VND", "1", 2)
	if err != nil {
		t.Fatal(err)
	}
	repo.suppliers[supplier.ID] = supplier
	repo.materials[material.ID] = material
	repo.offers[supplier.ID] = []domain.SupplierMaterialOffer{offer}
	handler := newHandler(repo)

	body := `{"supplier_id":"` + supplier.ID.String() + `","notes":"routine restock","lines":[{"material_id":"` + material.ID.String() + `","quantity":"2.5"}]}`
	created := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/purchase-orders", body, "creator", "SUPPLY_MANAGER")
	if created.Code != http.StatusCreated {
		t.Fatalf("create PO status = %d, body=%s", created.Code, created.Body.String())
	}
	var order domain.PurchaseOrder
	if err := json.Unmarshal(created.Body.Bytes(), &order); err != nil {
		t.Fatal(err)
	}
	if order.Status != domain.PurchaseOrderDraft || order.TotalAmountMinor != 25000 {
		t.Fatalf("unexpected created PO: %+v", order)
	}

	submitted := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/purchase-orders/"+order.ID.String()+"/submit", "", "creator", "SUPPLY_MANAGER")
	if submitted.Code != http.StatusOK {
		t.Fatalf("submit status = %d, body=%s", submitted.Code, submitted.Body.String())
	}
	selfApproval := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/purchase-orders/"+order.ID.String()+"/approve", "", "creator", "SUPPLY_MANAGER")
	if selfApproval.Code != http.StatusConflict {
		t.Fatalf("self approval status = %d, want 409", selfApproval.Code)
	}
	approved := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/purchase-orders/"+order.ID.String()+"/approve", "", "approver", "PROCUREMENT_MANAGER")
	if approved.Code != http.StatusOK {
		t.Fatalf("approval status = %d, body=%s", approved.Code, approved.Body.String())
	}
}

func TestWarehouseStaffCanCreateGoodsReceiptButNotManageSuppliers(t *testing.T) {
	repo := newHTTPRepository()
	supplier, _ := domain.NewSupplier("SUP-GR", "Supplier", "COMPANY", "", "", "", "")
	material, _ := domain.NewRawMaterial("RICE", "Rice", "KG", "")
	po, _ := domain.NewPurchaseOrder("PO-GR", supplier.ID, supplier.Name, "VND", "", "buyer", nil, []domain.PurchaseOrderLineInput{{MaterialID: material.ID, MaterialCode: material.Code, MaterialName: material.Name, Quantity: "5", Unit: "KG", UnitPriceMinor: 100}})
	po.Status = domain.PurchaseOrderSent
	repo.orders[po.ID] = po
	handler := newHandler(repo)
	body := `{"purchase_order_id":"` + po.ID.String() + `","warehouse_code":"WH-01","lines":[{"purchase_order_line_id":"` + po.Lines[0].ID.String() + `","quantity_received":"2"}]}`
	created := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/goods-receipts", body, "warehouse-1", "WAREHOUSE_STAFF")
	if created.Code != http.StatusCreated {
		t.Fatalf("create goods receipt status=%d body=%s", created.Code, created.Body.String())
	}
	denied := performRequest(handler, http.MethodGet, "/api/v1/admin/procurement/suppliers", "", "warehouse-1", "WAREHOUSE_STAFF")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("warehouse supplier access status=%d, want 403", denied.Code)
	}
}
