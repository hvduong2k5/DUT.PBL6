package grpc

import (
	"context"
	"testing"

	traceabilityv1 "dut-pbl6/inventory-service/pkg/proto/traceability/v1"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestTraceabilitySourceRequiresCredentialAndExactBinding(t *testing.T) {
	const token = "traceability-test-credential-0123456789"
	id := uuid.New()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := TraceabilitySource{DB: db, Token: token}
	req := &traceabilityv1.BatchRequest{BatchId: id.String(), SkuCode: "SKU-01"}
	if _, err := server.GetTraceabilityBatch(context.Background(), req); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	columns := []string{"id", "sku", "batch_code", "production_date", "expiry_date", "restricted", "depleted", "version"}
	mock.ExpectQuery("FROM batches WHERE id =").WithArgs(id, "SKU-01").
		WillReturnRows(sqlmock.NewRows(columns).AddRow(id.String(), "SKU-01", "LOT-01", "2026-11-13", "2027-01-01", true, true, 9))
	snapshot, err := server.GetTraceabilityBatch(ctx, req)
	if err != nil || !snapshot.GetRestricted() || !snapshot.GetDepleted() || snapshot.GetBatchVersion() != 9 {
		t.Fatalf("expected restricted, depleted batch at version 9, got %v (%v)", snapshot, err)
	}
	req.SkuCode = "wrong"
	mock.ExpectQuery("FROM batches WHERE id =").WithArgs(id, "wrong").WillReturnRows(sqlmock.NewRows(columns))
	if _, err := server.GetTraceabilityBatch(ctx, req); status.Code(err) != codes.NotFound {
		t.Fatalf("expected not found, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
