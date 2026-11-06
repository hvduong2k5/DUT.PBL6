package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/application"
	"github.com/omamx/procurement-service/internal/domain"
)

type fakeRepository struct {
	application.Repository
	suppliers map[uuid.UUID]domain.Supplier
	materials map[uuid.UUID]domain.RawMaterial
	offers    map[uuid.UUID][]domain.SupplierMaterialOffer
	orders    map[uuid.UUID]domain.PurchaseOrder
	receipts  map[uuid.UUID]domain.GoodsReceipt
	contracts map[uuid.UUID]domain.SupplierContract
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		suppliers: make(map[uuid.UUID]domain.Supplier), materials: make(map[uuid.UUID]domain.RawMaterial),
		offers: make(map[uuid.UUID][]domain.SupplierMaterialOffer), orders: make(map[uuid.UUID]domain.PurchaseOrder), receipts: make(map[uuid.UUID]domain.GoodsReceipt), contracts: make(map[uuid.UUID]domain.SupplierContract),
	}
}

func (r *fakeRepository) GetSupplier(_ context.Context, id uuid.UUID) (domain.Supplier, error) {
	item, ok := r.suppliers[id]
	if !ok {
		return domain.Supplier{}, domain.ErrNotFound
	}
	return item, nil
}

func (r *fakeRepository) GetMaterial(_ context.Context, id uuid.UUID) (domain.RawMaterial, error) {
	item, ok := r.materials[id]
	if !ok {
		return domain.RawMaterial{}, domain.ErrNotFound
	}
	return item, nil
}
func (r *fakeRepository) UpdateMaterial(_ context.Context, id uuid.UUID, item domain.RawMaterial) (domain.RawMaterial, error) {
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
func (r *fakeRepository) SetPreferredSupplier(_ context.Context, materialID, supplierID uuid.UUID) (domain.RawMaterial, error) {
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

func (r *fakeRepository) ListSupplierOffers(_ context.Context, supplierID uuid.UUID) ([]domain.SupplierMaterialOffer, error) {
	return append([]domain.SupplierMaterialOffer(nil), r.offers[supplierID]...), nil
}
func (r *fakeRepository) ListMaterialOffers(_ context.Context, materialID uuid.UUID) ([]domain.SupplierMaterialOffer, error) {
	items := []domain.SupplierMaterialOffer{}
	for _, offers := range r.offers {
		for _, offer := range offers {
			if offer.MaterialID == materialID {
				items = append(items, offer)
			}
		}
	}
	return items, nil
}

func (r *fakeRepository) CreateOffer(_ context.Context, offer domain.SupplierMaterialOffer) (domain.SupplierMaterialOffer, error) {
	r.offers[offer.SupplierID] = append(r.offers[offer.SupplierID], offer)
	return offer, nil
}
func (r *fakeRepository) SetOfferActive(_ context.Context, supplierID, offerID uuid.UUID, active bool) (domain.SupplierMaterialOffer, error) {
	items := r.offers[supplierID]
	for i := range items {
		if items[i].ID == offerID {
			items[i].Active = active
			r.offers[supplierID] = items
			return items[i], nil
		}
	}
	return domain.SupplierMaterialOffer{}, domain.ErrNotFound
}
func (r *fakeRepository) CreateSupplierContract(_ context.Context, item domain.SupplierContract) (domain.SupplierContract, error) {
	item.Revision = 1
	for _, existing := range r.contracts {
		if existing.SupplierID == item.SupplierID && existing.ContractNumber == item.ContractNumber && item.Revision <= existing.Revision {
			item.Revision = existing.Revision + 1
		}
	}
	r.contracts[item.ID] = item
	return item, nil
}
func (r *fakeRepository) GetSupplierContract(_ context.Context, id uuid.UUID) (domain.SupplierContract, error) {
	item, ok := r.contracts[id]
	if !ok {
		return domain.SupplierContract{}, domain.ErrNotFound
	}
	return item, nil
}
func (r *fakeRepository) ListSupplierContracts(_ context.Context, id uuid.UUID, _, _ int) ([]domain.SupplierContract, error) {
	items := []domain.SupplierContract{}
	for _, item := range r.contracts {
		if id == uuid.Nil || item.SupplierID == id {
			items = append(items, item)
		}
	}
	return items, nil
}
func (r *fakeRepository) SetSupplierContractStatus(_ context.Context, id uuid.UUID, status domain.SupplierContractStatus, actor string) (domain.SupplierContract, error) {
	item, ok := r.contracts[id]
	if !ok {
		return domain.SupplierContract{}, domain.ErrNotFound
	}
	item.Status = status
	r.contracts[id] = item
	return item, nil
}

func (r *fakeRepository) CreatePurchaseOrder(_ context.Context, order domain.PurchaseOrder) (domain.PurchaseOrder, error) {
	r.orders[order.ID] = order
	return order, nil
}

func (r *fakeRepository) GetPurchaseOrder(_ context.Context, id uuid.UUID) (domain.PurchaseOrder, error) {
	order, ok := r.orders[id]
	if !ok {
		return domain.PurchaseOrder{}, domain.ErrNotFound
	}
	return order, nil
}

func (r *fakeRepository) ListPurchaseOrders(_ context.Context, supplierID uuid.UUID, status domain.PurchaseOrderStatus, limit, offset int) ([]domain.PurchaseOrder, error) {
	items := make([]domain.PurchaseOrder, 0)
	for _, order := range r.orders {
		if supplierID != uuid.Nil && order.SupplierID != supplierID || status != "" && order.Status != status {
			continue
		}
		items = append(items, order)
	}
	return items, nil
}

func (r *fakeRepository) SavePurchaseOrder(_ context.Context, order domain.PurchaseOrder, expectedVersion int64, _ domain.PurchaseOrderStatus, _, _, _ string) (domain.PurchaseOrder, error) {
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
func (r *fakeRepository) RevisePurchaseOrder(_ context.Context, order domain.PurchaseOrder, expectedVersion int64, actor, reason string) (domain.PurchaseOrder, error) {
	current, ok := r.orders[order.ID]
	if !ok {
		return domain.PurchaseOrder{}, domain.ErrNotFound
	}
	if current.Version != expectedVersion {
		return domain.PurchaseOrder{}, domain.ErrConflict
	}
	order.Version = expectedVersion + 1
	r.orders[order.ID] = order
	return order, nil
}
func (r *fakeRepository) ListPurchaseOrderRevisions(_ context.Context, id uuid.UUID, _, _ int) ([]domain.PurchaseOrderRevision, error) {
	return nil, nil
}

func (r *fakeRepository) CreateGoodsReceipt(_ context.Context, receipt domain.GoodsReceipt) (domain.GoodsReceipt, error) {
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
func (r *fakeRepository) GetGoodsReceiptByIdempotency(_ context.Context, actor, key string) (domain.GoodsReceipt, error) {
	for _, receipt := range r.receipts {
		if receipt.ReceivedBy == actor && receipt.IdempotencyKey == key {
			return receipt, nil
		}
	}
	return domain.GoodsReceipt{}, domain.ErrNotFound
}
func (r *fakeRepository) GetGoodsReceipt(_ context.Context, id uuid.UUID) (domain.GoodsReceipt, error) {
	receipt, ok := r.receipts[id]
	if !ok {
		return domain.GoodsReceipt{}, domain.ErrNotFound
	}
	return receipt, nil
}
func (r *fakeRepository) HasUnresolvedGoodsReceiptVariances(_ context.Context, purchaseOrderID uuid.UUID) (bool, error) {
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
func (r *fakeRepository) ResolveGoodsReceiptVariance(_ context.Context, receiptID, varianceID uuid.UUID, decision, actor, note string) (domain.GoodsReceipt, error) {
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
func (r *fakeRepository) InspectGoodsReceipt(_ context.Context, receipt domain.GoodsReceipt, expected int64, _ string) (domain.GoodsReceipt, error) {
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

func makeApplicationFixture(t *testing.T) (*application.Service, *fakeRepository, uuid.UUID, uuid.UUID) {
	t.Helper()
	repo := newFakeRepository()
	supplier, err := domain.NewSupplier("SUP-ONE", "Supplier One", "COOPERATIVE", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	material, err := domain.NewRawMaterial("SESAME", "Sesame", "KG", "")
	if err != nil {
		t.Fatal(err)
	}
	offer, err := domain.NewSupplierMaterialOffer(supplier.ID, material.ID, "", "KG", 10001, "VND", "0.5", 4)
	if err != nil {
		t.Fatal(err)
	}
	repo.suppliers[supplier.ID] = supplier
	repo.materials[material.ID] = material
	repo.offers[supplier.ID] = []domain.SupplierMaterialOffer{offer}
	return application.NewService(repo), repo, supplier.ID, material.ID
}

func TestCreatePurchaseOrderUsesActiveSupplierOfferSnapshot(t *testing.T) {
	service, _, supplierID, materialID := makeApplicationFixture(t)
	delivery := time.Now().UTC().Add(24 * time.Hour)
	order, err := service.CreatePurchaseOrder(context.Background(), application.PurchaseOrderRequest{
		SupplierID: supplierID, ExpectedDeliveryDate: &delivery,
		Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "1.5"}},
	}, "staff-1")
	if err != nil {
		t.Fatalf("CreatePurchaseOrder() error = %v", err)
	}
	if order.Status != domain.PurchaseOrderDraft || order.Currency != "VND" || order.TotalAmountMinor != 15002 {
		t.Fatalf("unexpected draft snapshot: %+v", order)
	}
	if order.Lines[0].UnitPriceMinor != 10001 || order.Lines[0].Quantity != "1.5" {
		t.Fatalf("line did not snapshot supplier offer: %+v", order.Lines[0])
	}
}

func TestPurchaseOrderUsesOnlyEffectivePriceVersion(t *testing.T) {
	service, repo, supplierID, materialID := makeApplicationFixture(t)
	now := time.Now().UTC()
	old := repo.offers[supplierID][0]
	past := now.Add(-time.Hour)
	old.EffectiveTo = &past
	current, err := domain.NewSupplierMaterialOfferVersion(supplierID, materialID, "", "KG", 22000, "VND", "0.5", 3, now.Add(-time.Minute), nil)
	if err != nil {
		t.Fatal(err)
	}
	repo.offers[supplierID] = []domain.SupplierMaterialOffer{old, current}
	order, err := service.CreatePurchaseOrder(context.Background(), application.PurchaseOrderRequest{SupplierID: supplierID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "1"}}}, "buyer")
	if err != nil {
		t.Fatal(err)
	}
	if order.Lines[0].UnitPriceMinor != 22000 || order.TotalAmountMinor != 22000 {
		t.Fatalf("PO used ineffective price version: %+v", order.Lines[0])
	}
}

func TestCreatePurchaseOrderRejectsInactiveSupplierMissingOfferAndBelowMinimum(t *testing.T) {
	t.Run("inactive supplier", func(t *testing.T) {
		service, repo, supplierID, materialID := makeApplicationFixture(t)
		supplier := repo.suppliers[supplierID]
		supplier.Status = domain.SupplierSuspended
		repo.suppliers[supplierID] = supplier
		_, err := service.CreatePurchaseOrder(context.Background(), application.PurchaseOrderRequest{
			SupplierID: supplierID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "1"}},
		}, "staff-1")
		if !errors.Is(err, domain.ErrInvalidState) {
			t.Fatalf("error = %v, want ErrInvalidState", err)
		}
	})
	t.Run("missing offer", func(t *testing.T) {
		service, repo, supplierID, materialID := makeApplicationFixture(t)
		repo.offers[supplierID] = nil
		_, err := service.CreatePurchaseOrder(context.Background(), application.PurchaseOrderRequest{
			SupplierID: supplierID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "1"}},
		}, "staff-1")
		if !errors.Is(err, domain.ErrInvalidState) {
			t.Fatalf("error = %v, want ErrInvalidState", err)
		}
	})
	t.Run("below minimum", func(t *testing.T) {
		service, _, supplierID, materialID := makeApplicationFixture(t)
		_, err := service.CreatePurchaseOrder(context.Background(), application.PurchaseOrderRequest{
			SupplierID: supplierID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "0.25"}},
		}, "staff-1")
		if !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatalf("error = %v, want ErrInvalidArgument", err)
		}
	})
}

