package simulator

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain/core"
	common "github.com/omamx/order-service/internal/gen/common/v1"
	inventory "github.com/omamx/order-service/internal/gen/inventory/v1"
	"github.com/omamx/order-service/internal/paymentinventory"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strings"
	"time"
)

func (r *InventoryRPC) coreReserve(ctx context.Context, in *inventory.ReserveStockRequest) (*inventory.ReserveStockResponse, error) {
	if in.PreferredWarehouseId == nil || *in.PreferredWarehouseId != "HUE" || in.TtlMinutes != 15 || in.Channel != "D2C_WEB" || len(in.IdempotencyKey) < 16 {
		return nil, status.Error(codes.InvalidArgument, "target reserve contract")
	}
	lines := []core.SelectionLine{}
	for _, i := range in.Items {
		lines = append(lines, core.SelectionLine{SKU: i.SkuCode, Quantity: int64(i.Quantity)})
	}
	digest, e := core.HashManifest(lines)
	if e != nil || string(digest) != in.ItemsDigest || in.RequestHash != paymentinventory.ReserveHash(in.OrderId, in.ItemsDigest, *in.PreferredWarehouseId) {
		return nil, status.Error(codes.InvalidArgument, "manifest/hash")
	}
	out, e := r.coreEffect(ctx, "RESERVE", &inventory.ReservationCommand{OrderId: in.OrderId, IdempotencyKey: in.IdempotencyKey, RequestHash: in.RequestHash, ItemsDigest: in.ItemsDigest, WarehouseId: *in.PreferredWarehouseId}, lines)
	if e != nil {
		return nil, e
	}
	return &inventory.ReserveStockResponse{Status: inventory.ReservationStatus_RESERVATION_STATUS_SUCCESS, ReservationId: &out.ReservationId, ExpiresAt: out.ExpiresAt, Outcome: out}, nil
}
func (r *InventoryRPC) coreCommand(ctx context.Context, action string, in *inventory.ReservationCommand) (*inventory.ReservationOutcome, error) {
	if action == "query" {
		return r.coreEffect(ctx, "QUERY", in, nil)
	}
	action = strings.ToUpper(action)
	if action != "FINALIZE" && action != "RELEASE" {
		return nil, status.Error(codes.Unimplemented, "target action")
	}
	if in.ExpectedReservationId == nil || in.ExpectedVersion == nil || in.RequestHash != paymentinventory.CommandHash(action, in.OrderId, *in.ExpectedReservationId, in.ItemsDigest, in.WarehouseId) {
		return nil, status.Error(codes.InvalidArgument, "command hash/association")
	}
	out, e := r.coreEffect(ctx, action, in, nil)
	if e == nil && r.S.Fault(ctx, "core_"+strings.ToLower(action)) == "lost_ack" {
		return nil, status.Error(codes.Unavailable, "injected ACK loss")
	}
	return out, e
}
func (r *InventoryRPC) coreEffect(ctx context.Context, action string, in *inventory.ReservationCommand, lines []core.SelectionLine) (*inventory.ReservationOutcome, error) {
	if _, e := uuid.Parse(in.OrderId); e != nil {
		return nil, status.Error(codes.InvalidArgument, "order")
	}
	tx, e := r.S.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", "core-inventory:"+in.OrderId); e != nil {
		return nil, e
	}
	out := &inventory.ReservationOutcome{OrderId: in.OrderId, ContractVersion: "order-3.2"}
	var raw []byte
	var expiry, now time.Time
	e = tx.QueryRow(ctx, "SELECT id,digest,warehouse,items,state,version,expires_at FROM core_inventory WHERE order_id=$1 FOR UPDATE", in.OrderId).Scan(&out.ReservationId, &out.ItemsDigest, &out.WarehouseId, &raw, &out.State, &out.ResourceVersion, &expiry)
	exists := e == nil
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return nil, e
	}
	if !exists && action != "RESERVE" {
		return nil, status.Error(codes.NotFound, "target reservation not found")
	}
	if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
		return nil, e
	}
	if !exists {
		for _, i := range lines {
			var available int64
			if e = tx.QueryRow(ctx, "SELECT available FROM stock WHERE sku=$1 FOR UPDATE", i.SKU).Scan(&available); e != nil || available < i.Quantity {
				return nil, status.Error(codes.ResourceExhausted, "insufficient stock")
			}
		}
		for _, i := range lines {
			if _, e = tx.Exec(ctx, "UPDATE stock SET available=available-$2 WHERE sku=$1", i.SKU, i.Quantity); e != nil {
				return nil, e
			}
		}
		out.ReservationId = uuid.NewString()
		out.ItemsDigest = in.ItemsDigest
		out.WarehouseId = in.WarehouseId
		out.State = "RESERVED"
		out.ResourceVersion = 1
		expiry = now.Add(15 * time.Minute)
		raw, _ = json.Marshal(lines)
		if _, e = tx.Exec(ctx, "INSERT INTO core_inventory(order_id,id,digest,warehouse,items,state,version,expires_at) VALUES($1,$2,$3,$4,$5,'RESERVED',1,$6)", in.OrderId, out.ReservationId, in.ItemsDigest, in.WarehouseId, raw, expiry); e != nil {
			return nil, e
		}
	}
	if out.ItemsDigest != in.ItemsDigest || out.WarehouseId != in.WarehouseId || in.ExpectedReservationId != nil && *in.ExpectedReservationId != out.ReservationId {
		return nil, status.Error(codes.FailedPrecondition, "target association")
	}
	var held []core.SelectionLine
	if json.Unmarshal(raw, &held) != nil {
		return nil, status.Error(codes.Internal, "stored manifest")
	}
	if out.State == "RESERVED" && !now.Before(expiry) {
		for _, i := range held {
			if _, e = tx.Exec(ctx, "UPDATE stock SET available=available+$2 WHERE sku=$1", i.SKU, i.Quantity); e != nil {
				return nil, e
			}
		}
		out.State = "EXPIRED"
		out.ResourceVersion++
	}
	command := &common.CommandEvidence{}
	var decided time.Time
	e = tx.QueryRow(ctx, "SELECT operation_key,request_hash,action,decision_id,result,resulting_state,decided_at FROM core_inventory_commands WHERE order_id=$1 AND operation_key=$2", in.OrderId, in.IdempotencyKey).Scan(&command.IdempotencyKey, &command.RequestHash, &command.Action, &command.DecisionId, &command.Result, &command.ResultingState, &decided)
	if e == nil {
		if command.RequestHash != in.RequestHash {
			return nil, status.Error(codes.AlreadyExists, "idempotency conflict")
		}
		command.DecisionAt = timestamppb.New(decided)
		out.Command = command
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return nil, e
	} else if action != "QUERY" {
		result := "APPLIED"
		if action != "RESERVE" && in.ExpectedVersion != nil && *in.ExpectedVersion != out.ResourceVersion && out.State == "RESERVED" {
			return nil, status.Error(codes.Aborted, "version conflict")
		}
		switch action {
		case "FINALIZE":
			if out.State == "RESERVED" {
				for _, i := range held {
					if _, e = tx.Exec(ctx, "UPDATE stock SET physical=physical-$2 WHERE sku=$1", i.SKU, i.Quantity); e != nil {
						return nil, e
					}
				}
				out.State = "COMMITTED"
				out.ResourceVersion++
			} else if out.State == "COMMITTED" {
				result = "ALREADY_APPLIED"
			} else {
				result = "REJECTED"
			}
		case "RELEASE":
			if out.State == "RESERVED" {
				for _, i := range held {
					if _, e = tx.Exec(ctx, "UPDATE stock SET available=available+$2 WHERE sku=$1", i.SKU, i.Quantity); e != nil {
						return nil, e
					}
				}
				out.State = "RELEASED"
				out.ResourceVersion++
			} else if out.State == "COMMITTED" {
				result = "REJECTED"
			} else {
				result = "ALREADY_APPLIED"
			}
		}
		command = &common.CommandEvidence{IdempotencyKey: in.IdempotencyKey, RequestHash: in.RequestHash, Action: action, DecisionId: uuid.NewString(), Result: result, ResultingState: out.State, DecisionAt: timestamppb.New(now)}
		out.Command = command
		if _, e = tx.Exec(ctx, "INSERT INTO core_inventory_commands(order_id,operation_key,request_hash,action,decision_id,result,resulting_state,decided_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)", in.OrderId, command.IdempotencyKey, command.RequestHash, command.Action, command.DecisionId, command.Result, command.ResultingState, now); e != nil {
			return nil, e
		}
	}
	if _, e = tx.Exec(ctx, "UPDATE core_inventory SET state=$2,version=$3 WHERE order_id=$1", in.OrderId, out.State, out.ResourceVersion); e != nil {
		return nil, e
	}
	out.ExpiresAt = timestamppb.New(expiry)
	out.ObservedAt = timestamppb.New(now)
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	return out, nil
}
