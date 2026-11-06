package app

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omamx/order-service/internal/domain"
)

type orderReadScope struct {
	global     bool
	warehouses []string
}

func scopeForPermission(roles []identityRole, permission string) orderReadScope {
	scope := orderReadScope{warehouses: []string{}}
	for _, role := range roles {
		if !slices.Contains(role.Permissions, permission) {
			continue
		}
		if role.Scope == "GLOBAL" {
			return orderReadScope{global: true}
		}
		if role.Scope == "WAREHOUSE" && role.ScopeID != "" {
			scope.warehouses = append(scope.warehouses, role.ScopeID)
		}
	}
	slices.Sort(scope.warehouses)
	scope.warehouses = slices.Compact(scope.warehouses)
	return scope
}

type AdminOrderReader struct {
	id        string
	view      orderReadScope
	sensitive orderReadScope
}

func (s *Service) AuthenticateAdminOrderReader(ctx context.Context, bearer string) (AdminOrderReader, error) {
	identity := s.Identity
	if identity == nil || identity.identityURL == "" {
		return AdminOrderReader{}, domain.E("IDENTITY_UNAVAILABLE", 503)
	}
	claims, err := identity.verifiedClaims(ctx, bearer)
	if err != nil {
		return AdminOrderReader{}, err
	}
	profile, err := identity.profile(ctx, bearer, claims)
	if err != nil {
		return AdminOrderReader{}, err
	}
	if profile.Type != "EMPLOYEE" || profile.Status != "ACTIVE" {
		return AdminOrderReader{}, domain.E("STAFF_ACCESS_REQUIRED", 403)
	}
	view := scopeForPermission(profile.Roles, "ORDER_VIEW")
	if !view.global && len(view.warehouses) == 0 {
		return AdminOrderReader{}, domain.E("PERMISSION_DENIED", 403)
	}
	return AdminOrderReader{claims.UserID, view, scopeForPermission(profile.Roles, "ORDER_SENSITIVE_VIEW")}, nil
}

type adminOrderSummary struct {
	ID         string     `json:"order_id"`
	Code       string     `json:"order_code"`
	Status     string     `json:"status"`
	Payment    string     `json:"payment_status"`
	Method     string     `json:"payment_method"`
	Stock      string     `json:"stock_status"`
	Warehouse  string     `json:"warehouse_id"`
	CustomerID *string    `json:"customer_id"`
	Task       *string    `json:"task_id"`
	Shipment   *string    `json:"shipment_id"`
	Final      int64      `json:"final_amount"`
	ItemCount  int64      `json:"item_count"`
	Version    int64      `json:"version"`
	Hold       bool       `json:"operational_hold"`
	ActiveCase bool       `json:"has_active_case"`
	SLA        *time.Time `json:"sla_due_at"`
	Created    time.Time  `json:"created_at"`
	Updated    time.Time  `json:"updated_at"`
}

func adminSummary(o domain.Order, count int64) adminOrderSummary {
	return adminOrderSummary{
		ID: o.ID, Code: o.Code, Status: o.Status, Payment: o.Payment, Method: o.Method,
		Stock: o.Stock, Warehouse: o.Warehouse, CustomerID: o.CustomerID, Task: o.Task,
		Shipment: o.Shipment, Final: o.Final, ItemCount: count, Version: o.Version,
		Hold: o.Hold, ActiveCase: o.ActiveCase, SLA: o.SLA, Created: o.CreatedAt, Updated: o.UpdatedAt,
	}
}

const adminOrderWhere = ` FROM orders WHERE (cohort='sandbox-3.1' OR ($1 AND cohort='order-core-1'))
 AND ($2 OR warehouse_id=ANY($3::text[]))`

const adminOrderCount = `,(SELECT COALESCE(SUM(quantity),0) FROM order_items WHERE order_id=orders.id)`