func TestCreateOfferRequiresActiveSupplierAndMatchingMaterialUnit(t *testing.T) {
	t.Run("inactive supplier", func(t *testing.T) {
		service, repo, supplierID, materialID := makeApplicationFixture(t)
		supplier := repo.suppliers[supplierID]
		supplier.Status = domain.SupplierSuspended
		repo.suppliers[supplierID] = supplier
		offer, _ := domain.NewSupplierMaterialOffer(supplierID, materialID, "", "KG", 100, "VND", "1", 0)
		if _, err := service.CreateOffer(context.Background(), offer); !errors.Is(err, domain.ErrInvalidState) {
			t.Fatalf("error = %v, want ErrInvalidState", err)
		}
	})
	t.Run("unit differs from material base unit", func(t *testing.T) {
		service, _, supplierID, materialID := makeApplicationFixture(t)
		offer, _ := domain.NewSupplierMaterialOffer(supplierID, materialID, "", "G", 100, "VND", "1", 0)
		if _, err := service.CreateOffer(context.Background(), offer); !errors.Is(err, domain.ErrInvalidState) {
			t.Fatalf("error = %v, want ErrInvalidState", err)
		}
	})
}

func TestPurchaseOrderTransitionsAndConcurrentVersionConflict(t *testing.T) {
	service, repo, supplierID, materialID := makeApplicationFixture(t)
	order, err := service.CreatePurchaseOrder(context.Background(), application.PurchaseOrderRequest{
		SupplierID: supplierID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "1"}},
	}, "creator")
	if err != nil {
		t.Fatal(err)
	}
	order, err = service.SubmitPurchaseOrder(context.Background(), order.ID, "creator")
	if err != nil || order.Status != domain.PurchaseOrderPendingApproval {
		t.Fatalf("SubmitPurchaseOrder() = %s, %v", order.Status, err)
	}
	if _, err := service.ApprovePurchaseOrder(context.Background(), order.ID, "creator"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("self approval error = %v", err)
	}
	order, err = service.ApprovePurchaseOrder(context.Background(), order.ID, "approver")
	if err != nil || order.Status != domain.PurchaseOrderApproved {
		t.Fatalf("ApprovePurchaseOrder() = %s, %v", order.Status, err)
	}
	order, err = service.MarkPurchaseOrderSent(context.Background(), order.ID, "sender")
	if err != nil || order.Status != domain.PurchaseOrderSent {
		t.Fatalf("MarkPurchaseOrderSent() = %s, %v", order.Status, err)
	}
	stale := repo.orders[order.ID]
	stale.Version--
	if _, err := repo.SavePurchaseOrder(context.Background(), stale, stale.Version-1, stale.Status, "STALE", "staff", ""); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale write error = %v, want ErrConflict", err)
	}
}

