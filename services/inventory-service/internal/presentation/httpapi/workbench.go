package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

const WorkbenchPath = "/api/v1/admin/inventory/workbench"

type WorkbenchHandler struct{ ReceiptHandler }
type workbenchBalance struct {
	OnHand      int64 `json:"onHand"`
	Reserved    int64 `json:"reserved"`
	NonSellable int64 `json:"nonSellable"`
	Available   int64 `json:"available"`
}
type workbenchBatch struct {
	ID           string           `json:"batchId"`
	Code         string           `json:"batchCode"`
	Manufactured string           `json:"manufacturingDate"`
	Expires      string           `json:"expiresAt"`
	State        string           `json:"expiryState"`
	Label        string           `json:"expiryLabel"`
	Balance      workbenchBalance `json:"balance"`
}
type workbenchSKU struct {
	ID       string           `json:"skuId"`
	Code     string           `json:"skuCode"`
	Label    string           `json:"skuLabel"`
	Name     string           `json:"productName"`
	Balance  workbenchBalance `json:"balance"`
	Revision int64            `json:"revision"`
	Batches  []workbenchBatch `json:"batches"`
}

func (h WorkbenchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	id := uuid.NewString()
	w.Header().Set("X-Request-Id", id)
	fail := func(status int, code string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": http.StatusText(status), "requestId": id})
	}
	header := r.Header.Get("Authorization")
	if h.Key == nil || !strings.HasPrefix(header, "Bearer ") || len(header) > 8192 {
		fail(401, "UNAUTHENTICATED")
		return
	}
	claims := &struct {
		jwt.RegisteredClaims
		User   string   `json:"user_id"`
		Type   string   `json:"principal_type"`
		Grants []string `json:"permission_grants"`
	}{}
	token, err := jwt.ParseWithClaims(strings.TrimPrefix(header, "Bearer "), claims, func(*jwt.Token) (any, error) { return h.Key, nil }, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(h.Issuer), jwt.WithAudience(h.Audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(30*time.Second))
	if err != nil || !token.Valid || claims.Subject == "" {
		fail(401, "UNAUTHENTICATED")
		return
	}
	actor, err := uuid.Parse(claims.User)
	if err != nil || actor == uuid.Nil || claims.Type != "EMPLOYEE" {
		fail(403, "PERMISSION_DENIED")
		return
	}
	global := true
	intersection := map[string]bool{}
	for index, permission := range []string{"INVENTORY_VIEW", "WAREHOUSE_WORKBENCH_VIEW"} {
		all := false
		warehouses := map[string]bool{}
		for _, grant := range claims.Grants {
			if grant == permission+"#GLOBAL" {
				all = true
			}
			if warehouse, ok := strings.CutPrefix(grant, permission+"#WAREHOUSE#"); ok {
				if parsed, e := uuid.Parse(warehouse); e == nil && parsed != uuid.Nil && parsed.String() == warehouse {
					warehouses[warehouse] = true
				}
			}
		}
		if !all && len(warehouses) == 0 {
			fail(403, "PERMISSION_DENIED")
			return
		}
		if index == 0 {
			global = all
			intersection = warehouses
			continue
		}
		if global {
			global = all
			intersection = warehouses
		} else if !all {
			for warehouse := range intersection {
				if !warehouses[warehouse] {
					delete(intersection, warehouse)
				}
			}
		}
	}
	if !global && len(intersection) == 0 {
		fail(403, "PERMISSION_DENIED")
		return
	}
	warehouses := []string{}
	for warehouse := range intersection {
		warehouses = append(warehouses, warehouse)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	result, err := h.readWorkbench(ctx, global, warehouses)
	if err != nil {
		fail(503, "INVENTORY_UNAVAILABLE")
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}

func balance(onHand, reserved, available int64) workbenchBalance {
	nonSellable := onHand - reserved - available
	if nonSellable < 0 {
		nonSellable = 0
	}
	return workbenchBalance{OnHand: onHand, Reserved: reserved, Available: available, NonSellable: nonSellable}
}

func (h WorkbenchHandler) readWorkbench(ctx context.Context, global bool, warehouses []string) (map[string]any, error) {
	tx, err := h.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT sku,physical_qty,reserved_qty,inventory_sellable_quantity(sku,now()),inventory_version
 FROM inventory_items WHERE ($1 OR warehouse_id::text=ANY($2)) ORDER BY sku LIMIT 101`, global, pq.Array(warehouses))
	if err != nil {
		return nil, err
	}
	skus := []*workbenchSKU{}
	bySKU := map[string]*workbenchSKU{}
	codes := []string{}
	for rows.Next() {
		sku := &workbenchSKU{Batches: []workbenchBatch{}}
		var onHand, reserved, available int64
		if err = rows.Scan(&sku.ID, &onHand, &reserved, &available, &sku.Revision); err != nil {
			rows.Close()
			return nil, err
		}
		sku.Code, sku.Label, sku.Name = sku.ID, sku.ID, sku.ID
		sku.Balance = balance(onHand, reserved, available)
		skus = append(skus, sku)
		bySKU[sku.ID] = sku
		codes = append(codes, sku.ID)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if len(skus) > 100 {
		return nil, receiptError{503, "WORKBENCH_LIMIT"}
	}
	rows, err = tx.QueryContext(ctx, `SELECT id::text,sku,batch_code,mfg_date,exp_date,physical_qty,reserved_qty,
 CASE WHEN status IN ('ACTIVE','NEAR_EXPIRY') AND exp_date>CURRENT_DATE THEN physical_qty-reserved_qty ELSE 0 END,
 CASE WHEN exp_date<=CURRENT_DATE THEN 'EXPIRED' WHEN exp_date<=CURRENT_DATE+45 THEN 'EXPIRING' ELSE 'SAFE' END
 FROM batches WHERE sku=ANY($1) AND mfg_date IS NOT NULL AND exp_date IS NOT NULL
 AND status IN ('ACTIVE','NEAR_EXPIRY','EXPIRED') ORDER BY sku,exp_date,id LIMIT 1001`, pq.Array(codes))
	if err != nil {
		return nil, err
	}
	count := 0
	zone := time.FixedZone("Asia/Ho_Chi_Minh", 7*3600)
	for rows.Next() {
		var item workbenchBatch
		var sku string
		var manufactured, expires time.Time
		var onHand, reserved, available int64
		if err = rows.Scan(&item.ID, &sku, &item.Code, &manufactured, &expires, &onHand, &reserved, &available, &item.State); err != nil {
			rows.Close()
			return nil, err
		}
		item.Manufactured = time.Date(manufactured.Year(), manufactured.Month(), manufactured.Day(), 0, 0, 0, 0, zone).Format(time.RFC3339)
		item.Expires = time.Date(expires.Year(), expires.Month(), expires.Day(), 0, 0, 0, 0, zone).Format(time.RFC3339)
		item.Label = map[string]string{"SAFE": "Còn hạn", "EXPIRING": "Sắp hết hạn", "EXPIRED": "Hết hạn"}[item.State]
		item.Balance = balance(onHand, reserved, available)
		bySKU[sku].Batches = append(bySKU[sku].Batches, item)
		count++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if count > 1000 {
		return nil, receiptError{503, "WORKBENCH_LIMIT"}
	}
	rows, err = tx.QueryContext(ctx, `SELECT a.id::text,a.adjustment_type,a.quantity_delta,a.created_at,a.adjusted_by::text,
 COALESCE(a.reason,''),a.sku,COALESCE(b.batch_code,'') FROM stock_adjustments a LEFT JOIN batches b ON b.id=a.batch_id
 WHERE a.sku=ANY($1) ORDER BY a.created_at DESC,a.id DESC LIMIT 100`, pq.Array(codes))
	if err != nil {
		return nil, err
	}
	movements := []map[string]any{}
	for rows.Next() {
		var movement, kind, actor, reference, sku, batch string
		var quantity int64
		var occurred time.Time
		if err = rows.Scan(&movement, &kind, &quantity, &occurred, &actor, &reference, &sku, &batch); err != nil {
			rows.Close()
			return nil, err
		}
		movements = append(movements, map[string]any{"movementId": movement, "type": kind, "typeLabel": kind, "quantity": quantity, "occurredAt": occurred.UTC().Format(time.RFC3339), "actorLabel": actor, "reference": reference, "skuCode": sku, "batchCode": batch})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]any{"skus": skus, "recentMovements": movements, "pendingApprovalCount": 0, "calculatedAt": time.Now().UTC().Format(time.RFC3339), "policyNotice": "Số dư theo kho được giao; tên hiển thị bằng mã SKU. Chỉ liệt kê lô có ngày rõ ràng, không cách ly để chọn nhập. Phiếu nhập được áp dụng trực tiếp."}, nil
}
