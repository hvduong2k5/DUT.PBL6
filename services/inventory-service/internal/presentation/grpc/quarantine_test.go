package grpc

import (
	"context"
	"dut-pbl6/inventory-service/internal/testdb"
	traceabilityv1 "dut-pbl6/inventory-service/pkg/proto/traceability/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"strings"
	"testing"
)

func TestQuarantinedUnknownDatesRemainRestricted(t *testing.T) {
	db := testdb.Open(t)
	id := uuid.NewString()
	_, err := db.Exec("INSERT INTO inventory_items(sku) VALUES('RAW')")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO batches(id,sku,batch_code,supplier_id,physical_qty,status) VALUES($1,'RAW','QUARANTINE',gen_random_uuid(),100,'QUARANTINE')", id)
	require.NoError(t, err)
	token := strings.Repeat("x", 32)
	server := &TraceabilitySource{DB: db, Token: token}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	snapshot, err := server.GetTraceabilityBatch(ctx, &traceabilityv1.BatchRequest{BatchId: id, SkuCode: "RAW"})
	require.NoError(t, err)
	require.True(t, snapshot.Restricted)
	require.Empty(t, snapshot.ProductionDate)
	require.Empty(t, snapshot.ExpiryDate)
}