func (s *Service) AdminOrderPage(ctx context.Context, actor AdminOrderReader, status, cursor string, limit int, queue bool) (any, error) {
	// Bind the cursor to both the current effective scope and the selected filter.
	cursorScope := "staff-order:" + domain.Hash([]any{actor.id, actor.view.global, actor.view.warehouses, queue})
	var boundary time.Time
	var boundaryID string
	if cursor != "" {
		if err := s.decodeCursor(cursor, cursorScope, status, &boundary, &boundaryID); err != nil {
			return nil, err
		}
	}
	sortAt, direction, comparison := "created_at", " DESC", "<"
	where := adminOrderWhere
	if queue {
		sortAt = "COALESCE(sla_due_at,'9999-12-31T00:00:00Z'::timestamptz)"
		direction, comparison = " ASC", ">"
		where += readyOrderQueue
	}
	query := "SELECT " + orderColumns + adminOrderCount + "," + sortAt + where +
		" AND ($4='' OR status=$4) AND ($5='' OR (" + sortAt + ",id)" + comparison + "($6,NULLIF($5,'')::uuid))" +
		" ORDER BY " + sortAt + direction + ",id" + direction + " LIMIT $7"
	rows, err := s.DB.Query(ctx, query,
		s.CoreCheckout != nil, actor.view.global, actor.view.warehouses, status, boundaryID, boundary, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []adminOrderSummary{}
	boundaries := []time.Time{}
	for rows.Next() {
		var count int64
		var at time.Time
		o, err := scanOrder(rows, &count, &at)
		if err != nil {
			return nil, err
		}
		items = append(items, adminSummary(o, count))
		boundaries = append(boundaries, at)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var next *string
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		value := s.Cursor(domain.Order{ID: last.ID, CreatedAt: boundaries[limit-1]}, cursorScope, status)
		next = &value
	}
	return map[string]any{"items": items, "next_cursor": next, "calculated_at": time.Now().UTC()}, nil
}

type adminOrderLine struct {
	SKU       string `json:"sku_code"`
	Name      string `json:"product_name"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
	Total     int64  `json:"line_total"`
}

type adminOrderHistory struct {
	ID   string    `json:"event_id"`
	From *string   `json:"from_status"`
	To   string    `json:"to_status"`
	At   time.Time `json:"occurred_at"`
}

func (s *Service) AdminOrderDetail(ctx context.Context, actor AdminOrderReader, id string) (any, error) {
	if parsed, err := uuid.Parse(id); err != nil || parsed.String() != id {
		return nil, domain.E("ORDER_NOT_FOUND", 404)
	}
	var count, discount int64
	o, err := scanOrder(s.DB.QueryRow(ctx, "SELECT "+orderColumns+adminOrderCount+
		",merchandise_discount+shipping_discount"+adminOrderWhere+" AND id=$4",
		s.CoreCheckout != nil, actor.view.global, actor.view.warehouses, id), &count, &discount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.E("ORDER_NOT_FOUND", 404)
	}
	if err != nil {
		return nil, err
	}
	lines, err := s.adminOrderLines(ctx, id)
	if err != nil {
		return nil, err
	}
	history, err := s.adminOrderHistory(ctx, id, o.Version)
	if err != nil {
		return nil, err
	}
	var recipient map[string]string
	if actor.sensitive.global || slices.Contains(actor.sensitive.warehouses, o.Warehouse) {
		recipient, err = s.adminOrderRecipient(ctx, o)
		if err != nil {
			return nil, err
		}
	}
	return map[string]any{
		"order": adminSummary(o, count), "lines": lines, "history": history,
		"subtotal": o.Subtotal, "shipping_fee": o.Shipping, "discount": discount,
		"recipient": recipient, "calculated_at": time.Now().UTC(),
	}, nil
}

func (s *Service) adminOrderLines(ctx context.Context, id string) ([]adminOrderLine, error) {
	rows, err := s.DB.Query(ctx, "SELECT sku,product_name,quantity,unit_price,line_total FROM order_items WHERE order_id=$1 ORDER BY sku", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lines := []adminOrderLine{}
	for rows.Next() {
		var line adminOrderLine
		if err := rows.Scan(&line.SKU, &line.Name, &line.Quantity, &line.UnitPrice, &line.Total); err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	return lines, rows.Err()
}

func (s *Service) adminOrderHistory(ctx context.Context, id string, version int64) ([]adminOrderHistory, error) {
	rows, err := s.DB.Query(ctx, `SELECT id,from_status,to_status,occurred_at FROM
 (SELECT id,from_status,to_status,occurred_at,version FROM order_history
  WHERE order_id=$1 AND version<=$2 AND from_status IS DISTINCT FROM to_status
  ORDER BY version DESC LIMIT 50) history ORDER BY version`, id, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	history := []adminOrderHistory{}
	for rows.Next() {
		var entry adminOrderHistory
		if err := rows.Scan(&entry.ID, &entry.From, &entry.To, &entry.At); err != nil {
			return nil, err
		}
		history = append(history, entry)
	}
	return history, rows.Err()
}

func (s *Service) adminOrderRecipient(ctx context.Context, o domain.Order) (map[string]string, error) {
	if o.Cohort == "order-core-1" {
		order, err := s.Payment.Store.LoadOrder(ctx, o.ID, o.Scope)
		if err != nil {
			return nil, err
		}
		address := order.Snapshot().Data().Address
		return map[string]string{
			"name": address.RecipientName, "phone": address.PhoneE164,
			"address": strings.Join([]string{address.Street, address.WardName, address.ProvinceName}, ", "),
		}, nil
	}
	var cipher []byte
	if err := s.DB.QueryRow(ctx, "SELECT snapshot_cipher FROM orders WHERE id=$1", o.ID).Scan(&cipher); err != nil {
		return nil, err
	}
	var quote domain.Quote
	if err := s.Open(cipher, &quote); err != nil {
		return nil, err
	}
	return map[string]string{
		"name": quote.Address.Name, "phone": quote.Address.Phone,
		"address": strings.Join([]string{quote.Address.Street, quote.Address.Ward, quote.Address.Province}, ", "),
	}, nil
}
