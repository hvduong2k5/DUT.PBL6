package domain

import "encoding/json"

// Inputs are mutually exclusive; actor is supplied by verified authentication.
type MasterDataCreateCommand struct {
	Kind, Actor, Key, RequestHash string
	Supplier                      *Supplier
	Material                      *RawMaterial
	Offer                         *SupplierMaterialOffer
}

type MasterDataCreateResult struct {
	Kind, RequestHash string
	Snapshot          json.RawMessage
}
