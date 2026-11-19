package domain

import "time"

type Command struct {
	Caller, Action, OrderID, Key, Hash string
	Generation                         int64
}

func (c Command) Validate() error {
	if !UUID(c.OrderID) || !Key(c.Key) || !hashPattern.MatchString(c.Hash) || c.Generation < 1 {
		return Fail("INVALID_COMMAND", 400)
	}
	return nil
}

type Evidence struct {
	Key    string    `json:"idempotency_key"`
	Action string    `json:"action"`
	Hash   string    `json:"request_hash"`
	Result string    `json:"result"`
	Reason string    `json:"reason_code"`
	ID     string    `json:"decision_id"`
	At     time.Time `json:"decision_at"`
	State  string    `json:"resulting_state"`
}
type DecisionReference struct {
	Service    string `json:"service"`
	ID         string `json:"decision_id"`
	OrderID    string `json:"order_id"`
	Generation int64  `json:"generation"`
	Version    int64  `json:"resource_version"`
}
type Decision struct {
	OrderID    string             `json:"order_id"`
	Generation int64              `json:"generation"`
	State      string             `json:"state"`
	TaskID     *string            `json:"task_id"`
	Version    int64              `json:"resource_version"`
	Observed   time.Time          `json:"observed_at"`
	Command    *Evidence          `json:"command,omitempty"`
	Digest     string             `json:"items_digest"`
	Warehouse  string             `json:"warehouse_id"`
	Contract   string             `json:"contract_version"`
	Reference  *DecisionReference `json:"decision,omitempty"`
}
type Permit struct {
	ID, OrderID, Digest, Warehouse, Key, Audience, Policy string
	Generation, OrderVersion                              int64
	Expires                                               time.Time
	Allowed                                               bool
}
type AdmissionInput struct {
	Command    Command
	Permit     Permit
	Inputs     Inputs
	Stock      Stock
	Integrated bool
}
type Event struct {
	ID      string    `json:"id"`
	Spec    string    `json:"specversion"`
	Source  string    `json:"source"`
	Type    string    `json:"type"`
	Subject string    `json:"subject"`
	Time    time.Time `json:"time"`
	Content string    `json:"datacontenttype"`
	Trace   string    `json:"traceparent"`
	Data    any       `json:"data"`
}

func NewEvent(kind, order string, now time.Time, data any) Event {
	return Event{Spec: "1.0", Source: "https://omama.vn/services/fulfillment-service", Type: kind, Subject: "order_id:" + order, Time: now, Content: "application/json", Data: data}
}
