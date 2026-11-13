package grpc

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"

	traceabilityv1 "dut-pbl6/inventory-service/pkg/proto/traceability/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type TraceabilitySource struct {
	traceabilityv1.UnimplementedBatchSourceServer
	DB    *sql.DB
	Token string
}

func (s *TraceabilitySource) GetTraceabilityBatch(ctx context.Context, req *traceabilityv1.BatchRequest) (*traceabilityv1.BatchSnapshot, error) {
	values := metadata.ValueFromIncomingContext(ctx, "authorization")
	if len(s.Token) < 32 || len(values) != 1 || subtle.ConstantTimeCompare([]byte(values[0]), []byte("Bearer "+s.Token)) != 1 {
		return nil, status.Error(codes.Unauthenticated, "traceability credential required")
	}
	id, err := uuid.Parse(req.GetBatchId())
	if err != nil || id == uuid.Nil || id.String() != req.GetBatchId() || req.GetSkuCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "batch UUID and SKU required")
	}
	var snapshot traceabilityv1.BatchSnapshot
	err = s.DB.QueryRowContext(ctx, `
		SELECT id::text, sku, batch_code, to_char(mfg_date,'YYYY-MM-DD'), to_char(exp_date,'YYYY-MM-DD'),
		       status NOT IN ('ACTIVE','NEAR_EXPIRY'), physical_qty = 0, version
		FROM batches WHERE id = $1 AND sku = $2`, id, req.GetSkuCode()).Scan(
		&snapshot.BatchId, &snapshot.SkuCode, &snapshot.BatchCode, &snapshot.ProductionDate,
		&snapshot.ExpiryDate, &snapshot.Restricted, &snapshot.Depleted, &snapshot.BatchVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "batch and SKU binding not found")
	}
	if err != nil {
		return nil, status.Error(codes.Unavailable, "batch source unavailable")
	}
	return &snapshot, nil
}
