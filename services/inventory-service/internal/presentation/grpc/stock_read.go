package grpc

import (
	"context"
	"dut-pbl6/inventory-service/internal/application/usecase"
	"dut-pbl6/inventory-service/internal/infrastructure/postgres"
	inventoryv1 "dut-pbl6/inventory-service/pkg/proto/inventory/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// The terminal cohort also serves authenticated display reads using the owning
// Inventory query use case. It never delegates through a possibly nil embedded server.
func (s *TerminalServer) GetStockLevel(ctx context.Context, r *inventoryv1.GetStockLevelRequest) (*inventoryv1.GetStockLevelResponse, error) {
	if err := s.authorizeRead(ctx); err != nil {
		return nil, err
	}
	if r != nil && r.WarehouseId != nil {
		return nil, status.Error(codes.InvalidArgument, "warehouse-scoped stock reads are not supported")
	}
	repo := postgres.NewPostgresInventoryRepository(s.DB)
	h := NewInventoryGRPCHandler(nil, nil, usecase.NewGetStockLevelUseCase(repo), nil)
	return h.GetStockLevel(ctx, r)
}
