package application

import (
	"context"

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
	return s.repo.CreateOffer(ctx, offer)
}
func (s *Service) ListSupplierOffers(ctx context.Context, id uuid.UUID) ([]domain.SupplierMaterialOffer, error) {
	return s.repo.ListSupplierOffers(ctx, id)
}
