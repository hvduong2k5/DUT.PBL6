package domain

import (
	"bytes"
	"encoding/json"
	"io"
	"time"
)

type Ready struct {
	Spec        string    `json:"specversion"`
	ID          string    `json:"id"`
	Source      string    `json:"source"`
	Type        string    `json:"type"`
	Subject     string    `json:"subject"`
	Time        time.Time `json:"time"`
	Content     string    `json:"datacontenttype"`
	Trace       string    `json:"traceparent,omitempty"`
	TraceState  string    `json:"tracestate,omitempty"`
	Correlation string    `json:"correlationid,omitempty"`
	Data        struct {
		Order        string `json:"aggregate_id"`
		Version      int64  `json:"version"`
		Ordinal      *int64 `json:"ordinal"`
		Status       string `json:"status"`
		Payment      string `json:"payment_status"`
		Stock        string `json:"stock_status"`
		Digest       string `json:"items_digest"`
		Warehouse    string `json:"warehouse_id"`
		Reservation  string `json:"reservation_id"`
		StockVersion int64  `json:"stock_resource_version"`
		Generation   int64  `json:"ready_generation"`
		Policy       string `json:"policy_version"`
	} `json:"data"`
}

func ParseReady(raw, key []byte) (out Ready, e error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if e = dec.Decode(&out); e != nil {
		return out, Fail("INVALID_READY_EVENT", 400)
	}
	var trailing any
	if dec.Decode(&trailing) != io.EOF {
		return out, Fail("INVALID_READY_EVENT", 400)
	}
	x := out.Data
	if out.Spec != "1.0" || !UUID(out.ID) || out.Source != "https://omama.vn/services/order-service" || out.Type != "vn.omama.order.fulfillment_ready.v3" || !UUID(x.Order) || out.Subject != "order:"+x.Order || string(key) != x.Order || out.Time.IsZero() || out.Content != "application/json" || x.Version < 1 || x.Ordinal == nil || *x.Ordinal < 0 || x.Generation < 1 || x.StockVersion < 1 || x.Policy == "" || !UUID(x.Reservation) || !hashPattern.MatchString(x.Digest) || x.Warehouse == "" || x.Stock != "COMMITTED" || !((x.Status == "PAID" && x.Payment == "CONFIRMED") || (x.Status == "CONFIRMED_COD" && x.Payment == "UNPAID")) {
		return out, Fail("INVALID_READY_EVENT", 400)
	}
	return
}
