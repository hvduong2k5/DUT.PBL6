package sepay

import (
	"net/url"
	"strconv"
)

type Receiver struct {
	BankID   string `json:"bank_account_uuid"`
	BankCode string `json:"bank_code"`
	BankName string `json:"bank_name"`
	Account  string `json:"account_number"`
	Holder   string `json:"account_holder"`
	Mode     string `json:"mode"`
	Currency string `json:"currency"`
}

func (r Receiver) Validate() error {
	if !ValidUUID(r.BankID) || r.BankCode == "" || r.BankName == "" || r.Account == "" || r.Holder == "" || r.Currency != "VND" || r.Mode != "BANK_ACCOUNT" {
		return ErrInvalid
	}
	return nil
}

// Other VA/SEVQR modes require capability-specific onboarding, not a generic fallback.
func (r Receiver) QR(code string, amount int64) (string, error) {
	if r.Validate() != nil || !ValidCode(code) || amount < 1 || amount > 1e12 {
		return "", ErrInvalid
	}
	q := url.Values{"acc": {r.Account}, "bank": {r.BankCode}, "amount": {strconv.FormatInt(amount, 10)}, "des": {code}, "template": {"compact"}}
	return "https://vietqr.app/img?" + q.Encode(), nil
}
