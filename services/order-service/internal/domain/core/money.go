// Package core implements the target Order business rules without network,
// persistence, transport status codes, or an implicit clock. Callers must apply
// its decisions atomically under the aggregate lock and command registry.
package core

const MaxOrderAmount int64 = 1_000_000_000_000

type Violation string

func (e Violation) Error() string { return string(e) }

const (
	ErrMoney     Violation = "INVALID_VND_AMOUNT"
	ErrOverflow  Violation = "MONEY_OVERFLOW"
	ErrSnapshot  Violation = "INVALID_ORDER_SNAPSHOT"
	ErrIdentity  Violation = "INVALID_BUSINESS_IDENTITY"
	ErrVersion   Violation = "ORDER_VERSION_CONFLICT"
	ErrForbidden Violation = "ORDER_SCOPE_DENIED"
	ErrState     Violation = "INVALID_ORDER_TRANSITION"
	ErrDeadline  Violation = "ORDER_DEADLINE_GUARD"
	ErrBlocked   Violation = "ORDER_CONTROL_BLOCKED"
	ErrProof     Violation = "INVALID_RESOURCE_PROOF"
	ErrPayment   Violation = "INVALID_PAYMENT_ALLOCATION"
)

// Money is an immutable Order amount, not a receipt or refund ledger amount.
// A receipt can exceed the Order limit and must be preserved by Payment.
type Money struct{ amount int64 }

func NewMoney(currency string, amount int64) (Money, error) {
	if currency != "VND" || amount < 0 || amount > MaxOrderAmount {
		return Money{}, ErrMoney
	}
	return Money{amount: amount}, nil
}

func (m Money) Amount() int64    { return m.amount }
func (m Money) Currency() string { return "VND" }

func (m Money) Add(other Money) (Money, error) {
	if m.amount > MaxOrderAmount-other.amount {
		return Money{}, ErrOverflow
	}
	return Money{amount: m.amount + other.amount}, nil
}

func (m Money) Multiply(quantity int64) (Money, error) {
	if quantity < 0 {
		return Money{}, ErrMoney
	}
	if m.amount != 0 && quantity > MaxOrderAmount/m.amount {
		return Money{}, ErrOverflow
	}
	return Money{amount: m.amount * quantity}, nil
}
