package app

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/domain/core"
	packing "github.com/omamx/order-service/internal/gen/packing/v1"
	"github.com/omamx/order-service/internal/shippingprovider"
	"github.com/omamx/order-service/internal/traceabilityprovider"
	"google.golang.org/protobuf/proto"
)

type OrderBatchTrace struct {
	Line     string  `json:"order_line_id"`
	SKU      string  `json:"sku_code"`
	Batch    string  `json:"batch_id"`
	Code     string  `json:"batch_code"`
	Quantity int32   `json:"packed_quantity"`
	Status   string  `json:"status"`
	URL      *string `json:"qr_url"`
}
type OrderTraceability struct {
	Order string            `json:"order_id"`
	State string            `json:"state"`
	Items []OrderBatchTrace `json:"items"`
	Total int               `json:"total"`
}

func (s *Service) EnableTraceabilityFromEnv() error {
	if os.Getenv("ORDER_TRACEABILITY_ENABLED") != "true" {
		return nil
	}
	token, err := shippingprovider.TokenFile(os.Getenv("ORDER_TRACEABILITY_TOKEN_FILE"))
	if err != nil {
		return err
	}
	adapter, conn, err := traceabilityprovider.Dial(os.Getenv("ORDER_TRACEABILITY_GRPC"), token, os.Getenv("ORDER_TRACEABILITY_CA_FILE"), os.Getenv("ORDER_TRACEABILITY_SERVER_NAME"), os.Getenv("ORDER_TRACEABILITY_PUBLIC_ORIGIN"))
	if err != nil {
		return err
	}
	s.Traceability = adapter
	prior := s.PaymentClose
	s.PaymentClose = func() error {
		if prior != nil {
			_ = prior()
		}
		return conn.Close()
	}
	return nil
}

// CustomerTraceability is a read-only projection. It never discovers live
// allocations or writes order/payment/shipping state, even when peers fail.
func (s *Service) CustomerTraceability(ctx context.Context, id string, actor domain.Principal) (OrderTraceability, error) {
	out := OrderTraceability{Order: id, State: "UNAVAILABLE", Items: []OrderBatchTrace{}}
	if actor.PrincipalKind() != domain.Customer && actor.PrincipalKind() != domain.Guest {
		return out, domain.E("CUSTOMER_REQUIRED", 403)
	}
	if !canonicalID(id) {
		return out, domain.E("ORDER_NOT_FOUND", 404)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var scope string
	err := s.DB.QueryRow(ctx, `SELECT scope FROM orders WHERE id=$1`, id).Scan(&scope)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, domain.E("ORDER_NOT_FOUND", 404)
	}
	if err != nil {
		return out, err
	}
	if actor.Scope() != scope {
		return out, domain.E("PERMISSION_DENIED", 403)
	}
	order, err := s.GetOrder(ctx, id, actor, true)
	if err != nil {
		return out, err
	}
	var packageID, task string
	var version int64
	err = s.DB.QueryRow(ctx, `SELECT package_id::text,task_id::text,sealed_version FROM fulfillment_shipping_handoffs WHERE order_id=$1`, id).Scan(&packageID, &task, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		if strings.HasPrefix(order.Status, "CANCELLED") {
			out.State = "NOT_APPLICABLE"
		} else if order.Seal == nil && order.Task == nil {
			out.State = "PACKING_PENDING"
		} else if order.Seal == nil && order.Status == "PROCESSING" {
			out.State = "PACKING_PENDING"
		}
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if order.CoreSnapshot == nil || s.Fulfillment == nil || s.Traceability == nil {
		return out, nil
	}
	proof, err := s.Fulfillment.Package(ctx, task, packageID, version)
	if err != nil || proof.OrderId != id || proof.ReadyGeneration != order.Generation || order.Task == nil || *order.Task != task || order.Seal == nil || *order.Seal != proof.SealCode || proof.ItemsDigest != string(order.CoreSnapshot.Provenance.ItemsDigest) || proof.WarehouseId != order.CoreSnapshot.WarehouseID || proto.Size(proof) > 512<<10 {
		return out, nil
	}
	rows, ok := sealedTraceRows(proof, order.CoreSnapshot.Items)
	if !ok {
		return out, nil
	}
	bindings := make([]traceabilityprovider.Binding, len(rows))
	for i, row := range rows {
		bindings[i] = traceabilityprovider.Binding{Batch: row.Batch, SKU: row.SKU}
	}
	links := s.Traceability.Resolve(ctx, bindings)
	for i := range rows {
		rows[i].Status, rows[i].URL = links[i].Status, links[i].URL
	}
	out.State, out.Items, out.Total = "READY", rows, len(rows)
	encoded, err := json.Marshal(out)
	if err != nil || len(encoded) > 512<<10 {
		return OrderTraceability{Order: id, State: "UNAVAILABLE", Items: []OrderBatchTrace{}}, nil
	}
	return out, nil
}

func canonicalID(raw string) bool {
	parsed, err := uuid.Parse(raw)
	return err == nil && parsed != uuid.Nil && parsed.String() == raw
}

// Validate the additive extension independently of the base Shipping proof.
// Every frozen line must be represented exactly, including split batches.
func sealedTraceRows(proof *packing.PackageRevision, items []core.SnapshotItem) ([]OrderBatchTrace, bool) {
	if proof == nil || len(proof.PackedLines) == 0 || len(proof.PackedLines) > 1000 || len(items) == 0 || len(items) > 100 {
		return nil, false
	}
	lines := map[string]core.SnapshotItem{}
	skus := map[string]bool{}
	positions := map[string]int{}
	for i, item := range items {
		if !canonicalID(item.LineID) || item.SKU == "" || len(item.SKU) > 100 || item.Quantity < 1 || item.Quantity > math.MaxInt32 || skus[item.SKU] {
			return nil, false
		}
		if _, exists := lines[item.LineID]; exists {
			return nil, false
		}
		lines[item.LineID], skus[item.SKU], positions[item.LineID] = item, true, i
	}
	rows := make([]OrderBatchTrace, 0, len(proof.PackedLines))
	totals := map[string]int64{}
	seen := map[string]bool{}
	codes := map[string]string{}
	for _, packed := range proof.PackedLines {
		if packed == nil {
			return nil, false
		}
		item, exists := lines[packed.OrderLineId]
		key := packed.OrderLineId + ":" + packed.BatchId
		if !exists || item.SKU != packed.SkuCode || !canonicalID(packed.BatchId) || packed.BatchCode == "" || len(packed.BatchCode) > 100 || strings.TrimSpace(packed.BatchCode) != packed.BatchCode || packed.PackedQuantity < 1 || seen[key] {
			return nil, false
		}
		if code, exists := codes[packed.BatchId]; exists && code != packed.BatchCode {
			return nil, false
		}
		codes[packed.BatchId], seen[key] = packed.BatchCode, true
		totals[item.LineID] += int64(packed.PackedQuantity)
		if totals[item.LineID] > item.Quantity {
			return nil, false
		}
		rows = append(rows, OrderBatchTrace{Line: item.LineID, SKU: item.SKU, Batch: packed.BatchId, Code: packed.BatchCode, Quantity: packed.PackedQuantity, Status: "UNAVAILABLE"})
	}
	for id, item := range lines {
		if totals[id] != item.Quantity {
			return nil, false
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Line != rows[j].Line {
			return positions[rows[i].Line] < positions[rows[j].Line]
		}
		return rows[i].Batch < rows[j].Batch
	})
	return rows, true
}
