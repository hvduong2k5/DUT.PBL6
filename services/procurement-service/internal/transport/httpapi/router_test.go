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
	contracts map[uuid.UUID]domain.SupplierContract
	revisions map[uuid.UUID][]domain.PurchaseOrderRevision
}

func newHTTPRepository() *httpRepository {
	return &httpRepository{
		suppliers: make(map[uuid.UUID]domain.Supplier), materials: make(map[uuid.UUID]domain.RawMaterial),
		offers: make(map[uuid.UUID][]domain.SupplierMaterialOffer), orders: make(map[uuid.UUID]domain.PurchaseOrder), receipts: make(map[uuid.UUID]domain.GoodsReceipt), contracts: make(map[uuid.UUID]domain.SupplierContract), revisions: make(map[uuid.UUID][]domain.PurchaseOrderRevision),
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
func (r *httpRepository) UpdateMaterial(_ context.Context, id uuid.UUID, item domain.RawMaterial) (domain.RawMaterial, error) {
	current, ok := r.materials[id]
	if !ok {
		return domain.RawMaterial{}, domain.ErrNotFound
	}
	item.ID = id
	item.Status = current.Status
	item.PreferredSupplierID = current.PreferredSupplierID
	r.materials[id] = item
	return item, nil
}
func (r *httpRepository) SetPreferredSupplier(_ context.Context, materialID, supplierID uuid.UUID) (domain.RawMaterial, error) {
	item, ok := r.materials[materialID]
	if !ok {
		return domain.RawMaterial{}, domain.ErrNotFound
	}
	if supplierID == uuid.Nil {
		item.PreferredSupplierID = nil
	} else {
		item.PreferredSupplierID = &supplierID
	}
	r.materials[materialID] = item
	return item, nil
}
func (r *httpRepository) ListSupplierOffers(_ context.Context, id uuid.UUID) ([]domain.SupplierMaterialOffer, error) {
	return r.offers[id], nil
}
func (r *httpRepository) ListMaterialOffers(_ context.Context, id uuid.UUID) ([]domain.SupplierMaterialOffer, error) {
	items := []domain.SupplierMaterialOffer{}
	for _, offers := range r.offers {
		for _, offer := range offers {
			if offer.MaterialID == id {
				items = append(items, offer)
			}
		}
	}
	return items, nil
}
func (r *httpRepository) CreateOffer(_ context.Context, item domain.SupplierMaterialOffer) (domain.SupplierMaterialOffer, error) {
	r.offers[item.SupplierID] = append(r.offers[item.SupplierID], item)
	return item, nil
}
func (r *httpRepository) SetOfferActive(_ context.Context, supplierID, offerID uuid.UUID, active bool) (domain.SupplierMaterialOffer, error) {
	for i, item := range r.offers[supplierID] {
		if item.ID == offerID {
			item.Active = active
			r.offers[supplierID][i] = item
			return item, nil
		}
	}
	return domain.SupplierMaterialOffer{}, domain.ErrNotFound
}
func (r *httpRepository) CreateSupplierContract(_ context.Context, item domain.SupplierContract) (domain.SupplierContract, error) {
	item.Revision = 1
	r.contracts[item.ID] = item
	return item, nil
}
func (r *httpRepository) GetSupplierContract(_ context.Context, id uuid.UUID) (domain.SupplierContract, error) {
	item, ok := r.contracts[id]
	if !ok {
		return domain.SupplierContract{}, domain.ErrNotFound
	}
	return item, nil
}
func (r *httpRepository) ListSupplierContracts(_ context.Context, id uuid.UUID, _, _ int) ([]domain.SupplierContract, error) {
	items := []domain.SupplierContract{}
	for _, item := range r.contracts {
		if id == uuid.Nil || item.SupplierID == id {
			items = append(items, item)
		}
	}
	return items, nil
}
func (r *httpRepository) SetSupplierContractStatus(_ context.Context, id uuid.UUID, status domain.SupplierContractStatus, actor string) (domain.SupplierContract, error) {
	item, ok := r.contracts[id]
	if !ok {
		return domain.SupplierContract{}, domain.ErrNotFound
	}
	item.Status = status
	r.contracts[id] = item
	return item, nil
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
func (r *httpRepository) RevisePurchaseOrder(_ context.Context, order domain.PurchaseOrder, expectedVersion int64, actor, reason string) (domain.PurchaseOrder, error) {
	current, ok := r.orders[order.ID]
	if !ok {
		return domain.PurchaseOrder{}, domain.ErrNotFound
	}
	if current.Version != expectedVersion {
		return domain.PurchaseOrder{}, domain.ErrConflict
	}
	r.revisions[order.ID] = append(r.revisions[order.ID], domain.PurchaseOrderRevision{Revision: current.Revision, Snapshot: current, ChangedBy: actor, ChangeReason: reason})
	r.orders[order.ID] = order
	return order, nil
}
func (r *httpRepository) ListPurchaseOrderRevisions(_ context.Context, id uuid.UUID, limit, offset int) ([]domain.PurchaseOrderRevision, error) {
	return r.revisions[id], nil
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
	created := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/suppliers", `{"code":"SUP-1","name":"Supplier","supplier_type":"COMPANY","legal_identifier":" ab-123 "}`, "staff", "SUPPLY_MANAGER")
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", created.Code, created.Body.String())
	}
	var supplier domain.Supplier
	if err := json.Unmarshal(created.Body.Bytes(), &supplier); err != nil {
		t.Fatal(err)
	}
	if supplier.Code != "SUP-1" || supplier.Status != domain.SupplierActive || supplier.LegalIdentifier != "AB-123" {
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

func TestPurchaseOrderSupplierResponseRevisionAndSendRetryHTTP(t *testing.T) {
	repo := newHTTPRepository()
	supplier, _ := domain.NewSupplier("SUP-REV", "Supplier", "COMPANY", "", "", "", "")
	material, _ := domain.NewRawMaterial("SESAME", "Sesame", "KG", "")
	offer, _ := domain.NewSupplierMaterialOffer(supplier.ID, material.ID, "", "KG", 10000, "VND", "1", 2)
	repo.suppliers[supplier.ID] = supplier
	repo.materials[material.ID] = material
	repo.offers[supplier.ID] = []domain.SupplierMaterialOffer{offer}
	handler := newHandler(repo)
	body := `{"supplier_id":"` + supplier.ID.String() + `","lines":[{"material_id":"` + material.ID.String() + `","quantity":"1"}]}`
	created := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/purchase-orders", body, "buyer", "SUPPLY_MANAGER")
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var order domain.PurchaseOrder
	if err := json.Unmarshal(created.Body.Bytes(), &order); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/submit", "/approve", "/send"} {
		rec := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/purchase-orders/"+order.ID.String()+path, "", "staff", "PROCUREMENT_MANAGER")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
	sentVersion := repo.orders[order.ID].Version
	retry := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/purchase-orders/"+order.ID.String()+"/send", "", "staff", "PROCUREMENT_MANAGER")
	if retry.Code != http.StatusOK || repo.orders[order.ID].Version != sentVersion {
		t.Fatalf("send retry status=%d version=%d want %d", retry.Code, repo.orders[order.ID].Version, sentVersion)
	}
	response := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/purchase-orders/"+order.ID.String()+"/supplier-response", `{"response":"CHANGE_REQUESTED","note":"Change delivery quantity"}`, "buyer", "SUPPLY_MANAGER")
	if response.Code != http.StatusOK {
		t.Fatalf("supplier response status=%d body=%s", response.Code, response.Body.String())
	}
	revisionBody := `{"supplier_id":"` + supplier.ID.String() + `","notes":"revision 2","reason":"supplier requested quantity change","lines":[{"material_id":"` + material.ID.String() + `","quantity":"2"}]}`
	revised := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/purchase-orders/"+order.ID.String()+"/revisions", revisionBody, "buyer-2", "SUPPLY_MANAGER")
	if revised.Code != http.StatusOK {
		t.Fatalf("revision status=%d body=%s", revised.Code, revised.Body.String())
	}
	var current domain.PurchaseOrder
	if err := json.Unmarshal(revised.Body.Bytes(), &current); err != nil {
		t.Fatal(err)
	}
	if current.Revision != 2 || current.Status != domain.PurchaseOrderDraft {
		t.Fatalf("unexpected current revision: %+v", current)
	}
	history := performRequest(handler, http.MethodGet, "/api/v1/admin/procurement/purchase-orders/"+order.ID.String()+"/revisions", "", "buyer", "SUPPLY_MANAGER")
	if history.Code != http.StatusOK || !strings.Contains(history.Body.String(), `"revision":1`) {
		t.Fatalf("revision history response status=%d body=%s", history.Code, history.Body.String())
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

func TestSupplierContractsRequireSensitiveRoleAndCreateDraftRevision(t *testing.T) {
	repo := newHTTPRepository()
	supplier, _ := domain.NewSupplier("SUP-CON", "Supplier", "COMPANY", "", "", "", "")
	repo.suppliers[supplier.ID] = supplier
	handler := newHandler(repo)
	body := `{"contract_number":"AGR-01","scope":"RAW_MATERIALS","payment_terms":"NET 30","document_file_id":"file-abc","effective_from":"2026-11-01"}`
	denied := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/suppliers/"+supplier.ID.String()+"/contracts", body, "supply", "SUPPLY_MANAGER")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("supply manager contract status=%d, want 403", denied.Code)
	}
	created := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/suppliers/"+supplier.ID.String()+"/contracts", body, "procurement", "PROCUREMENT_MANAGER")
	if created.Code != http.StatusCreated {
		t.Fatalf("create contract status=%d body=%s", created.Code, created.Body.String())
	}
	var contract domain.SupplierContract
	if err := json.Unmarshal(created.Body.Bytes(), &contract); err != nil {
		t.Fatal(err)
	}
	if contract.Status != domain.SupplierContractDraft || contract.Revision != 1 || contract.DocumentFileID != "file-abc" {
		t.Fatalf("unexpected contract response: %+v", contract)
	}
}

func TestUpdateMaterialRoutePreservesStatus(t *testing.T) {
	repo := newHTTPRepository()
	material, _ := domain.NewRawMaterial("SESAME", "Sesame", "KG", "")
	material.Status = domain.MaterialInactive
	repo.materials[material.ID] = material
	handler := newHandler(repo)
	updated := performRequest(handler, http.MethodPut, "/api/v1/admin/procurement/materials/"+material.ID.String(), `{"code":"SESAME-2","name":"Sesame updated","base_unit":"KG","description":"new"}`, "staff", "SUPPLY_MANAGER")
	if updated.Code != http.StatusOK {
		t.Fatalf("update material status=%d body=%s", updated.Code, updated.Body.String())
	}
	var result domain.RawMaterial
	if err := json.Unmarshal(updated.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Code != "SESAME-2" || result.Status != domain.MaterialInactive {
		t.Fatalf("unexpected updated material: %+v", result)
	}
}