func TestPurchaseOrderSupplierResponseRevisionAndRepeatedSend(t *testing.T) {
	service, _, supplierID, materialID := makeApplicationFixture(t)
	order, err := service.CreatePurchaseOrder(context.Background(), application.PurchaseOrderRequest{SupplierID: supplierID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "1"}}}, "buyer")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.SubmitPurchaseOrder(context.Background(), order.ID, "buyer"); err != nil {
		t.Fatal(err)
	}
	if _, err = service.ApprovePurchaseOrder(context.Background(), order.ID, "approver"); err != nil {
		t.Fatal(err)
	}
	sent, err := service.MarkPurchaseOrderSent(context.Background(), order.ID, "sender")
	if err != nil {
		t.Fatal(err)
	}
	retried, err := service.MarkPurchaseOrderSent(context.Background(), order.ID, "sender")
	if err != nil {
		t.Fatal(err)
	}
	if retried.Version != sent.Version {
		t.Fatal("repeated send created another PO transition")
	}
	changed, err := service.RecordSupplierResponse(context.Background(), order.ID, "procurement", "CHANGE_REQUESTED", "Please change the quantity")
	if err != nil || changed.Status != domain.PurchaseOrderChangeRequested {
		t.Fatalf("supplier response = %+v, %v", changed, err)
	}
	revised, err := service.RevisePurchaseOrder(context.Background(), order.ID, application.PurchaseOrderRequest{SupplierID: supplierID, Notes: "revised", Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "2"}}}, "buyer-2", "Supplier requested a quantity update")
	if err != nil {
		t.Fatal(err)
	}
	if revised.Status != domain.PurchaseOrderDraft || revised.Revision != 2 || revised.Lines[0].Quantity != "2" {
		t.Fatalf("unexpected new PO revision: %+v", revised)
	}
}

