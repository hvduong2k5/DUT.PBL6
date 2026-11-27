package domain

// PurchaseOrderActionCommand is internal transaction metadata, not a caller's
// identity or an HTTP response. Actor comes from verified employee authentication.
type PurchaseOrderActionCommand struct {
	Kind, HistoryAction, Actor, Key, RequestHash, Reason string
	ExpectedVersion                                      int64
}

type PurchaseOrderActionResult struct {
	RequestHash string
	Snapshot    PurchaseOrder
}
