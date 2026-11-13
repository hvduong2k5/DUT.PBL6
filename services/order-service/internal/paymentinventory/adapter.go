package paymentinventory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/omamx/order-service/internal/domain/core"
	inventory "github.com/omamx/order-service/internal/gen/inventory/v1"
	"google.golang.org/grpc/metadata"
	"time"
)

var ErrProof = errors.New("INVENTORY_FINALIZATION_PROOF_UNAVAILABLE")

type Adapter struct {
	Client inventory.InventoryServiceClient
	Token  string
	// Terminal is opt-in for the real TLS/Bearer provider. The legacy simulator
	// remains a separate profile; no error triggers a fallback between profiles.
	Terminal bool
}

func (a Adapter) Finalize(ctx context.Context, root core.Aggregate, key string) (core.ResourceProof, error) {
	if a.Terminal {
		r := root.Record().Reservation
		return a.terminalCommand(ctx, core.SnapshotInput{OrderID: root.ID(), WarehouseID: r.WarehouseID, Provenance: core.Provenance{ItemsDigest: r.ItemsDigest}}, r, key, "FinalizeReservation")
	}
	if a.Client == nil || len(a.Token) < 16 {
		return core.ResourceProof{}, ErrProof
	}
	r := root.Record().Reservation
	// Fixed command document, independent from worker epoch/retry counters.
	raw, _ := json.Marshal(struct{ Action, OrderID, ReservationID, Digest, Warehouse string }{"FINALIZE", root.ID(), r.ReservationID, string(r.ItemsDigest), r.WarehouseID})
	hash := sha256.Sum256(raw)
	requestHash := hex.EncodeToString(hash[:])
	reservation := r.ReservationID
	version := r.ResourceVersion
	request := &inventory.ReservationCommand{OrderId: root.ID(), IdempotencyKey: key, ContractVersion: "order-3.2", RequestHash: requestHash, ExpectedReservationId: &reservation, ItemsDigest: string(r.ItemsDigest), WarehouseId: r.WarehouseID, ExpectedVersion: &version, CorrelationId: root.ID(), QueryAction: "FINALIZE"}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-internal-token", a.Token)
	outcome, e := a.Client.GetReservationByOrder(ctx, request)
	if e != nil {
		return core.ResourceProof{}, e
	}
	if outcome.State == "RESERVED" {
		if outcome.OrderId != root.ID() || outcome.ReservationId != r.ReservationID || outcome.ItemsDigest != string(r.ItemsDigest) || outcome.WarehouseId != r.WarehouseID || outcome.ContractVersion != "order-3.2" {
			return core.ResourceProof{}, ErrProof
		}
		request.QueryAction = ""
		outcome, e = a.Client.FinalizeReservation(ctx, request)
		if e != nil {
			return core.ResourceProof{}, e
		}
	}
	command := outcome.GetCommand()
	if outcome.GetContractVersion() != "order-3.2" || outcome.GetOrderId() != root.ID() || outcome.GetReservationId() != r.ReservationID || outcome.GetItemsDigest() != string(r.ItemsDigest) || outcome.GetWarehouseId() != r.WarehouseID || outcome.GetResourceVersion() < r.ResourceVersion || outcome.ExpiresAt == nil || outcome.ExpiresAt.CheckValid() != nil || command == nil || command.DecisionAt == nil || command.DecisionAt.CheckValid() != nil || command.IdempotencyKey != key || command.RequestHash != requestHash || command.Action != "FINALIZE" || command.DecisionId == "" || (command.Result != "APPLIED" && command.Result != "ALREADY_APPLIED" && command.Result != "REJECTED") || command.ResultingState != outcome.State {
		return core.ResourceProof{}, ErrProof
	}
	if outcome.State != "COMMITTED" && outcome.State != "EXPIRED" && outcome.State != "RELEASED" {
		return core.ResourceProof{}, ErrProof
	}
	if outcome.State == "COMMITTED" && command.Result == "REJECTED" || command.DecisionAt.AsTime().After(time.Now().Add(time.Minute)) {
		return core.ResourceProof{}, ErrProof
	}
	return core.ResourceProof{OrderID: outcome.OrderId, ReservationID: outcome.ReservationId, ItemsDigest: core.ManifestHash(outcome.ItemsDigest), WarehouseID: outcome.WarehouseId, State: outcome.State, ResourceVersion: outcome.ResourceVersion, DecisionID: command.DecisionId, DecisionAt: command.DecisionAt.AsTime(), ExpiresAt: outcome.ExpiresAt.AsTime()}, nil
}
