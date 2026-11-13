package httpapi_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
	"github.com/omamx/procurement-service/internal/transport/httpapi"
)

var (
	testIdentityOnce sync.Once
	testIdentityKey  *rsa.PrivateKey
	testJWKS         *httptest.Server
)

type httpRepository struct {
	application.Repository
	suppliers     map[uuid.UUID]domain.Supplier
	materials     map[uuid.UUID]domain.RawMaterial
	offers        map[uuid.UUID][]domain.SupplierMaterialOffer
	orders        map[uuid.UUID]domain.PurchaseOrder
	receipts      map[uuid.UUID]domain.GoodsReceipt
	contracts     map[uuid.UUID]domain.SupplierContract
	revisions     map[uuid.UUID][]domain.PurchaseOrderRevision
	plans         map[uuid.UUID]domain.SupplyPlan
	planRevisions map[uuid.UUID][]domain.SupplyPlanRevision
	planHistory   map[uuid.UUID][]domain.SupplyPlanStatusEvent
}

func newHTTPRepository() *httpRepository {
	return &httpRepository{
		suppliers: make(map[uuid.UUID]domain.Supplier), materials: make(map[uuid.UUID]domain.RawMaterial),
		offers: make(map[uuid.UUID][]domain.SupplierMaterialOffer), orders: make(map[uuid.UUID]domain.PurchaseOrder), receipts: make(map[uuid.UUID]domain.GoodsReceipt), contracts: make(map[uuid.UUID]domain.SupplierContract), revisions: make(map[uuid.UUID][]domain.PurchaseOrderRevision), plans: make(map[uuid.UUID]domain.SupplyPlan), planRevisions: make(map[uuid.UUID][]domain.SupplyPlanRevision), planHistory: make(map[uuid.UUID][]domain.SupplyPlanStatusEvent),
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
func (r *httpRepository) CreateSupplyPlan(_ context.Context, plan domain.SupplyPlan) (domain.SupplyPlan, error) {
	r.plans[plan.ID] = plan
	r.planRevisions[plan.ID] = []domain.SupplyPlanRevision{{SupplyPlanID: plan.ID, Revision: plan.Revision, Snapshot: plan.Snapshot, CreatedBy: plan.CreatedBy, CreatedAt: plan.CreatedAt}}
	r.planHistory[plan.ID] = []domain.SupplyPlanStatusEvent{{SupplyPlanID: plan.ID, FromStatus: plan.Status, ToStatus: plan.Status, Action: "CREATED", Actor: plan.CreatedBy, Version: plan.Version, CreatedAt: plan.CreatedAt}}
	return plan, nil
}
func (r *httpRepository) GetSupplyPlan(_ context.Context, id uuid.UUID) (domain.SupplyPlan, error) {
	plan, ok := r.plans[id]
	if !ok {
		return domain.SupplyPlan{}, domain.ErrNotFound
	}
	return plan, nil
}
func (r *httpRepository) ListSupplyPlans(_ context.Context, limit, offset int) ([]domain.SupplyPlan, error) {
	items := make([]domain.SupplyPlan, 0, len(r.plans))
	for _, plan := range r.plans {
		items = append(items, plan)
	}
	start := offset
	if start > len(items) {
		start = len(items)
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], nil
}
func (r *httpRepository) ReviseSupplyPlan(_ context.Context, plan domain.SupplyPlan, revision domain.SupplyPlanRevision, expectedVersion int64) (domain.SupplyPlan, error) {
	current, ok := r.plans[plan.ID]
	if !ok {
		return domain.SupplyPlan{}, domain.ErrNotFound
	}
	if current.Version != expectedVersion {
		return domain.SupplyPlan{}, domain.ErrConflict
	}
	r.planRevisions[plan.ID] = append(r.planRevisions[plan.ID], revision)
	r.plans[plan.ID] = plan
	return plan, nil
}
func (r *httpRepository) ListSupplyPlanRevisions(_ context.Context, id uuid.UUID) ([]domain.SupplyPlanRevision, error) {
	if _, ok := r.plans[id]; !ok {
		return nil, domain.ErrNotFound
	}
	return append([]domain.SupplyPlanRevision(nil), r.planRevisions[id]...), nil
}
func (r *httpRepository) TransitionSupplyPlan(_ context.Context, plan domain.SupplyPlan, expectedVersion int64, previousStatus domain.SupplyPlanStatus, actor, action, reason string) (domain.SupplyPlan, error) {
	current, ok := r.plans[plan.ID]
	if !ok {
		return domain.SupplyPlan{}, domain.ErrNotFound
	}
	if current.Version != expectedVersion || current.Status != previousStatus {
		return domain.SupplyPlan{}, domain.ErrConflict
	}
	r.plans[plan.ID] = plan
	r.planHistory[plan.ID] = append(r.planHistory[plan.ID], domain.SupplyPlanStatusEvent{SupplyPlanID: plan.ID, FromStatus: previousStatus,
		ToStatus: plan.Status, Action: action, Actor: actor, Reason: reason, Version: plan.Version, CreatedAt: plan.UpdatedAt})
	return plan, nil
}
func (r *httpRepository) ListSupplyPlanStatusHistory(_ context.Context, id uuid.UUID) ([]domain.SupplyPlanStatusEvent, error) {
	if _, ok := r.plans[id]; !ok {
		return nil, domain.ErrNotFound
	}
	return append([]domain.SupplyPlanStatusEvent(nil), r.planHistory[id]...), nil
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
func (r *httpRepository) GetPurchaseOrderByIdempotency(_ context.Context, actor, key string) (domain.PurchaseOrder, error) {
	for _, order := range r.orders {
		if order.CreatedBy == actor && order.IdempotencyKey == key {
			return order, nil
		}
	}
	return domain.PurchaseOrder{}, domain.ErrNotFound
}
func (r *httpRepository) CreatePurchaseOrderFromSupplyPlan(_ context.Context, plan domain.SupplyPlan, order domain.PurchaseOrder, expectedVersion int64, actor string) (domain.PurchaseOrder, error) {
	current, ok := r.plans[plan.ID]
	if !ok {
		return domain.PurchaseOrder{}, domain.ErrNotFound
	}
	if current.Status == domain.SupplyPlanConverted && current.ConvertedPurchaseOrderID != nil {
		existing := r.orders[*current.ConvertedPurchaseOrderID]
		if existing.SupplierID != order.SupplierID {
			return domain.PurchaseOrder{}, domain.ErrConflict
		}
		return existing, nil
	}
	if current.Status != domain.SupplyPlanApproved || current.Version != expectedVersion {
		return domain.PurchaseOrder{}, domain.ErrConflict
	}
	r.orders[order.ID] = order
	r.plans[plan.ID] = plan
	r.planHistory[plan.ID] = append(r.planHistory[plan.ID], domain.SupplyPlanStatusEvent{SupplyPlanID: plan.ID, FromStatus: current.Status,
		ToStatus: plan.Status, Action: "PO_CREATED", Actor: actor, Version: plan.Version, CreatedAt: plan.UpdatedAt})
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
	for _, existing := range r.receipts {
		if existing.ReceivedBy == receipt.ReceivedBy && existing.IdempotencyKey == receipt.IdempotencyKey {
			if existing.RequestHash != receipt.RequestHash {
				return domain.GoodsReceipt{}, domain.ErrConflict
			}
			return existing, nil
		}
	}
	r.receipts[receipt.ID] = receipt
	return receipt, nil
}
func (r *httpRepository) GetGoodsReceiptByIdempotency(_ context.Context, actor, key string) (domain.GoodsReceipt, error) {
	for _, receipt := range r.receipts {
		if receipt.ReceivedBy == actor && receipt.IdempotencyKey == key {
			return receipt, nil
		}
	}
	return domain.GoodsReceipt{}, domain.ErrNotFound
}
func (r *httpRepository) GetGoodsReceipt(_ context.Context, id uuid.UUID) (domain.GoodsReceipt, error) {
	receipt, ok := r.receipts[id]
	if !ok {
		return domain.GoodsReceipt{}, domain.ErrNotFound
	}
	return receipt, nil
}
func (r *httpRepository) HasUnresolvedGoodsReceiptVariances(_ context.Context, purchaseOrderID uuid.UUID) (bool, error) {
	for _, receipt := range r.receipts {
		if receipt.PurchaseOrderID != purchaseOrderID {
			continue
		}
		for _, variance := range receipt.Variances {
			if variance.Status == "OPEN" {
				return true, nil
			}
		}
	}
	return false, nil
}
func (r *httpRepository) ResolveGoodsReceiptVariance(_ context.Context, receiptID, varianceID uuid.UUID, decision, actor, note string) (domain.GoodsReceipt, error) {
	receipt, ok := r.receipts[receiptID]
	if !ok {
		return domain.GoodsReceipt{}, domain.ErrNotFound
	}
	if err := receipt.ResolveVariance(varianceID, decision, actor, note); err != nil {
		return domain.GoodsReceipt{}, err
	}
	r.receipts[receiptID] = receipt
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
	config := testIdentityConfig()
	handler, err := httpapi.NewRouter(application.NewService(repo), config, repo.Ping)
	if err != nil {
		panic(err)
	}
	return handler
}

func testIdentityConfig() httpapi.IdentityConfig {
	testIdentityOnce.Do(func() {
		var err error
		testIdentityKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
		exponent := big.NewInt(int64(testIdentityKey.PublicKey.E)).Bytes()
		jwks := map[string]any{"keys": []any{map[string]string{
			"kid": "procurement-test-key", "kty": "RSA", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(testIdentityKey.PublicKey.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(exponent),
		}}}
		testJWKS = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(jwks)
		}))
	})
	return httpapi.IdentityConfig{Issuer: "https://identity.test/realms/omama", Audience: "procurement-api", JWKSURL: testJWKS.URL, AllowLocalHTTP: true}
}

func employeeToken(user, role string) string {
	grants := []string{}
	switch role {
	case "ADMIN", "PROCUREMENT_MANAGER":
		grants = []string{"PROCUREMENT_VIEW#GLOBAL", "PROCUREMENT_MANAGE#GLOBAL", "PROCUREMENT_APPROVE#GLOBAL", "PROCUREMENT_RECEIVE#GLOBAL", "PROCUREMENT_VARIANCE_RESOLVE#GLOBAL"}
	case "SUPPLY_MANAGER":
		grants = []string{"PROCUREMENT_VIEW#GLOBAL", "PROCUREMENT_MANAGE#GLOBAL", "PROCUREMENT_RECEIVE#GLOBAL"}
	case "WAREHOUSE_STAFF":
		grants = []string{"PROCUREMENT_RECEIVE#WAREHOUSE#WH-01"}
	}
	principalID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(user)).String()
	claims := map[string]any{
		"iss": "https://identity.test/realms/omama", "sub": "subject-" + user, "aud": []string{"omama-api", "procurement-api"},
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Add(-time.Minute).Unix(), "typ": "Bearer",
		"user_id": principalID, "principal_type": "EMPLOYEE", "permission_grants": grants,
	}
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "procurement-test-key", "typ": "JWT"})
	body, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(body)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, testIdentityKey, crypto.SHA256, digest[:])
	if err != nil {
		panic(err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func performRequest(handler http.Handler, method, path, body, user, role string) *httptest.ResponseRecorder {
	return performRequestWithIdempotency(handler, method, path, body, user, role, "")
}

func performRequestWithIdempotency(handler http.Handler, method, path, body, user, role, key string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+employeeToken(user, role))
	if key != "" {
		request.Header.Set("Idempotency-Key", key)
	}
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestAdminSupplierAPIRequiresEmployeeJWTAndPermission(t *testing.T) {
	repo := newHTTPRepository()
	handler := newHandler(repo)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/procurement/suppliers", strings.NewReader(`{"code":"SUP-1","name":"Supplier","supplier_type":"COMPANY"}`))
	request.Header.Set("X-Internal-Token", "test-token")
	request.Header.Set("X-User-ID", uuid.NewString())
	request.Header.Set("X-User-Role", "ADMIN")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	noPermission := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/suppliers", `{"code":"SUP-0","name":"Supplier","supplier_type":"COMPANY"}`, "unassigned", "UNASSIGNED")
	if noPermission.Code != http.StatusForbidden {
		t.Fatalf("valid employee without PROCUREMENT_MANAGE status=%d body=%s, want 403", noPermission.Code, noPermission.Body.String())
	}

	denied := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/suppliers", `{"code":"SUP-1","name":"Supplier","supplier_type":"COMPANY"}`, "staff", "WAREHOUSE_STAFF")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("warehouse staff status = %d, want 403", denied.Code)
	}
}

func TestWarehouseReceiptGrantCannotCrossWarehouseScope(t *testing.T) {
	handler := newHandler(newHTTPRepository())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/procurement/goods-receipts", strings.NewReader(`{"purchase_order_id":"`+uuid.NewString()+`","warehouse_code":"WH-02","lines":[]}`))
	request.Header.Set("Authorization", "Bearer "+employeeToken("warehouse-1", "WAREHOUSE_STAFF"))
	request.Header.Set("Idempotency-Key", "warehouse-scope-check-001")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("out-of-scope goods receipt status = %d body=%s, want 403", recorder.Code, recorder.Body.String())
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

func TestCreatePurchaseOrderHTTPRequiresAndReplaysIdempotencyKey(t *testing.T) {
	repo := newHTTPRepository()
	supplier, err := domain.NewSupplier("SUP-PO-IDEMP", "Idempotent supplier", "COMPANY", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	material, err := domain.NewRawMaterial("RICE-PO-IDEMP", "Rice", "KG", "")
	if err != nil {
		t.Fatal(err)
	}
	offer, err := domain.NewSupplierMaterialOfferVersion(supplier.ID, material.ID, "", "KG", 12500, "VND", "0.5", 2, time.Now().UTC().Add(-time.Hour), nil)
	if err != nil {
		t.Fatal(err)
	}
	repo.suppliers[supplier.ID], repo.materials[material.ID] = supplier, material
	repo.offers[supplier.ID] = []domain.SupplierMaterialOffer{offer}
	handler := newHandler(repo)
	path := "/api/v1/admin/procurement/purchase-orders"
	body := `{"supplier_id":"` + supplier.ID.String() + `","notes":"weekly restock","lines":[{"material_id":"` + material.ID.String() + `","quantity":"1.5"}]}`
	missingKey := performRequest(handler, http.MethodPost, path, body, "buyer", "SUPPLY_MANAGER")
	if missingKey.Code != http.StatusBadRequest {
		t.Fatalf("missing idempotency key status=%d, want 400", missingKey.Code)
	}
	key := "purchase-order-http-0001"
	first := performRequestWithIdempotency(handler, http.MethodPost, path, body, "buyer", "SUPPLY_MANAGER", key)
	if first.Code != http.StatusCreated {
		t.Fatalf("create PO status=%d body=%s", first.Code, first.Body.String())
	}
	var created domain.PurchaseOrder
	if err := json.Unmarshal(first.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	retry := performRequestWithIdempotency(handler, http.MethodPost, path, body, "buyer", "SUPPLY_MANAGER", key)
	if retry.Code != http.StatusCreated || !strings.Contains(retry.Body.String(), created.ID.String()) || len(repo.orders) != 1 {
		t.Fatalf("retry status=%d body=%s orders=%d", retry.Code, retry.Body.String(), len(repo.orders))
	}
	changedBody := strings.Replace(body, "weekly restock", "changed request", 1)
	conflict := performRequestWithIdempotency(handler, http.MethodPost, path, changedBody, "buyer", "SUPPLY_MANAGER", key)
	if conflict.Code != http.StatusConflict || len(repo.orders) != 1 {
		t.Fatalf("changed payload status=%d body=%s orders=%d", conflict.Code, conflict.Body.String(), len(repo.orders))
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
	created := performRequestWithIdempotency(handler, http.MethodPost, "/api/v1/admin/procurement/purchase-orders", body, "creator", "PROCUREMENT_MANAGER", "purchase-order-create-flow")
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

	submitted := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/purchase-orders/"+order.ID.String()+"/submit", "", "creator", "PROCUREMENT_MANAGER")
	if submitted.Code != http.StatusOK {
		t.Fatalf("submit status = %d, body=%s", submitted.Code, submitted.Body.String())
	}
	selfApproval := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/purchase-orders/"+order.ID.String()+"/approve", "", "creator", "PROCUREMENT_MANAGER")
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
	created := performRequestWithIdempotency(handler, http.MethodPost, "/api/v1/admin/procurement/purchase-orders", body, "buyer", "SUPPLY_MANAGER", "purchase-order-revision-flow")
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
	created := performRequestWithIdempotency(handler, http.MethodPost, "/api/v1/admin/procurement/goods-receipts", body, "warehouse-1", "WAREHOUSE_STAFF", "warehouse-receipt-0001")
	if created.Code != http.StatusCreated {
		t.Fatalf("create goods receipt status=%d body=%s", created.Code, created.Body.String())
	}
	denied := performRequest(handler, http.MethodGet, "/api/v1/admin/procurement/suppliers", "", "warehouse-1", "WAREHOUSE_STAFF")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("warehouse supplier access status=%d, want 403", denied.Code)
	}
}

func TestGoodsReceiptIdempotencyReplaysAndRejectsChangedPayload(t *testing.T) {
	repo := newHTTPRepository()
	supplier, _ := domain.NewSupplier("SUP-GR-RETRY", "Supplier", "COMPANY", "", "", "", "")
	material, _ := domain.NewRawMaterial("RICE-RETRY", "Rice", "KG", "")
	po, _ := domain.NewPurchaseOrder("PO-GR-RETRY", supplier.ID, supplier.Name, "VND", "", "buyer", nil, []domain.PurchaseOrderLineInput{{MaterialID: material.ID, MaterialCode: material.Code, MaterialName: material.Name, Quantity: "5", Unit: "KG", UnitPriceMinor: 100}})
	po.Status = domain.PurchaseOrderSent
	repo.orders[po.ID] = po
	handler := newHandler(repo)
	path := "/api/v1/admin/procurement/goods-receipts"
	body := `{"purchase_order_id":"` + po.ID.String() + `","warehouse_code":"WH-01","lines":[{"purchase_order_line_id":"` + po.Lines[0].ID.String() + `","quantity_received":"2"}]}`
	key := "warehouse-receipt-retry-0001"
	first := performRequestWithIdempotency(handler, http.MethodPost, path, body, "warehouse-1", "WAREHOUSE_STAFF", key)
	retry := performRequestWithIdempotency(handler, http.MethodPost, path, body, "warehouse-1", "WAREHOUSE_STAFF", key)
	if first.Code != http.StatusCreated || retry.Code != http.StatusCreated {
		t.Fatalf("create/retry statuses = %d/%d, want 201/201; bodies=%s / %s", first.Code, retry.Code, first.Body.String(), retry.Body.String())
	}
	var firstReceipt, retriedReceipt domain.GoodsReceipt
	if err := json.Unmarshal(first.Body.Bytes(), &firstReceipt); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(retry.Body.Bytes(), &retriedReceipt); err != nil {
		t.Fatal(err)
	}
	if firstReceipt.ID != retriedReceipt.ID || len(repo.receipts) != 1 {
		t.Fatalf("retry created duplicate receipt: first=%s retry=%s stored=%d", firstReceipt.ID, retriedReceipt.ID, len(repo.receipts))
	}
	changedBody := strings.Replace(body, `"quantity_received":"2"`, `"quantity_received":"3"`, 1)
	conflict := performRequestWithIdempotency(handler, http.MethodPost, path, changedBody, "warehouse-1", "WAREHOUSE_STAFF", key)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("same key with changed payload status = %d body=%s, want 409", conflict.Code, conflict.Body.String())
	}
	missingKey := performRequest(handler, http.MethodPost, path, body, "warehouse-1", "WAREHOUSE_STAFF")
	if missingKey.Code != http.StatusBadRequest {
		t.Fatalf("missing idempotency key status = %d, want 400", missingKey.Code)
	}
}

func TestResolvingGoodsReceiptVarianceRequiresSensitiveRole(t *testing.T) {
	repo := newHTTPRepository()
	receiptID, varianceID := uuid.New(), uuid.New()
	repo.receipts[receiptID] = domain.GoodsReceipt{ID: receiptID, WarehouseCode: "WH-01", Status: domain.GoodsReceiptInspected, Variances: []domain.GoodsReceiptVariance{{ID: varianceID, Type: domain.GoodsReceiptVarianceOverReceived, Status: "OPEN"}}}
	handler := newHandler(repo)
	path := "/api/v1/admin/procurement/goods-receipts/" + receiptID.String() + "/variances/" + varianceID.String() + "/resolve"
	body := `{"decision":"ACCEPT_VARIANCE","note":"approved as documented exception"}`
	denied := performRequest(handler, http.MethodPost, path, body, "supply", "SUPPLY_MANAGER")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("supply manager resolve status = %d, want 403", denied.Code)
	}
	resolved := performRequest(handler, http.MethodPost, path, body, "procurement", "PROCUREMENT_MANAGER")
	if resolved.Code != http.StatusOK || !strings.Contains(resolved.Body.String(), `"status":"RESOLVED"`) {
		t.Fatalf("procurement manager resolve status=%d body=%s", resolved.Code, resolved.Body.String())
	}
}

func TestSupplyPlanPreviewRouteReturnsComputedProposal(t *testing.T) {
	repo := newHTTPRepository()
	material, _ := domain.NewRawMaterial("RICE-PLAN", "Rice", "KG", "")
	repo.materials[material.ID] = material
	supplier, err := domain.NewSupplier("SUP-PLAN-PO", "Plan supplier", "COMPANY", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	repo.suppliers[supplier.ID] = supplier
	offer, err := domain.NewSupplierMaterialOfferVersion(supplier.ID, material.ID, "", "KG", 15000, "VND", "0.5", 3, time.Now().UTC().Add(-time.Hour), nil)
	if err != nil {
		t.Fatal(err)
	}
	repo.offers[supplier.ID] = []domain.SupplierMaterialOffer{offer}
	handler := newHandler(repo)
	now := time.Now().UTC().Truncate(time.Second)
	bodyBytes, err := json.Marshal(map[string]any{
		"horizon_start": now.AddDate(0, 0, 1).Format("2006-01-02"), "horizon_end": now.AddDate(0, 1, 0).Format("2006-01-02"),
		"forecast_source": "forecast-snapshot-1", "forecast_as_of": now.Add(-time.Minute).Format(time.RFC3339),
		"inventory_source": "inventory-snapshot-1", "inventory_as_of": now.Add(-time.Minute).Format(time.RFC3339),
		"stale_after_minutes": 120, "policy_version": "policy-v1",
		"lines": []map[string]any{{"material_id": material.ID.String(), "forecast_quantity": "15", "on_hand_quantity": "2", "open_po_quantity": "3", "safety_stock": "1", "minimum_order_quantity": "4", "pack_size": "1", "lead_time_days": 2, "manual_override_quantity": "8", "manual_override_reason": "Limit purchase to current budget"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	response := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/supply-plans/preview", string(bodyBytes), "supply", "SUPPLY_MANAGER")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"net_requirement":"11.000000"`) || !strings.Contains(response.Body.String(), `"recommended_order_quantity":"11.000000"`) || !strings.Contains(response.Body.String(), `"proposed_order_quantity":"8.000000"`) || !strings.Contains(response.Body.String(), `"manual_override_reason":"Limit purchase to current budget"`) || !strings.Contains(response.Body.String(), `"material_code":"RICE-PLAN"`) {
		t.Fatalf("supply plan preview status=%d body=%s", response.Code, response.Body.String())
	}
	created := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/supply-plans", string(bodyBytes), "supply", "SUPPLY_MANAGER")
	if created.Code != http.StatusCreated {
		t.Fatalf("create supply plan status=%d body=%s", created.Code, created.Body.String())
	}
	var plan domain.SupplyPlan
	if err := json.Unmarshal(created.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Status != domain.SupplyPlanDraft || plan.Revision != 1 {
		t.Fatalf("unexpected created supply plan state: %+v", plan)
	}
	loaded := performRequest(handler, http.MethodGet, "/api/v1/admin/procurement/supply-plans/"+plan.ID.String(), "", "supply", "SUPPLY_MANAGER")
	if loaded.Code != http.StatusOK || !strings.Contains(loaded.Body.String(), `"net_requirement":"11.000000"`) {
		t.Fatalf("get supply plan status=%d body=%s", loaded.Code, loaded.Body.String())
	}
	listed := performRequest(handler, http.MethodGet, "/api/v1/admin/procurement/supply-plans?limit=20&offset=0", "", "supply", "SUPPLY_MANAGER")
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), plan.ID.String()) {
		t.Fatalf("list supply plans status=%d body=%s", listed.Code, listed.Body.String())
	}
	var revisionRequest map[string]any
	if err := json.Unmarshal(bodyBytes, &revisionRequest); err != nil {
		t.Fatal(err)
	}
	revisionRequest["expected_version"] = float64(plan.Version)
	lines := revisionRequest["lines"].([]any)
	lines[0].(map[string]any)["forecast_quantity"] = "20"
	revisionBody, err := json.Marshal(revisionRequest)
	if err != nil {
		t.Fatal(err)
	}
	revised := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/supply-plans/"+plan.ID.String()+"/revisions", string(revisionBody), "supply", "SUPPLY_MANAGER")
	if revised.Code != http.StatusOK || !strings.Contains(revised.Body.String(), `"revision":2`) {
		t.Fatalf("revise supply plan status=%d body=%s", revised.Code, revised.Body.String())
	}
	revisionList := performRequest(handler, http.MethodGet, "/api/v1/admin/procurement/supply-plans/"+plan.ID.String()+"/revisions", "", "supply", "SUPPLY_MANAGER")
	if revisionList.Code != http.StatusOK || !strings.Contains(revisionList.Body.String(), `"revision":1`) || !strings.Contains(revisionList.Body.String(), `"revision":2`) || !strings.Contains(revisionList.Body.String(), `"forecast_quantity":"15.000000"`) {
		t.Fatalf("list supply plan revisions status=%d body=%s", revisionList.Code, revisionList.Body.String())
	}
	submit := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/supply-plans/"+plan.ID.String()+"/submit", "", "supply", "SUPPLY_MANAGER")
	if submit.Code != http.StatusOK || !strings.Contains(submit.Body.String(), `"status":"PENDING_APPROVAL"`) {
		t.Fatalf("submit supply plan status=%d body=%s", submit.Code, submit.Body.String())
	}
	deniedApproval := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/supply-plans/"+plan.ID.String()+"/approve", "", "supply", "SUPPLY_MANAGER")
	if deniedApproval.Code != http.StatusForbidden {
		t.Fatalf("supply manager approval status=%d, want 403", deniedApproval.Code)
	}
	approved := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/supply-plans/"+plan.ID.String()+"/approve", "", "procurement", "PROCUREMENT_MANAGER")
	if approved.Code != http.StatusOK || !strings.Contains(approved.Body.String(), `"status":"APPROVED"`) {
		t.Fatalf("approve supply plan status=%d body=%s", approved.Code, approved.Body.String())
	}
	var approvedPlan domain.SupplyPlan
	if err := json.Unmarshal(approved.Body.Bytes(), &approvedPlan); err != nil {
		t.Fatal(err)
	}
	statusHistory := performRequest(handler, http.MethodGet, "/api/v1/admin/procurement/supply-plans/"+plan.ID.String()+"/status-history", "", "supply", "SUPPLY_MANAGER")
	if statusHistory.Code != http.StatusOK || !strings.Contains(statusHistory.Body.String(), `"to_status":"PENDING_APPROVAL"`) || !strings.Contains(statusHistory.Body.String(), `"to_status":"APPROVED"`) {
		t.Fatalf("supply plan status history status=%d body=%s", statusHistory.Code, statusHistory.Body.String())
	}
	conversionBody, err := json.Marshal(map[string]any{"supplier_id": supplier.ID.String(), "expected_version": approvedPlan.Version})
	if err != nil {
		t.Fatal(err)
	}
	conversionPath := "/api/v1/admin/procurement/supply-plans/" + plan.ID.String() + "/purchase-order"
	converted := performRequest(handler, http.MethodPost, conversionPath, string(conversionBody), "supply", "SUPPLY_MANAGER")
	if converted.Code != http.StatusCreated || !strings.Contains(converted.Body.String(), `"supply_plan_id":"`+plan.ID.String()+`"`) {
		t.Fatalf("convert supply plan status=%d body=%s", converted.Code, converted.Body.String())
	}
	var convertedOrder domain.PurchaseOrder
	if err := json.Unmarshal(converted.Body.Bytes(), &convertedOrder); err != nil {
		t.Fatal(err)
	}
	retriedConversion := performRequest(handler, http.MethodPost, conversionPath, string(conversionBody), "supply", "SUPPLY_MANAGER")
	if retriedConversion.Code != http.StatusCreated || !strings.Contains(retriedConversion.Body.String(), convertedOrder.ID.String()) || len(repo.orders) != 1 {
		t.Fatalf("conversion retry status=%d body=%s orders=%d", retriedConversion.Code, retriedConversion.Body.String(), len(repo.orders))
	}
	rejectedPlanResponse := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/supply-plans", string(bodyBytes), "supply", "SUPPLY_MANAGER")
	if rejectedPlanResponse.Code != http.StatusCreated {
		t.Fatalf("create plan for rejection status=%d body=%s", rejectedPlanResponse.Code, rejectedPlanResponse.Body.String())
	}
	var rejectedPlan domain.SupplyPlan
	if err := json.Unmarshal(rejectedPlanResponse.Body.Bytes(), &rejectedPlan); err != nil {
		t.Fatal(err)
	}
	rejectedSubmit := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/supply-plans/"+rejectedPlan.ID.String()+"/submit", "", "supply", "SUPPLY_MANAGER")
	if rejectedSubmit.Code != http.StatusOK {
		t.Fatalf("submit plan for rejection status=%d body=%s", rejectedSubmit.Code, rejectedSubmit.Body.String())
	}
	rejection := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/supply-plans/"+rejectedPlan.ID.String()+"/reject", `{"reason":"forecast requires correction"}`, "procurement", "PROCUREMENT_MANAGER")
	if rejection.Code != http.StatusOK || !strings.Contains(rejection.Body.String(), `"status":"REJECTED"`) || !strings.Contains(rejection.Body.String(), `"rejection_reason":"forecast requires correction"`) {
		t.Fatalf("reject supply plan status=%d body=%s", rejection.Code, rejection.Body.String())
	}
}

func TestSupplierContractsRequireManagePermissionAndCreateDraftRevision(t *testing.T) {
	repo := newHTTPRepository()
	supplier, _ := domain.NewSupplier("SUP-CON", "Supplier", "COMPANY", "", "", "", "")
	repo.suppliers[supplier.ID] = supplier
	handler := newHandler(repo)
	body := `{"contract_number":"AGR-01","scope":"RAW_MATERIALS","payment_terms":"NET 30","document_file_id":"file-abc","effective_from":"2026-11-01"}`
	denied := performRequest(handler, http.MethodPost, "/api/v1/admin/procurement/suppliers/"+supplier.ID.String()+"/contracts", body, "warehouse", "WAREHOUSE_STAFF")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("warehouse staff contract status=%d, want 403", denied.Code)
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
