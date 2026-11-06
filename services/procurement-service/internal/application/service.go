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
	SetMaterialStatus(context.Context, uuid.UUID, domain.MaterialStatus) (domain.RawMaterial, error)
	GetMaterial(context.Context, uuid.UUID) (domain.RawMaterial, error)
	ListMaterials(context.Context, int, int) ([]domain.RawMaterial, error)
	CreateOffer(context.Context, domain.SupplierMaterialOffer) (domain.SupplierMaterialOffer, error)
	ListSupplierOffers(context.Context, uuid.UUID) ([]domain.SupplierMaterialOffer, error)
	CreatePurchaseOrder(context.Context, domain.PurchaseOrder) (domain.PurchaseOrder, error)
	GetPurchaseOrder(context.Context, uuid.UUID) (domain.PurchaseOrder, error)
	ListPurchaseOrders(context.Context, uuid.UUID, domain.PurchaseOrderStatus, int, int) ([]domain.PurchaseOrder, error)
	SavePurchaseOrder(context.Context, domain.PurchaseOrder, int64, domain.PurchaseOrderStatus, string, string, string) (domain.PurchaseOrder, error)
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
func (s *Service) CreateMaterial(ctx context.Context, material domain.RawMaterial) (domain.RawMaterial, error) {
	return s.repo.CreateMaterial(ctx, material)
}
func (s *Service) SetMaterialStatus(ctx context.Context, id uuid.UUID, status domain.MaterialStatus) (domain.RawMaterial, error) {
	return s.repo.SetMaterialStatus(ctx, id, status)
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
func (s *Service) ListSupplierOffers(ctx context.Context, id uuid.UUID) ([]domain.SupplierMaterialOffer, error) {
	if _, err := s.repo.GetSupplier(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.ListSupplierOffers(ctx, id)
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
	return s.transitionPurchaseOrder(ctx, id, actor, "", "SENT", func(order *domain.PurchaseOrder) error { return order.MarkSent(actor) })
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
		if offer.Active {
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
		domain.PurchaseOrderRejected, domain.PurchaseOrderSent, domain.PurchaseOrderPartiallyReceived,
		domain.PurchaseOrderReceived, domain.PurchaseOrderCancelled:
		return true
	default:
		return false
	}
}
