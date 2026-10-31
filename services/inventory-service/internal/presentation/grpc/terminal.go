package grpc

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"sort"
	"time"

	"dut-pbl6/inventory-service/internal/domain/entity"
	"dut-pbl6/inventory-service/internal/domain/service"
	"dut-pbl6/inventory-service/internal/infrastructure/postgres"
	commonv1 "dut-pbl6/inventory-service/pkg/proto/common/v1"
	inventoryv1 "dut-pbl6/inventory-service/pkg/proto/inventory/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TerminalServer serves an explicitly enabled internal cohort. The credential must
// be delivered over an authenticated encrypted deployment transport; never expose publicly.
type TerminalServer struct {
	inventoryv1.InventoryServiceServer
	DB    *sql.DB
	Token string
}
type terminalRecord struct {
	outcome     *inventoryv1.ReservationOutcome
	allocations []entity.ReservationItemAllocation
	commands    map[string]*commonv1.CommandEvidence
}

func (s *TerminalServer) authorize(ctx context.Context) error {
	md, _ := metadata.FromIncomingContext(ctx)
	tokens := md.Get("authorization")
	if s.Token == "" || len(tokens) != 1 || subtle.ConstantTimeCompare([]byte(tokens[0]), []byte("Bearer "+s.Token)) != 1 {
		return status.Error(codes.Unauthenticated, "internal Order credential required")
	}
	return nil
}
func terminalError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	if errors.Is(err, entity.ErrInsufficientStock) {
		return status.Error(codes.ResourceExhausted, "insufficient stock")
	}
	if errors.Is(err, entity.ErrItemNotFound) {
		return status.Error(codes.NotFound, "SKU not found")
	}
	return status.Error(codes.Unavailable, "inventory transaction failed; recover with original command identity")
}
func (s *TerminalServer) transact(ctx context.Context, order string, fn func(*sql.Tx, *terminalRecord, time.Time) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return terminalError(err)
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,731))`, order); err != nil {
		return terminalError(err)
	}
	var legacy bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM stock_reservations WHERE order_id=$1)`, order).Scan(&legacy); err != nil {
		return terminalError(err)
	}
	if legacy {
		return status.Error(codes.FailedPrecondition, "legacy order cannot enter terminal cohort")
	}
	rec := &terminalRecord{commands: map[string]*commonv1.CommandEvidence{}}
	var data, alloc, commands []byte
	err = tx.QueryRowContext(ctx, `SELECT outcome,allocations,commands FROM inventory_terminal WHERE order_id=$1 FOR UPDATE`, order).Scan(&data, &alloc, &commands)
	if err == nil {
		rec.outcome = &inventoryv1.ReservationOutcome{}
		if err = proto.Unmarshal(data, rec.outcome); err != nil {
			return terminalError(err)
		}
		if err = json.Unmarshal(alloc, &rec.allocations); err != nil {
			return terminalError(err)
		}
		if err = json.Unmarshal(commands, &rec.commands); err != nil {
			return terminalError(err)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return terminalError(err)
	}
	var now time.Time
	if err = tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return terminalError(err)
	}
	if err = fn(tx, rec, now); err != nil {
		return terminalError(err)
	}
	return terminalError(tx.Commit())
}
func saveTerminal(ctx context.Context, tx *sql.Tx, r *terminalRecord) error {
	clean := proto.Clone(r.outcome).(*inventoryv1.ReservationOutcome)
	clean.Command = nil
	data, err := proto.Marshal(clean)
	if err != nil {
		return err
	}
	a, err := json.Marshal(r.allocations)
	if err != nil {
		return err
	}
	c, err := json.Marshal(r.commands)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO inventory_terminal(order_id,outcome,allocations,commands) VALUES($1,$2,$3,$4) ON CONFLICT(order_id) DO UPDATE SET outcome=$2,allocations=$3,commands=$4`, r.outcome.OrderId, data, string(a), string(c))
	return err
}
func evidence(r *terminalRecord, action, key, hash, result, reason string, now time.Time) {
	e := &commonv1.CommandEvidence{Action: action, IdempotencyKey: key, RequestHash: hash, Result: result, ReasonCode: reason, DecisionId: uuid.NewString(), DecisionAt: timestamppb.New(now), ResultingState: r.outcome.State}
	r.commands[action+":"+key] = e
	r.outcome.Command = proto.Clone(e).(*commonv1.CommandEvidence)
}
func replay(r *terminalRecord, action, key, hash string) (bool, error) {
	e := r.commands[action+":"+key]
	if e == nil {
		return false, nil
	}
	if e.RequestHash != hash {
		return false, status.Error(codes.AlreadyExists, "idempotency hash conflict")
	}
	r.outcome.Command = proto.Clone(e).(*commonv1.CommandEvidence)
	if e.Result == "APPLIED" {
		r.outcome.Command.Result = "ALREADY_APPLIED"
	}
	return true, nil
}
func (s *TerminalServer) ReserveStock(ctx context.Context, req *inventoryv1.ReserveStockRequest) (*inventoryv1.ReserveStockResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if req.ContractVersion == "" {
		return s.InventoryServiceServer.ReserveStock(ctx, req)
	}
	if err := s.authorize(ctx); err != nil {
		return nil, err
	}
	req = proto.Clone(req).(*inventoryv1.ReserveStockRequest)
	if err := normalizeReserve(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if !validKey(req.IdempotencyKey) || req.RequestHash != requestHash(req.ProtoReflect()) {
		return nil, status.Error(codes.InvalidArgument, "invalid key/request_hash")
	}
	var out *inventoryv1.ReservationOutcome
	err := s.transact(ctx, req.OrderId, func(tx *sql.Tx, r *terminalRecord, now time.Time) error {
		if found, err := replay(r, "ReserveStock", req.IdempotencyKey, req.RequestHash); err != nil {
			return err
		} else if found {
			r.outcome.ObservedAt = timestamppb.New(now)
			out = r.outcome
			return nil
		}
		if r.outcome != nil {
			return status.Error(codes.AlreadyExists, "order already has reservation or cancellation tombstone")
		}
		repo := postgres.NewPostgresInventoryRepository(s.DB)
		st := &postgres.SQLTransaction{Tx: tx}
		for _, i := range req.Items {
			item, err := repo.GetItemBySKUForUpdate(ctx, st, i.SkuCode)
			if err != nil {
				return err
			}
			if item.WarehouseID.String() != req.GetPreferredWarehouseId() || item.Status != entity.ItemStatusActive {
				return status.Error(codes.FailedPrecondition, "warehouse mismatch or inactive SKU")
			}
			batches, err := repo.GetActiveBatchesBySKUForUpdate(ctx, st, i.SkuCode)
			if err != nil {
				return err
			}
			allocations, err := service.AllocateByFEFOAt(batches, int(i.Quantity), now)
			if err != nil {
				return err
			}
			if err = item.Reserve(int(i.Quantity)); err != nil {
				return err
			}
			if err = repo.UpdateItem(ctx, st, item); err != nil {
				return err
			}
			for _, b := range batches {
				if err = repo.UpdateBatch(ctx, st, b); err != nil {
					return err
				}
			}
			for _, a := range allocations {
				r.allocations = append(r.allocations, entity.ReservationItemAllocation{SKU: a.SKU, BatchID: a.BatchID, AllocatedQty: a.AllocatedQty})
			}
		}
		r.outcome = &inventoryv1.ReservationOutcome{OrderId: req.OrderId, ReservationId: uuid.NewString(), State: "RESERVED", ExpiresAt: timestamppb.New(now.Add(time.Duration(req.TtlMinutes) * time.Minute)), ContractVersion: "order-3.2", ItemsDigest: req.ItemsDigest, WarehouseId: req.GetPreferredWarehouseId(), ResourceVersion: 1, ObservedAt: timestamppb.New(now)}

		for _, i := range req.Items {
			item, e := repo.GetItemBySKUForUpdate(ctx, st, i.SkuCode)
			if e != nil {
				return e
			}
			ce, e := entity.NewStockReservedCloudEvent(r.outcome.ReservationId, req.OrderId, i.SkuCode, r.outcome.WarehouseId, int(i.Quantity), item.AvailableQty(), r.outcome.ExpiresAt.AsTime(), "")
			if e != nil {
				return e
			}
			event, e := entity.NewOutboxEvent("StockReservation", r.outcome.ReservationId, entity.EventTypeStockReserved, ce)
			if e != nil {
				return e
			}
			if e = postgres.NewPostgresOutboxRepository(s.DB).SaveEvent(ctx, st, event); e != nil {
				return e
			}
		}
		evidence(r, "ReserveStock", req.IdempotencyKey, req.RequestHash, "APPLIED", "", now)
		out = r.outcome
		return saveTerminal(ctx, tx, r)
	})
	if err != nil {
		return nil, err
	}
	id := out.ReservationId
	return &inventoryv1.ReserveStockResponse{Status: inventoryv1.ReservationStatus_RESERVATION_STATUS_SUCCESS, ReservationId: &id, ExpiresAt: out.ExpiresAt, Outcome: out}, nil
}

// mutateStock holds item locks in SKU order then batch locks in UUID order.
func (s *TerminalServer) mutateStock(ctx context.Context, tx *sql.Tx, r *terminalRecord, commit bool) error {
	repo := postgres.NewPostgresInventoryRepository(s.DB)
	st := &postgres.SQLTransaction{Tx: tx}
	skus := map[string]int{}
	batches := map[uuid.UUID]int{}
	for _, a := range r.allocations {
		skus[a.SKU] += a.AllocatedQty
		batches[a.BatchID] += a.AllocatedQty
	}
	committedItems := []entity.StockCommittedItem{}
	names := []string{}
	for n := range skus {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		item, err := repo.GetItemBySKUForUpdate(ctx, st, n)
		if err != nil {
			return err
		}
		if commit {
			err = item.Commit(skus[n])
		} else {
			err = item.Release(skus[n])
		}
		if err != nil {
			return err
		}
		if err = repo.UpdateItem(ctx, st, item); err != nil {
			return err
		}
		if commit {
			committedItems = append(committedItems, entity.StockCommittedItem{SKUCode: n, Quantity: skus[n]})
		} else {
			ce, e := entity.NewStockReleasedCloudEvent(r.outcome.ReservationId, r.outcome.OrderId, n, r.outcome.WarehouseId, skus[n], item.AvailableQty(), "ORDER_CANCELLED_OR_EXPIRED", "")
			if e != nil {
				return e
			}
			event, e := entity.NewOutboxEvent("StockReservation", r.outcome.ReservationId, entity.EventTypeStockReleased, ce)
			if e != nil {
				return e
			}
			if e = postgres.NewPostgresOutboxRepository(s.DB).SaveEvent(ctx, st, event); e != nil {
				return e
			}
		}
	}
	ids := []uuid.UUID{}
	for id := range batches {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	for _, id := range ids {
		b, err := repo.GetBatchByID(ctx, st, id)
		if err != nil {
			return err
		}
		if commit {
			err = b.Commit(batches[id])
		} else {
			err = b.Release(batches[id])
		}
		if err != nil {
			return err
		}
		if err = repo.UpdateBatch(ctx, st, b); err != nil {
			return err
		}
	}
	if commit {
		ce, e := entity.NewStockCommittedCloudEvent(r.outcome.ReservationId, r.outcome.OrderId, r.outcome.WarehouseId, committedItems, "")
		if e != nil {
			return e
		}
		event, e := entity.NewOutboxEvent("StockReservation", r.outcome.ReservationId, entity.EventTypeStockCommitted, ce)
		if e != nil {
			return e
		}
		if e = postgres.NewPostgresOutboxRepository(s.DB).SaveEvent(ctx, st, event); e != nil {
			return e
		}
	}
	return nil
}
func (s *TerminalServer) command(ctx context.Context, req *inventoryv1.ReservationCommand, action string) (*inventoryv1.ReservationOutcome, error) {
	if err := s.authorize(ctx); err != nil {
		return nil, err
	}
	if req == nil || req.OrderId == "" || req.ContractVersion != "order-3.2" || !validKey(req.IdempotencyKey) || req.RequestHash != requestHash(req.ProtoReflect()) {
		return nil, status.Error(codes.InvalidArgument, "invalid command identity/hash")
	}
	var out *inventoryv1.ReservationOutcome
	err := s.transact(ctx, req.OrderId, func(tx *sql.Tx, r *terminalRecord, now time.Time) error {
		if found, err := replay(r, action, req.IdempotencyKey, req.RequestHash); err != nil {
			return err
		} else if found {
			r.outcome.ObservedAt = timestamppb.New(now)
			out = r.outcome
			return nil
		}
		if r.outcome == nil {
			if action != "ReleaseReservationByOrder" {
				return status.Error(codes.NotFound, "reservation not found")
			}
			if req.ExpectedReservationId != nil || req.ExpectedVersion != nil || req.ItemsDigest != "" || req.WarehouseId != "" {
				return status.Error(codes.FailedPrecondition, "expectations cannot match absent reservation")
			}
			r.outcome = &inventoryv1.ReservationOutcome{OrderId: req.OrderId, State: "RELEASED", ContractVersion: "order-3.2", ResourceVersion: 1, CancellationTombstone: true}
		} else {
			o := r.outcome
			if (req.ExpectedReservationId != nil && *req.ExpectedReservationId != o.ReservationId) || (req.ExpectedVersion != nil && *req.ExpectedVersion != o.ResourceVersion) || (req.ItemsDigest != "" && req.ItemsDigest != o.ItemsDigest) || (req.WarehouseId != "" && req.WarehouseId != o.WarehouseId) {
				return status.Error(codes.FailedPrecondition, "resource expectation mismatch")
			}
			if o.State == "RESERVED" {
				expired := !now.Before(o.ExpiresAt.AsTime())
				commit := action == "FinalizeReservation" && !expired
				if err := s.mutateStock(ctx, tx, r, commit); err != nil {
					return err
				}
				if expired {
					o.State = "EXPIRED"
				} else if commit {
					o.State = "COMMITTED"
				} else {
					o.State = "RELEASED"
				}
				o.ResourceVersion++
			}
		}
		result, reason := "APPLIED", ""
		if (action == "FinalizeReservation" && r.outcome.State != "COMMITTED") || (action == "ReleaseReservationByOrder" && r.outcome.State == "COMMITTED") {
			result = "REJECTED"
			reason = "TERMINAL_STATE"
		}
		r.outcome.ObservedAt = timestamppb.New(now)
		evidence(r, action, req.IdempotencyKey, req.RequestHash, result, reason, now)
		out = r.outcome
		return saveTerminal(ctx, tx, r)
	})
	return out, err
}
func (s *TerminalServer) FinalizeReservation(ctx context.Context, r *inventoryv1.ReservationCommand) (*inventoryv1.ReservationOutcome, error) {
	return s.command(ctx, r, "FinalizeReservation")
}
func (s *TerminalServer) ReleaseReservationByOrder(ctx context.Context, r *inventoryv1.ReservationCommand) (*inventoryv1.ReservationOutcome, error) {
	return s.command(ctx, r, "ReleaseReservationByOrder")
}
func (s *TerminalServer) ReverseCommittedStock(context.Context, *inventoryv1.ReservationCommand) (*inventoryv1.ReservationOutcome, error) {
	return nil, status.Error(codes.FailedPrecondition, "reversal gated: approval and remote stop authority not accepted")
}
func (s *TerminalServer) GetReservationByOrder(ctx context.Context, req *inventoryv1.ReservationCommand) (*inventoryv1.ReservationOutcome, error) {
	if err := s.authorize(ctx); err != nil {
		return nil, err
	}
	if req == nil || req.OrderId == "" || req.ContractVersion != "order-3.2" || ((req.QueryAction == "") != (req.IdempotencyKey == "")) {
		return nil, status.Error(codes.InvalidArgument, "invalid query")
	}
	var out *inventoryv1.ReservationOutcome
	err := s.transact(ctx, req.OrderId, func(tx *sql.Tx, r *terminalRecord, now time.Time) error {
		if r.outcome == nil {
			return status.Error(codes.NotFound, "reservation not found")
		}
		if req.QueryAction != "" {
			e := r.commands[req.QueryAction+":"+req.IdempotencyKey]
			if e == nil {
				return status.Error(codes.NotFound, "command not found")
			}
			r.outcome.Command = e
		}
		r.outcome.ObservedAt = timestamppb.New(now)
		out = r.outcome
		return nil
	})
	return out, err
}

// RunExpiry releases expired stock without fabricating an Order command proof.
func (s *TerminalServer) RunExpiry(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rows, err := s.DB.QueryContext(ctx, `SELECT order_id FROM inventory_terminal`)
			if err != nil {
				log.Print("terminal expiry scan failed")
				continue
			}
			orders := []string{}
			for rows.Next() {
				var id string
				if err = rows.Scan(&id); err != nil {
					break
				}
				orders = append(orders, id)
			}
			scanErr := rows.Err()
			rows.Close()
			if err != nil || scanErr != nil {
				log.Print("terminal expiry scan failed")
				continue
			}
			for _, id := range orders {
				if ctx.Err() != nil {
					return
				}
				err = s.transact(ctx, id, func(tx *sql.Tx, r *terminalRecord, now time.Time) error {
					if r.outcome == nil || r.outcome.State != "RESERVED" || now.Before(r.outcome.ExpiresAt.AsTime()) {
						return nil
					}
					if err := s.mutateStock(ctx, tx, r, false); err != nil {
						return err
					}
					r.outcome.State = "EXPIRED"
					r.outcome.ResourceVersion++
					r.outcome.ObservedAt = timestamppb.New(now)
					return saveTerminal(ctx, tx, r)
				})
				if err != nil {
					log.Print("terminal expiry transaction failed")
				}
			}
		}
	}
}
