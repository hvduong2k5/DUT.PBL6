package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SupplierContractCommand struct {
	Action, Actor, Key, Reason, RequestHash string
	ContractID                              uuid.UUID
	ExpectedUpdatedAt                       time.Time
	Contract                                *SupplierContract
}
type SupplierContractCommandResult struct {
	RequestHash string
	Snapshot    json.RawMessage
}
