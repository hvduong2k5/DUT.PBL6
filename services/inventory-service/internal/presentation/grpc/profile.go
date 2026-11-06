package grpc

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	inventoryv1 "dut-pbl6/inventory-service/pkg/proto/inventory/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// canonicalRequest implements the order-3.2 JSON hash profile, including optional presence.
// Keep this separate from protojson: protobuf int64 JSON strings are not this protocol.
func canonicalRequest(m protoreflect.Message) map[string]any {
	out := map[string]any{}
	m.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		name := string(f.Name())
		if name == "request_hash" || name == "correlation_id" || name == "idempotency_key" {
			return true
		}
		convert := func(v protoreflect.Value) any { return v.Interface() }
		if f.Kind() == protoreflect.MessageKind {
			convert = func(v protoreflect.Value) any { return canonicalRequest(v.Message()) }
		}
		if f.IsList() {
			a := []any{}
			for i := 0; i < v.List().Len(); i++ {
				a = append(a, convert(v.List().Get(i)))
			}
			out[name] = a
		} else {
			out[name] = convert(v)
		}
		return true
	})
	return out
}

func requestHash(m protoreflect.Message) string {
	b, _ := json.Marshal(canonicalRequest(m))
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func normalizeReserve(r *inventoryv1.ReserveStockRequest) error {
	if r == nil || r.ContractVersion != "order-3.2" || r.OrderId == "" || r.TtlMinutes <= 0 || r.TtlMinutes > 15 || r.GetPreferredWarehouseId() == "" {
		return fmt.Errorf("order, contract, TTL 1..15 and warehouse required")
	}
	quantities := map[string]int64{}
	for _, item := range r.Items {
		if item == nil || item.SkuCode == "" || item.Quantity <= 0 {
			return fmt.Errorf("invalid item")
		}
		for _, b := range []byte(item.SkuCode) {
			if b < 33 || b > 126 {
				return fmt.Errorf("SKU must be printable ASCII")
			}
		}
		quantities[item.SkuCode] += int64(item.Quantity)
		if quantities[item.SkuCode] > math.MaxInt32 {
			return fmt.Errorf("quantity overflow")
		}
	}
	if len(quantities) == 0 {
		return fmt.Errorf("items required")
	}
	skus := []string{}
	for sku := range quantities {
		skus = append(skus, sku)
	}
	sort.Strings(skus)
	r.Items = nil
	canonical := []map[string]any{}
	for _, sku := range skus {
		r.Items = append(r.Items, &inventoryv1.ReserveItem{SkuCode: sku, Quantity: int32(quantities[sku])})
		canonical = append(canonical, map[string]any{"quantity": quantities[sku], "sku_code": sku})
	}
	b, _ := json.Marshal(canonical)
	digest := sha256.Sum256(b)
	if r.ItemsDigest != hex.EncodeToString(digest[:]) {
		return fmt.Errorf("items_digest mismatch")
	}
	return nil
}

func validKey(key string) bool {
	if len(key) < 16 || len(key) > 128 {
		return false
	}
	for _, b := range []byte(key) {
		if b < 33 || b > 126 {
			return false
		}
	}
	return true
}
