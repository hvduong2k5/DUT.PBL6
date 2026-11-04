package grpc

import (
	"context"
	"dut-pbl6/inventory-service/internal/application/usecase"
	"dut-pbl6/inventory-service/internal/infrastructure/postgres"
	api "dut-pbl6/inventory-service/pkg/proto/inventory/v1"
)

// The terminal cohort also serves authenticated display reads using the owning
// Inventory query use case. It never delegates through a possibly nil embedded server.
func (s *TerminalServer) GetStockLevel(ctx context.Context, request *api.GetStockLevelRequest) (*api.GetStockLevelResponse, error) {
	if err := s.authorize(ctx); err != nil {
		return nil, err
	}
	query := usecase.NewGetStockLevelUseCase(postgres.NewPostgresInventoryRepository(s.DB))
	return NewInventoryGRPCHandler(nil, nil, query, nil).GetStockLevel(ctx, request)
}