func TestUpdatePurchaseOrderDraftOnly(t *testing.T) {
	service, _, supplierID, materialID := makeApplicationFixture(t)
	order, err := service.CreatePurchaseOrder(context.Background(), application.PurchaseOrderRequest{
		SupplierID: supplierID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "1"}},
	}, "creator")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdatePurchaseOrderDraft(context.Background(), order.ID, application.PurchaseOrderRequest{
		SupplierID: supplierID, Notes: "updated", Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "2"}},
	}, "staff-2")
	if err != nil || updated.Notes != "updated" || updated.Version != order.Version+1 {
		t.Fatalf("UpdatePurchaseOrderDraft() = %+v, %v", updated, err)
	}
	if _, err := service.SubmitPurchaseOrder(context.Background(), order.ID, "creator"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdatePurchaseOrderDraft(context.Background(), order.ID, application.PurchaseOrderRequest{
		SupplierID: supplierID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "2"}},
	}, "staff-2"); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("edit after submit error = %v", err)
	}
}

func TestGoodsReceiptApplicationFlow(t *testing.T) {
	service, repo, supplierID, materialID := makeApplicationFixture(t)
	order, err := service.CreatePurchaseOrder(context.Background(), application.PurchaseOrderRequest{SupplierID: supplierID, Lines: []application.PurchaseOrderLineRequest{{MaterialID: materialID, Quantity: "5"}}}, "buyer")
	if err != nil {
		t.Fatal(err)
	}
	order.Status = domain.PurchaseOrderSent
	repo.orders[order.ID] = order
	request := application.GoodsReceiptRequest{PurchaseOrderID: order.ID, WarehouseCode: "wh-1", Lines: []application.GoodsReceiptLineRequest{{PurchaseOrderLineID: order.Lines[0].ID, QuantityReceived: "2"}}}
	receipt, err := service.CreateGoodsReceipt(context.Background(), request, "receiver", "warehouse-receipt-0001")
	if err != nil {
		t.Fatal(err)
	}
	order.Status = domain.PurchaseOrderReceived
	repo.orders[order.ID] = order
	retried, err := service.CreateGoodsReceipt(context.Background(), request, "receiver", "warehouse-receipt-0001")
	if err != nil || retried.ID != receipt.ID {
		t.Fatalf("same-key receipt retry = (%s, %v), want original receipt %s", retried.ID, err, receipt.ID)
	}
	request.Lines[0].QuantityReceived = "3"
	if _, err := service.CreateGoodsReceipt(context.Background(), request, "receiver", "warehouse-receipt-0001"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("same key with changed payload error = %v, want conflict", err)
	}
	inspected, err := service.InspectGoodsReceipt(context.Background(), receipt.ID, "qc", []domain.GoodsReceiptInspectionLine{{GoodsReceiptLineID: receipt.Lines[0].ID, QuantityAccepted: "1.5", QuantityRejected: "0.5", InspectionNotes: "damaged portion"}})
	if err != nil {
		t.Fatal(err)
	}
	if inspected.Status != domain.GoodsReceiptInspected || inspected.Lines[0].QuantityAccepted != "1.5" {
		t.Fatalf("unexpected inspected receipt: %+v", inspected)
	}
}

