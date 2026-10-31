// Package sepay implements the bank webhook/API v2 protocol, not hosted gateway IPN.
package sepay

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalid = errors.New("SEPAY_INVALID_DATA")
var ErrAuth = errors.New("SEPAY_AUTHENTICATION_FAILED")
var ErrUnavailable = errors.New("SEPAY_UNAVAILABLE")

// Decode rejects duplicate keys at every depth, invalid/trailing JSON and lossy numbers.
func Decode(raw []byte, out any) error {
	if len(raw) == 0 || len(raw) > 1<<20 {
		return ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func() error
	walk = func() error {
		t, e := d.Token()
		if e != nil {
			return e
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return e
				}
				s, ok := k.(string)
				if !ok || seen[s] {
					return ErrInvalid
				}
				seen[s] = true
				if e = walk(); e != nil {
					return e
				}
			}
		case '[':
			for d.More() {
				if e := walk(); e != nil {
					return e
				}
			}
		default:
			return ErrInvalid
		}
		_, e = d.Token()
		return e
	}
	if e := walk(); e != nil {
		return ErrInvalid
	}
	if _, e := d.Token(); e != io.EOF {
		return ErrInvalid
	}
	if e := json.Unmarshal(raw, out); e != nil {
		return ErrInvalid
	}
	return nil
}

type Webhook struct {
	ID        int64  `json:"id"`
	Gateway   string `json:"gateway"`
	Date      string `json:"transactionDate"`
	Account   string `json:"accountNumber"`
	VA        string `json:"subAccount"`
	Content   string `json:"content"`
	Code      string `json:"code"`
	Direction string `json:"transferType"`
	Amount    int64  `json:"transferAmount"`
	Reference string `json:"referenceCode"`
}

func BankTime(s string) (time.Time, error) {
	if t, e := time.Parse(time.RFC3339, s); e == nil {
		return t.UTC(), nil
	}
	t, e := time.ParseInLocation("2006-01-02 15:04:05", s, time.FixedZone("Asia/Ho_Chi_Minh", 7*3600))
	if e != nil {
		return time.Time{}, ErrInvalid
	}
	return t.UTC(), nil
}
func Verify(raw []byte, timestamp, signature, secret string, now time.Time) error {
	n, e := strconv.ParseInt(timestamp, 10, 64)
	if e != nil || strconv.FormatInt(n, 10) != timestamp || n < 0 || n < now.Unix()-300 || n > now.Unix()+300 {
		return ErrAuth
	}
	if !strings.HasPrefix(signature, "sha256=") {
		return ErrAuth
	}
	mac, e := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if e != nil || len(mac) != 32 {
		return ErrAuth
	}
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(timestamp + "."))
	h.Write(raw)
	if !hmac.Equal(h.Sum(nil), mac) {
		return ErrAuth
	}
	return nil
}
func ParseWebhook(raw []byte) (Webhook, error) {
	var w Webhook
	if e := Decode(raw, &w); e != nil {
		return w, e
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields["content"] == nil || bytes.Equal(bytes.TrimSpace(fields["content"]), []byte("null")) || len(w.Date) != 19 {
		return w, ErrInvalid
	}
	if w.ID < 1 || w.Gateway == "" || w.Account == "" || w.Amount < 1 || (w.Direction != "in" && w.Direction != "out") {
		return w, ErrInvalid
	}
	if _, e := BankTime(w.Date); e != nil {
		return w, e
	}
	return w, nil
}

type Transaction struct {
	ID            string `json:"id"`
	BankAccountID string `json:"bank_account_id"`
	Account       string `json:"account_number"`
	VA            string `json:"va"`
	VAID          string `json:"va_id"`
	Direction     string `json:"transfer_type"`
	In            int64  `json:"amount_in"`
	Out           int64  `json:"amount_out"`
	Date          string `json:"transaction_date"`
	Content       string `json:"transaction_content"`
	Reference     string `json:"reference_number"`
	Code          string `json:"code"`
}

func ValidUUID(s string) bool {
	u, e := uuid.Parse(s)
	return e == nil && u != uuid.Nil && u.String() == s
}
func (t Transaction) Validate() error {
	if !ValidUUID(t.ID) || !ValidUUID(t.BankAccountID) || t.Account == "" || t.In < 0 || t.Out < 0 {
		return ErrInvalid
	}
	if _, e := BankTime(t.Date); e != nil {
		return e
	}
	if (t.Direction == "in" && t.In > 0 && t.Out == 0) || (t.Direction == "out" && t.Out > 0 && t.In == 0) {
		return nil
	}
	return ErrInvalid
}

// FinancialBytes excludes derived code/webhook_success/accumulated and pins UTC bank time.
func (t Transaction) FinancialBytes() ([]byte, error) {
	if e := t.Validate(); e != nil {
		return nil, e
	}
	at, _ := BankTime(t.Date)
	return json.Marshal(struct {
		ID, Bank, Account, VA, VAID, Direction string
		In, Out                                int64
		At                                     time.Time
		Content, Reference                     string
	}{t.ID, t.BankAccountID, t.Account, t.VA, t.VAID, t.Direction, t.In, t.Out, at, t.Content, t.Reference})
}
func (w Webhook) Matches(t Transaction) bool {
	a, e := BankTime(w.Date)
	b, f := BankTime(t.Date)
	amount := t.In
	if t.Direction == "out" {
		amount = t.Out
	}
	return e == nil && f == nil && w.Reference != "" && w.Reference == t.Reference && w.Account == t.Account && w.VA == t.VA && w.Direction == t.Direction && w.Amount == amount && w.Content == t.Content && a.Equal(b)
}

var codeRE = regexp.MustCompile(`^OMP[A-Z0-9]{12}$`)

func ValidCode(s string) bool { return codeRE.MatchString(s) }
func PaymentCode(content, hint string) (string, string) {
	// ASCII normalization only; split whole alphanumeric tokens to reject longer matches.
	upper := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r - 32
		}
		return r
	}, content)
	tokens := strings.FieldsFunc(upper, func(r rune) bool { return !(r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') })
	codes := []string{}
	for _, s := range tokens {
		if ValidCode(s) {
			codes = append(codes, s)
		}
	}
	if len(codes) > 1 {
		return "", "AMBIGUOUS_REFERENCE"
	}
	if len(codes) == 0 {
		return "", "UNMATCHED_REFERENCE"
	}
	if hint != "" && strings.ToUpper(hint) != codes[0] {
		return "", "REFERENCE_CONFLICT"
	}
	return codes[0], ""
}
