package domain

import "github.com/google/uuid"

type VarianceResolutionCommand struct {
	Actor, Key, RequestHash, Decision, Note string
	VarianceID                              uuid.UUID
	ExpectedVersion                         int64
}

type VarianceResolutionResult struct {
	RequestHash string
	Snapshot    GoodsReceipt
}