func TestPurchaseOrderCannotCloseWhileReceiptVarianceIsOpen(t *testing.T) {
	repo := newFakeRepository()
	supplier, _ := domain.NewSupplier("SUP-VAR", "Supplier", "COMPANY", "", "", "", "")
	material, _ := domain.NewRawMaterial("RICE-VAR", "Rice", "KG", "")
	order, _ := domain.NewPurchaseOrder("PO-VAR", supplier.ID, supplier.Name, "VND", "", "buyer", nil, []domain.PurchaseOrderLineInput{{MaterialID: material.ID, MaterialCode: material.Code, MaterialName: material.Name, Quantity: "5", Unit: "KG", UnitPriceMinor: 100}})
	order.Status = domain.PurchaseOrderPartiallyReceived
	repo.orders[order.ID] = order
	repo.receipts[uuid.New()] = domain.GoodsReceipt{PurchaseOrderID: order.ID, Variances: []domain.GoodsReceiptVariance{{Status: "OPEN"}}}
	_, err := application.NewService(repo).CloseRemainingPurchaseOrder(context.Background(), order.ID, "manager", "close remaining")
	if !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("close PO with open variance error = %v, want invalid state", err)
	}
}

func TestResolvedReceiptVarianceAllowsClosingRemainingPurchaseOrder(t *testing.T) {
	repo := newFakeRepository()
	supplier, _ := domain.NewSupplier("SUP-VAR-RES", "Supplier", "COMPANY", "", "", "", "")
	material, _ := domain.NewRawMaterial("RICE-VAR-RES", "Rice", "KG", "")
	order, _ := domain.NewPurchaseOrder("PO-VAR-RES", supplier.ID, supplier.Name, "VND", "", "buyer", nil, []domain.PurchaseOrderLineInput{{MaterialID: material.ID, MaterialCode: material.Code, MaterialName: material.Name, Quantity: "5", Unit: "KG", UnitPriceMinor: 100}})
	order.Status = domain.PurchaseOrderPartiallyReceived
	repo.orders[order.ID] = order
	receiptID, varianceID := uuid.New(), uuid.New()
	repo.receipts[receiptID] = domain.GoodsReceipt{ID: receiptID, PurchaseOrderID: order.ID, Status: domain.GoodsReceiptInspected, Variances: []domain.GoodsReceiptVariance{{ID: varianceID, Status: "OPEN"}}}
	service := application.NewService(repo)
	if _, err := service.ResolveGoodsReceiptVariance(context.Background(), receiptID, varianceID, "ACCEPT_VARIANCE", "manager", "approved exception"); err != nil {
		t.Fatal(err)
	}
	closed, err := service.CloseRemainingPurchaseOrder(context.Background(), order.ID, "manager", "supplier confirmed no further delivery")
	if err != nil || closed.Status != domain.PurchaseOrderClosed {
		t.Fatalf("close after resolving variance = (%s, %v), want CLOSED", closed.Status, err)
	}
}

