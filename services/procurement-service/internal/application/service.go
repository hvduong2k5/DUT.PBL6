package application

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/procurement-service/internal/domain"
)

type Repository interface {
	CreateSupplier(context.Context, domain.Supplier) (domain.Supplier, error)
	UpdateSupplier(context.Context, uuid.UUID, domain.Supplier) (domain.Supplier, error)
	GetSupplier(context.Context, uuid.UUID) (domain.Supplier, error)
	ListSuppliers(context.Context, int, int) ([]domain.Supplier, error)
	SetSupplierStatus(context.Context, uuid.UUID, domain.SupplierStatus) (domain.Supplier, error)
	CreateMaterial(context.Context, domain.RawMaterial) (domain.RawMaterial, error)
	UpdateMaterial(context.Context, uuid.UUID, domain.RawMaterial) (domain.RawMaterial, error)
	SetMaterialStatus(context.Context, uuid.UUID, domain.MaterialStatus) (domain.RawMaterial, error)
	SetPreferredSupplier(context.Context, uuid.UUID, uuid.UUID) (domain.RawMaterial, error)
	GetMaterial(context.Context, uuid.UUID) (domain.RawMaterial, error)
	ListMaterials(context.Context, int, int) ([]domain.RawMaterial, error)
	CreateOffer(context.Context, domain.SupplierMaterialOffer) (domain.SupplierMaterialOffer, error)
	SetOfferActive(context.Context, uuid.UUID, uuid.UUID, bool) (domain.SupplierMaterialOffer, error)
	ListSupplierOffers(context.Context, uuid.UUID) ([]domain.SupplierMaterialOffer, error)
	ListMaterialOffers(context.Context, uuid.UUID) ([]domain.SupplierMaterialOffer, error)
	CreateSupplierContract(context.Context, domain.SupplierContract) (domain.SupplierContract, error)
	GetSupplierContract(context.Context, uuid.UUID) (domain.SupplierContract, error)
	ListSupplierContracts(context.Context, uuid.UUID, int, int) ([]domain.SupplierContract, error)
	SetSupplierContractStatus(context.Context, uuid.UUID, domain.SupplierContractStatus, string) (domain.SupplierContract, error)
	CreatePurchaseOrder(context.Context, domain.PurchaseOrder) (domain.PurchaseOrder, error)
	GetPurchaseOrder(context.Context, uuid.UUID) (domain.PurchaseOrder, error)
	ListPurchaseOrders(context.Context, uuid.UUID, domain.PurchaseOrderStatus, int, int) ([]domain.PurchaseOrder, error)
	SavePurchaseOrder(context.Context, domain.PurchaseOrder, int64, domain.PurchaseOrderStatus, string, string, string) (domain.PurchaseOrder, error)
	RevisePurchaseOrder(context.Context, domain.PurchaseOrder, int64, string, string) (domain.PurchaseOrder, error)
	ListPurchaseOrderRevisions(context.Context, uuid.UUID, int, int) ([]domain.PurchaseOrderRevision, error)
	CreateGoodsReceipt(context.Context, domain.GoodsReceipt) (domain.GoodsReceipt, error)
	GetGoodsReceipt(context.Context, uuid.UUID) (domain.GoodsReceipt, error)
	InspectGoodsReceipt(context.Context, domain.GoodsReceipt, int64, string) (domain.GoodsReceipt, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) CreateSupplier(ctx context.Context, supplier domain.Supplier) (domain.Supplier, error) {
	return s.repo.CreateSupplier(ctx, supplier)
}
func (s *Service) UpdateSupplier(ctx context.Context, id uuid.UUID, supplier domain.Supplier) (domain.Supplier, error) {
	return s.repo.UpdateSupplier(ctx, id, supplier)
}
func (s *Service) GetSupplier(ctx context.Context, id uuid.UUID) (domain.Supplier, error) {
	return s.repo.GetSupplier(ctx, id)
}
func (s *Service) ListSuppliers(ctx context.Context, limit, offset int) ([]domain.Supplier, error) {
	return s.repo.ListSuppliers(ctx, limit, offset)
}
func (s *Service) SetSupplierStatus(ctx context.Context, id uuid.UUID, status domain.SupplierStatus) (domain.Supplier, error) {
	return s.repo.SetSupplierStatus(ctx, id, status)
}
func (s *Service) UpdateMaterial(ctx context.Context, id uuid.UUID, material domain.RawMaterial) (domain.RawMaterial, error) {
	current, err := s.repo.GetMaterial(ctx, id)
	if err != nil {
		return domain.RawMaterial{}, err
	}
	if err := current.Replace(material); err != nil {
		return domain.RawMaterial{}, err
	}
	return s.repo.UpdateMaterial(ctx, id, current)
}
func (s *Service) CreateMaterial(ctx context.Context, material domain.RawMaterial) (domain.RawMaterial, error) {
	return s.repo.CreateMaterial(ctx, material)
}
func (s *Service) SetMaterialStatus(ctx context.Context, id uuid.UUID, status domain.MaterialStatus) (domain.RawMaterial, error) {
	return s.repo.SetMaterialStatus(ctx, id, status)
}
func (s *Service) SetPreferredSupplier(ctx context.Context, materialID, supplierID uuid.UUID) (domain.RawMaterial, error) {
	if _, err := s.repo.GetMaterial(ctx, materialID); err != nil {
		return domain.RawMaterial{}, err
	}
	if supplierID != uuid.Nil {
		if _, err := s.repo.GetSupplier(ctx, supplierID); err != nil {
			return domain.RawMaterial{}, err
		}
	}
	return s.repo.SetPreferredSupplier(ctx, materialID, supplierID)
}
func (s *Service) GetMaterial(ctx context.Context, id uuid.UUID) (domain.RawMaterial, error) {
	return s.repo.GetMaterial(ctx, id)
}
func (s *Service) ListMaterials(ctx context.Context, limit, offset int) ([]domain.RawMaterial, error) {
	return s.repo.ListMaterials(ctx, limit, offset)
}
func (s *Service) CreateOffer(ctx context.Context, offer domain.SupplierMaterialOffer) (domain.SupplierMaterialOffer, error) {
	supplier, err := s.repo.GetSupplier(ctx, offer.SupplierID)
	if err != nil {
		return domain.SupplierMaterialOffer{}, err
	}
	if supplier.Status != domain.SupplierActive {
		return domain.SupplierMaterialOffer{}, domain.ErrInvalidState
	}
	material, err := s.repo.GetMaterial(ctx, offer.MaterialID)
	if err != nil {
		return domain.SupplierMaterialOffer{}, err
	}
	if material.Status != domain.MaterialActive || material.BaseUnit != offer.Unit {
		return domain.SupplierMaterialOffer{}, domain.ErrInvalidState
	}
	return s.repo.CreateOffer(ctx, offer)
}
func (s *Service) SetOfferActive(ctx context.Context, supplierID, offerID uuid.UUID, active bool) (domain.SupplierMaterialOffer, error) {
	return s.repo.SetOfferActive(ctx, supplierID, offerID, active)
}
func (s *Service) ListSupplierOffers(ctx context.Context, id uuid.UUID) ([]domain.SupplierMaterialOffer, error) {
	if _, err := s.repo.GetSupplier(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.ListSupplierOffers(ctx, id)
}
func (s *Service) ListMaterialOffers(ctx context.Context, id uuid.UUID) ([]domain.SupplierMaterialOffer, error) {
	if _, err := s.repo.GetMaterial(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.ListMaterialOffers(ctx, id)
}

func (s *Service) CreateSupplierContract(ctx context.Context, contract domain.SupplierContract) (domain.SupplierContract, error) {
	supplier, err := s.repo.GetSupplier(ctx, contract.SupplierID)
	if err != nil {
		return domain.SupplierContract{}, err
	}
	if supplier.Status != domain.SupplierActive {
		return domain.SupplierContract{}, domain.ErrInvalidState
	}
	return s.repo.CreateSupplierContract(ctx, contract)
}
func (s *Service) GetSupplierContract(ctx context.Context, id uuid.UUID) (domain.SupplierContract, error) {
	return s.repo.GetSupplierContract(ctx, id)
}
func (s *Service) ListSupplierContracts(ctx context.Context, supplierID uuid.UUID, limit, offset int) ([]domain.SupplierContract, error) {
	if supplierID != uuid.Nil {
		if _, err := s.repo.GetSupplier(ctx, supplierID); err != nil {
			return nil, err
		}
	}
	return s.repo.ListSupplierContracts(ctx, supplierID, limit, offset)
}
func (s *Service) SetSupplierContractStatus(ctx context.Context, id uuid.UUID, status domain.SupplierContractStatus, actor string) (domain.SupplierContract, error) {
	if status != domain.SupplierContractActive && status != domain.SupplierContractTerminated {
		return domain.SupplierContract{}, domain.ErrInvalidArgument
	}
	if strings.TrimSpace(actor) == "" {
		return domain.SupplierContract{}, domain.ErrInvalidArgument
	}
	return s.repo.SetSupplierContractStatus(ctx, id, status, actor)
}

type PurchaseOrderLineRequest struct {
	MaterialID uuid.UUID
	Quantity   string
}

type PurchaseOrderRequest struct {
	SupplierID           uuid.UUID
	ExpectedDeliveryDate *time.Time
	Notes                string
	Lines                []PurchaseOrderLineRequest
}

func (s *Service) CreatePurchaseOrder(ctx context.Context, request PurchaseOrderRequest, actor string) (domain.PurchaseOrder, error) {
	order, err := s.buildPurchaseOrder(ctx, "", request, actor)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	return s.repo.CreatePurchaseOrder(ctx, order)
}

func (s *Service) UpdatePurchaseOrderDraft(ctx context.Context, id uuid.UUID, request PurchaseOrderRequest, actor string) (domain.PurchaseOrder, error) {
	current, err := s.repo.GetPurchaseOrder(ctx, id)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	if current.Status != domain.PurchaseOrderDraft {
		return domain.PurchaseOrder{}, domain.ErrInvalidState
	}
	updated, err := s.buildPurchaseOrder(ctx, current.Number, request, current.CreatedBy)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	if strings.TrimSpace(actor) == "" {
		return domain.PurchaseOrder{}, domain.ErrInvalidArgument
	}
	if err := current.ReplaceDraft(updated); err != nil {
		return domain.PurchaseOrder{}, err
	}
	return s.repo.SavePurchaseOrder(ctx, current, current.Version-1, domain.PurchaseOrderDraft, "DRAFT_UPDATED", actor, "")
}

func (s *Service) GetPurchaseOrder(ctx context.Context, id uuid.UUID) (domain.PurchaseOrder, error) {
	return s.repo.GetPurchaseOrder(ctx, id)
}

func (s *Service) ListPurchaseOrders(ctx context.Context, supplierID uuid.UUID, status domain.PurchaseOrderStatus, limit, offset int) ([]domain.PurchaseOrder, error) {
	if status != "" && !validPurchaseOrderStatus(status) {
		return nil, domain.ErrInvalidArgument
	}
	return s.repo.ListPurchaseOrders(ctx, supplierID, status, limit, offset)
}

func (s *Service) SubmitPurchaseOrder(ctx context.Context, id uuid.UUID, actor string) (domain.PurchaseOrder, error) {
	return s.transitionPurchaseOrder(ctx, id, actor, "", "SUBMITTED", func(order *domain.PurchaseOrder) error { return order.Submit(actor) })
}

func (s *Service) ApprovePurchaseOrder(ctx context.Context, id uuid.UUID, actor string) (domain.PurchaseOrder, error) {
	return s.transitionPurchaseOrder(ctx, id, actor, "", "APPROVED", func(order *domain.PurchaseOrder) error { return order.Approve(actor) })
}

func (s *Service) RejectPurchaseOrder(ctx context.Context, id uuid.UUID, actor, reason string) (domain.PurchaseOrder, error) {
	return s.transitionPurchaseOrder(ctx, id, actor, reason, "REJECTED", func(order *domain.PurchaseOrder) error { return order.Reject(actor, reason) })
}

func (s *Service) MarkPurchaseOrderSent(ctx context.Context, id uuid.UUID, actor string) (domain.PurchaseOrder, error) {
	current, err := s.repo.GetPurchaseOrder(ctx, id)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	switch current.Status {
	case domain.PurchaseOrderSent, domain.PurchaseOrderAcknowledged, domain.PurchaseOrderChangeRequested, domain.PurchaseOrderSupplierDeclined, domain.PurchaseOrderPartiallyReceived, domain.PurchaseOrderReceived, domain.PurchaseOrderClosed:
		return current, nil
	}
	previousStatus, expectedVersion := current.Status, current.Version
	if err := current.MarkSent(actor); err != nil {
		return domain.PurchaseOrder{}, err
	}
	return s.repo.SavePurchaseOrder(ctx, current, expectedVersion, previousStatus, "SENT", actor, "")
}

func (s *Service) RecordSupplierResponse(ctx context.Context, id uuid.UUID, actor string, response domain.SupplierPOResponse, note string) (domain.PurchaseOrder, error) {
	action := string(response)
	return s.transitionPurchaseOrder(ctx, id, actor, note, "SUPPLIER_"+action, func(order *domain.PurchaseOrder) error { return order.RecordSupplierResponse(response, actor, note) })
}

func (s *Service) RevisePurchaseOrder(ctx context.Context, id uuid.UUID, request PurchaseOrderRequest, actor, reason string) (domain.PurchaseOrder, error) {
	current, err := s.repo.GetPurchaseOrder(ctx, id)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	expectedVersion := current.Version
	updated, err := s.buildPurchaseOrder(ctx, current.Number, request, actor)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	if err := current.ReviseAsDraft(updated, actor, reason); err != nil {
		return domain.PurchaseOrder{}, err
	}
	return s.repo.RevisePurchaseOrder(ctx, current, expectedVersion, actor, reason)
}

func (s *Service) ListPurchaseOrderRevisions(ctx context.Context, id uuid.UUID, limit, offset int) ([]domain.PurchaseOrderRevision, error) {
	if _, err := s.repo.GetPurchaseOrder(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.ListPurchaseOrderRevisions(ctx, id, limit, offset)
}

func (s *Service) CloseRemainingPurchaseOrder(ctx context.Context, id uuid.UUID, actor, reason string) (domain.PurchaseOrder, error) {
	return s.transitionPurchaseOrder(ctx, id, actor, reason, "CLOSED", func(order *domain.PurchaseOrder) error { return order.CloseRemaining(actor, reason) })
}

func (s *Service) CancelPurchaseOrder(ctx context.Context, id uuid.UUID, actor, reason string) (domain.PurchaseOrder, error) {
	return s.transitionPurchaseOrder(ctx, id, actor, reason, "CANCELLED", func(order *domain.PurchaseOrder) error { return order.Cancel(actor, reason) })
}

func (s *Service) transitionPurchaseOrder(ctx context.Context, id uuid.UUID, actor, reason, action string, transition func(*domain.PurchaseOrder) error) (domain.PurchaseOrder, error) {
	current, err := s.repo.GetPurchaseOrder(ctx, id)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	previousStatus, expectedVersion := current.Status, current.Version
	if err := transition(&current); err != nil {
		return domain.PurchaseOrder{}, err
	}
	return s.repo.SavePurchaseOrder(ctx, current, expectedVersion, previousStatus, action, actor, reason)
}

type GoodsReceiptLineRequest struct {
	PurchaseOrderLineID uuid.UUID
	QuantityReceived    string
	SupplierLotCode     string
	ManufacturedOn      *time.Time
	ExpiresOn           *time.Time
}

type GoodsReceiptRequest struct {
	PurchaseOrderID uuid.UUID
	WarehouseCode   string
	Notes           string
	Lines           []GoodsReceiptLineRequest
}

func (s *Service) CreateGoodsReceipt(ctx context.Context, request GoodsReceiptRequest, actor string) (domain.GoodsReceipt, error) {
	order, err := s.repo.GetPurchaseOrder(ctx, request.PurchaseOrderID)
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	lineByID := make(map[uuid.UUID]domain.PurchaseOrderLine, len(order.Lines))
	for _, line := range order.Lines {
		lineByID[line.ID] = line
	}
	inputs := make([]domain.GoodsReceiptLineInput, 0, len(request.Lines))
	for _, requested := range request.Lines {
		line, exists := lineByID[requested.PurchaseOrderLineID]
		if !exists {
			return domain.GoodsReceipt{}, domain.ErrInvalidArgument
		}
		inputs = append(inputs, domain.GoodsReceiptLineInput{PurchaseOrderLineID: line.ID, MaterialID: line.MaterialID,
			MaterialCode: line.MaterialCode, MaterialName: line.MaterialName, Unit: line.Unit,
			QuantityReceived: requested.QuantityReceived, SupplierLotCode: requested.SupplierLotCode,
			ManufacturedOn: requested.ManufacturedOn, ExpiresOn: requested.ExpiresOn})
	}
	number := "GR-" + time.Now().UTC().Format("20060102") + "-" + strings.ToUpper(uuid.NewString()[:8])
	receipt, err := domain.NewGoodsReceipt(number, order, request.WarehouseCode, actor, request.Notes, inputs)
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	return s.repo.CreateGoodsReceipt(ctx, receipt)
}

func (s *Service) GetGoodsReceipt(ctx context.Context, id uuid.UUID) (domain.GoodsReceipt, error) {
	return s.repo.GetGoodsReceipt(ctx, id)
}

func (s *Service) InspectGoodsReceipt(ctx context.Context, id uuid.UUID, actor string, inspections []domain.GoodsReceiptInspectionLine) (domain.GoodsReceipt, error) {
	receipt, err := s.repo.GetGoodsReceipt(ctx, id)
	if err != nil {
		return domain.GoodsReceipt{}, err
	}
	expectedVersion := receipt.Version
	if err := receipt.Inspect(actor, inspections); err != nil {
		return domain.GoodsReceipt{}, err
	}
	return s.repo.InspectGoodsReceipt(ctx, receipt, expectedVersion, actor)
}

func (s *Service) buildPurchaseOrder(ctx context.Context, number string, request PurchaseOrderRequest, actor string) (domain.PurchaseOrder, error) {
	supplier, err := s.repo.GetSupplier(ctx, request.SupplierID)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	if supplier.Status != domain.SupplierActive {
		return domain.PurchaseOrder{}, domain.ErrInvalidState
	}
	offers, err := s.repo.ListSupplierOffers(ctx, supplier.ID)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	offerByMaterial := make(map[uuid.UUID]domain.SupplierMaterialOffer, len(offers))
	for _, offer := range offers {
		if offer.EffectiveAt(time.Now().UTC()) {
			offerByMaterial[offer.MaterialID] = offer
		}
	}
	lineInputs := make([]domain.PurchaseOrderLineInput, 0, len(request.Lines))
	currency := ""
	for _, requested := range request.Lines {
		material, materialErr := s.repo.GetMaterial(ctx, requested.MaterialID)
		if materialErr != nil {
			return domain.PurchaseOrder{}, materialErr
		}
		if material.Status != domain.MaterialActive {
			return domain.PurchaseOrder{}, domain.ErrInvalidState
		}
		offer, exists := offerByMaterial[material.ID]
		if !exists || offer.Unit != material.BaseUnit {
			return domain.PurchaseOrder{}, domain.ErrInvalidState
		}
		requestedQuantity, requestedOK := new(big.Rat).SetString(strings.TrimSpace(requested.Quantity))
		minimumQuantity, minimumOK := new(big.Rat).SetString(offer.MinimumQty)
		if !requestedOK || !minimumOK || requestedQuantity.Cmp(minimumQuantity) < 0 {
			return domain.PurchaseOrder{}, domain.ErrInvalidArgument
		}
		if currency == "" {
			currency = offer.Currency
		} else if currency != offer.Currency {
			return domain.PurchaseOrder{}, domain.ErrInvalidArgument
		}
		lineInputs = append(lineInputs, domain.PurchaseOrderLineInput{
			MaterialID: material.ID, MaterialCode: material.Code, MaterialName: material.Name,
			Quantity: requested.Quantity, Unit: offer.Unit, UnitPriceMinor: offer.UnitPriceMinor,
		})
	}
	if number == "" {
		number = "PO-" + time.Now().UTC().Format("20060102") + "-" + strings.ToUpper(uuid.NewString()[:8])
	}
	return domain.NewPurchaseOrder(number, supplier.ID, supplier.Name, currency, request.Notes, actor, request.ExpectedDeliveryDate, lineInputs)
}

func validPurchaseOrderStatus(status domain.PurchaseOrderStatus) bool {
	switch status {
	case domain.PurchaseOrderDraft, domain.PurchaseOrderPendingApproval, domain.PurchaseOrderApproved,
		domain.PurchaseOrderRejected, domain.PurchaseOrderSent, domain.PurchaseOrderAcknowledged, domain.PurchaseOrderChangeRequested,
		domain.PurchaseOrderSupplierDeclined, domain.PurchaseOrderPartiallyReceived, domain.PurchaseOrderReceived, domain.PurchaseOrderCancelled, domain.PurchaseOrderClosed:
		return true
	default:
		return false
	}
}
