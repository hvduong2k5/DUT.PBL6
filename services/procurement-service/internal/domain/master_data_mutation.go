package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Mutation keys form one actor-scoped namespace across the six master actions,
// separate from create/PO/receiving keys. Authentication supplies Actor.
type MasterDataMutation struct {
	Action              string
	Actor               string
	Key                 string
	ResourceID          uuid.UUID
	SupplierID          uuid.UUID
	ExpectedUpdatedAt   time.Time
	Supplier            *Supplier
	Material            *RawMaterial
	Status              string
	Active              *bool
	PreferredSupplierID uuid.UUID
	RequestHash         string
}
type MasterDataMutationResult struct {
	RequestHash string
	Snapshot    json.RawMessage
}