func TestSupplierContractCreationRequiresActiveSupplier(t *testing.T) {
	service, repo, supplierID, _ := makeApplicationFixture(t)
	from := time.Now().UTC().Truncate(24 * time.Hour)
	contract, err := domain.NewSupplierContract(supplierID, "AGR-1", "RAW_MATERIALS", "NET 30", "doc-1", from, nil, "manager")
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateSupplierContract(context.Background(), contract)
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != domain.SupplierContractDraft || created.Revision != 1 {
		t.Fatalf("unexpected contract: %+v", created)
	}
	supplier := repo.suppliers[supplierID]
	supplier.Status = domain.SupplierSuspended
	repo.suppliers[supplierID] = supplier
	if _, err := service.CreateSupplierContract(context.Background(), contract); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("inactive supplier contract error=%v", err)
	}
}

func TestUpdateMaterialPreservesStatusAndPreferredSupplier(t *testing.T) {
	service, repo, _, materialID := makeApplicationFixture(t)
	material := repo.materials[materialID]
	preferred := uuid.New()
	material.Status = domain.MaterialInactive
	material.PreferredSupplierID = &preferred
	repo.materials[materialID] = material
	updated, err := service.UpdateMaterial(context.Background(), materialID, domain.RawMaterial{Code: "SESAME-NEW", Name: "New sesame", BaseUnit: "KG", Description: "updated"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Code != "SESAME-NEW" || updated.Status != domain.MaterialInactive || updated.PreferredSupplierID == nil || *updated.PreferredSupplierID != preferred {
		t.Fatalf("update lost existing state: %+v", updated)
	}
}
