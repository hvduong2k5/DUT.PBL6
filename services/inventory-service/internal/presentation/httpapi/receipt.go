package httpapi

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

const ReceiptPath = "/api/v1/admin/inventory/workbench/receipts"

var receiptKey = regexp.MustCompile("^[!-~]{16,128}$")

type ReceiptHandler struct {
	DB               *sql.DB
	Key              *rsa.PublicKey
	Issuer, Audience string
}

type receiptRequest struct {
	SKU          string `json:"skuId"`
	BatchMode    string `json:"batchMode"`
	BatchID      string `json:"batchId"`
	BatchCode    string `json:"batchCode"`
	SupplierID   string `json:"supplierId"`
	Manufactured string `json:"manufacturingDate"`
	Expires      string `json:"expiresAt"`
	Reference    string `json:"sourceReference"`
	Reason       string `json:"reasonCode"`
	Quantity     int32  `json:"quantity"`
	Revision     *int64 `json:"expectedRevision"`
	Key          string `json:"idempotencyKey"`
}

type receiptResult struct {
	ID       string    `json:"operationId"`
	Status   string    `json:"status"`
	Label    string    `json:"statusLabel"`
	Movement string    `json:"movementId"`
	Approval *string   `json:"approvalRequestId"`
	Occurred time.Time `json:"occurredAt"`
	Revision int64     `json:"revision"`
	Message  string    `json:"message"`
}

type receiptError struct {
	status int
	code   string
}

func (e receiptError) Error() string { return e.code }

func (h ReceiptHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	requestID := uuid.NewString()
	w.Header().Set("X-Request-Id", requestID)
	fail := func(status int, code string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "message": http.StatusText(status), "requestId": requestID})
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
	token, err := jwt.ParseWithClaims(strings.TrimPrefix(header, "Bearer "), claims,
		func(*jwt.Token) (any, error) { return h.Key, nil }, jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(h.Issuer), jwt.WithAudience(h.Audience), jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(), jwt.WithLeeway(30*time.Second))
	if err != nil || !token.Valid || claims.Subject == "" {
		fail(401, "UNAUTHENTICATED")
		return
	}
	actor, err := uuid.Parse(claims.User)
	if err != nil || actor == uuid.Nil || claims.Type != "EMPLOYEE" {
		fail(403, "PERMISSION_DENIED")
		return
	}
	global := false
	warehouses := map[string]bool{}
	for _, grant := range claims.Grants {
		if grant == "INVENTORY_RECEIPT_CREATE#GLOBAL" {
			global = true
		}
		if scope, ok := strings.CutPrefix(grant, "INVENTORY_RECEIPT_CREATE#WAREHOUSE#"); ok {
			warehouses[scope] = true
		}
	}
	if !global && len(warehouses) == 0 {
		fail(403, "PERMISSION_DENIED")
		return
	}
	var req receiptRequest
	key := r.Header.Get("Idempotency-Key")
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&req) != nil || decoder.Decode(new(any)) != io.EOF || !receiptKey.MatchString(key) ||
		(req.Key != "" && req.Key != key) || !req.valid() {
		fail(400, "INVALID_ARGUMENT")
		return
	}
	if req.BatchMode == "" {
		req.Reference = "legacy:" + actor.String() + ":" + key
	}
	req.Key = ""
	result, err := h.receive(r.Context(), actor, global, warehouses, key, req)
	if err != nil {
		var problem receiptError
		var dbError *pq.Error
		switch {
		case errors.As(err, &problem):
			fail(problem.status, problem.code)
		case errors.As(err, &dbError) && dbError.Code == "23505":
			fail(409, "RECEIPT_CONFLICT")
		case errors.As(err, &dbError) && dbError.Code == "22003":
			fail(422, "QUANTITY_OUT_OF_RANGE")
		default:
			slog.Error("Inventory receipt transaction failed", "request_id", requestID, "error", err)
			fail(503, "INVENTORY_UNAVAILABLE")
		}
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}

func (r receiptRequest) valid() bool {
	if strings.TrimSpace(r.SKU) != r.SKU || r.SKU == "" || len(r.SKU) > 50 ||
		r.Quantity <= 0 || r.Revision == nil || *r.Revision < 0 ||
		len(r.Reference) > 160 || len(r.Reason) > 80 {
		return false
	}
	if r.BatchMode != "" && strings.TrimSpace(r.Reference) == "" {
		return false
	}
	switch r.BatchMode {
	case "":
		return r.BatchCode != "" && len(r.BatchCode) <= 100 && r.Reason != "" && r.Key != "" &&
			r.BatchID == "" && r.SupplierID == "" && r.Manufactured == "" && r.Expires == ""
	case "EXISTING":
		id, err := uuid.Parse(r.BatchID)
		return err == nil && id != uuid.Nil && r.BatchCode == "" && r.SupplierID == "" && r.Manufactured == "" && r.Expires == ""
	case "NEW":
		id, supplierErr := uuid.Parse(r.SupplierID)
		mfg, mfgErr := time.Parse(time.DateOnly, r.Manufactured)
		exp, expErr := time.Parse(time.DateOnly, r.Expires)
		return r.BatchID == "" && supplierErr == nil && id != uuid.Nil && mfgErr == nil && expErr == nil &&
			strings.TrimSpace(r.BatchCode) == r.BatchCode && r.BatchCode != "" && len(r.BatchCode) <= 100 && exp.After(mfg)
	default:
		return false
	}
}

