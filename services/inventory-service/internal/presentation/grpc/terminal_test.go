package grpc

import (
	"context"
	commonv1 "dut-pbl6/inventory-service/pkg/proto/common/v1"
	inventoryv1 "dut-pbl6/inventory-service/pkg/proto/inventory/v1"
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"testing"
	"time"
)

func TestCanonicalReserve(t *testing.T) {
	w := "warehouse"
	r := &inventoryv1.ReserveStockRequest{OrderId: "order", ContractVersion: "order-3.2", TtlMinutes: 15, PreferredWarehouseId: &w, ItemsDigest: "47a", Items: []*inventoryv1.ReserveItem{{SkuCode: "B", Quantity: 1}, {SkuCode: "A", Quantity: 1}, {SkuCode: "A", Quantity: 1}}}
	// Independent canonical byte vector, including the exact documented key order.
	r.ItemsDigest = ""
	require.Error(t, normalizeReserve(r))
	require.Equal(t, int32(2), r.Items[0].Quantity)
	require.Equal(t, "A", r.Items[0].SkuCode)
	a := requestHash(r.ProtoReflect())
	r.CorrelationId = "ignored"
	r.IdempotencyKey = "also-ignored"
	r.RequestHash = "ignored"
	require.Equal(t, a, requestHash(r.ProtoReflect()))
	r.PreferredWarehouseId = nil
	require.NotEqual(t, a, requestHash(r.ProtoReflect()))
	r.Items = []*inventoryv1.ReserveItem{{SkuCode: "A", Quantity: 2147483647}, {SkuCode: "A", Quantity: 1}}
	r.PreferredWarehouseId = &w
	require.Error(t, normalizeReserve(r))
}
func TestTerminalTombstoneReplayAndConflict(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	s := &TerminalServer{DB: db, Token: "secret"}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer secret"))
	r := &inventoryv1.ReservationCommand{OrderId: "order", ContractVersion: "order-3.2", IdempotencyKey: "1234567890123456"}
	r.RequestHash = requestHash(r.ProtoReflect())
	begin := func(data, commands []byte) {
		m.ExpectBegin()
		m.ExpectExec("SELECT pg_advisory").WithArgs("order").WillReturnResult(sqlmock.NewResult(0, 1))
		m.ExpectQuery("SELECT EXISTS").WithArgs("order").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
		q := m.ExpectQuery("SELECT outcome,allocations,commands").WithArgs("order")
		if data == nil {
			q.WillReturnRows(sqlmock.NewRows([]string{"outcome", "allocations", "commands"}))
		} else {
			q.WillReturnRows(sqlmock.NewRows([]string{"outcome", "allocations", "commands"}).AddRow(data, []byte("[]"), commands))
		}
		m.ExpectQuery("SELECT clock_timestamp").WillReturnRows(sqlmock.NewRows([]string{"now"}).AddRow(time.Now()))
	}
	begin(nil, nil)
	m.ExpectExec("INSERT INTO inventory_terminal").WillReturnResult(sqlmock.NewResult(0, 1))
	m.ExpectCommit()
	out, err := s.ReleaseReservationByOrder(ctx, r)
	require.NoError(t, err)
	require.True(t, out.CancellationTombstone)
	require.Equal(t, "RELEASED", out.State)
	proof := proto.Clone(out.Command).(*commonv1.CommandEvidence)
	out.Command = nil
	data, err := proto.Marshal(out)
	require.NoError(t, err)
	commands, err := json.Marshal(map[string]*commonv1.CommandEvidence{"ReleaseReservationByOrder:" + r.IdempotencyKey: proof})
	require.NoError(t, err)
	begin(data, commands)
	m.ExpectCommit()
	again, err := s.ReleaseReservationByOrder(ctx, r)
	require.NoError(t, err)
	require.Equal(t, proof.DecisionId, again.Command.DecisionId)
	require.Equal(t, "ALREADY_APPLIED", again.Command.Result)
	r.CodUnpaid = true
	r.RequestHash = requestHash(r.ProtoReflect())
	begin(data, commands)
	m.ExpectRollback()
	_, err = s.ReleaseReservationByOrder(ctx, r)
	require.Equal(t, codes.AlreadyExists, status.Code(err))
	require.NoError(t, m.ExpectationsWereMet())
}
func TestTerminalRejectsUnauthenticatedAndReversal(t *testing.T) {
	s := &TerminalServer{Token: "secret"}
	_, err := s.FinalizeReservation(context.Background(), nil)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	_, err = s.ReverseCommittedStock(context.Background(), nil)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}
