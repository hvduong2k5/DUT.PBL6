package domain

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrLeaseLost = errors.New("outbox lease no longer owned")

type OutboxMessage struct {
	ID         string
	ProfileID  string
	Version    int64
	Type       string
	Topic      string
	Key        string
	Payload    json.RawMessage
	Attempts   int
	LeaseToken string
	LeaseUntil time.Time
}
type OutboxStats struct {
	Pending              int64
	Failed               int64
	OldestPendingSeconds float64
}