func (h ReceiptHandler) receive(ctx context.Context, actor uuid.UUID, global bool, warehouses map[string]bool, key string, req receiptRequest) (result receiptResult, err error) {
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	var warehouse string
	var revision int64
	if err = tx.QueryRowContext(ctx, `SELECT warehouse_id::text,inventory_version FROM inventory_items
		WHERE sku=$1 FOR UPDATE`, req.SKU).Scan(&warehouse, &revision); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result, receiptError{422, "SKU_NOT_FOUND"}
		}
		return result, err
	}
	if !global && !warehouses[warehouse] {
		return result, receiptError{403, "PERMISSION_DENIED"}
	}

	raw, err := json.Marshal(req)
	if err != nil {
		return result, err
	}
	var matches bool
	err = tx.QueryRowContext(ctx, `SELECT request=$3::jsonb,response FROM stock_receipts WHERE actor_id=$1 AND idem_key=$2`,
		actor, key, string(raw)).Scan(&matches, &raw)
	if err == nil {
		if !matches {
			return result, receiptError{409, "IDEMPOTENCY_CONFLICT"}
		}
		return result, json.Unmarshal(raw, &result)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	if revision != *req.Revision {
		return result, receiptError{409, "REVISION_CONFLICT"}
	}

	var batchID string
	if req.BatchMode == "NEW" {
		err = tx.QueryRowContext(ctx, `INSERT INTO batches(batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,status)
			SELECT $1::varchar,$2::varchar,$3::uuid,$4::date,$5::date,$6::integer,CASE WHEN $5::date <= CURRENT_DATE+45 THEN 'NEAR_EXPIRY' ELSE 'ACTIVE' END
			WHERE $4::date<=CURRENT_DATE AND $5::date>CURRENT_DATE
			  AND NOT EXISTS(SELECT 1 FROM batches WHERE sku=$2 AND batch_code=$1)
			RETURNING id::text`, req.BatchCode, req.SKU, req.SupplierID, req.Manufactured, req.Expires, req.Quantity).Scan(&batchID)
	} else {
		if req.BatchMode == "" {
			var candidates int
			err = tx.QueryRowContext(ctx, `SELECT count(*),COALESCE(min(id::text),'') FROM batches
				WHERE sku=$1 AND batch_code=$2`, req.SKU, req.BatchCode).Scan(&candidates, &req.BatchID)
			if err != nil {
				return result, err
			}
			if candidates != 1 {
				return result, receiptError{422, "INVALID_BATCH"}
			}
		}
		err = tx.QueryRowContext(ctx, `UPDATE batches SET physical_qty=physical_qty+$3,version=version+1
			WHERE id=$1 AND sku=$2 AND exp_date>CURRENT_DATE AND status IN ('ACTIVE','NEAR_EXPIRY')
			RETURNING id::text`, req.BatchID, req.SKU, req.Quantity).Scan(&batchID)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return result, receiptError{422, "INVALID_BATCH"}
	}
	if err != nil {
		return result, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE inventory_items SET physical_qty=physical_qty+$2,updated_at=clock_timestamp() WHERE sku=$1`, req.SKU, req.Quantity); err != nil {
		return result, err
	}
	result.ID = uuid.NewString()
	if err = tx.QueryRowContext(ctx, `INSERT INTO stock_adjustments(adjustment_code,sku,batch_id,adjustment_type,quantity_delta,reason,adjusted_by)
		VALUES ($1,$2,$3,'RECEIPT',$4,$5,$6) RETURNING id::text,created_at`,
		result.ID, req.SKU, batchID, req.Quantity, req.Reason, actor).Scan(&result.Movement, &result.Occurred); err != nil {
		return result, err
	}
	if _, err = tx.ExecContext(ctx, `SET CONSTRAINTS inventory_item_snapshot, inventory_batch_snapshot IMMEDIATE`); err != nil {
		return result, err
	}
	if err = tx.QueryRowContext(ctx, `SELECT inventory_version FROM inventory_items WHERE sku=$1`, req.SKU).Scan(&result.Revision); err != nil {
		return result, err
	}
	result.Status, result.Label, result.Message = "APPLIED", "Đã áp dụng", "Đã ghi nhận nhập kho."
	response, err := json.Marshal(result)
	if err != nil {
		return result, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO stock_receipts(actor_id,idem_key,sku,source_reference,request,response)
		VALUES($1,$2,$3,$4,$5,$6)`, actor, key, req.SKU, req.Reference, string(raw), string(response)); err != nil {
		return result, err
	}
	return result, tx.Commit()
}
