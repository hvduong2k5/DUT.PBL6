// Package sepay implements the Payment Gateway protocol, not SePay bank webhooks.
package sepay

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment string
	MerchantID  string
	SecretKey   string
	IPNSecret   string
	ReturnURL   string
}

func (c Config) Validate() error {
	if c.Environment != "sandbox" {
		return errors.New("this delivery enables SePay sandbox only")
	}
	if strings.TrimSpace(c.MerchantID) == "" || len(c.SecretKey) < 16 || len(c.IPNSecret) < 16 {
		return errors.New("SePay merchant, secret and SECRET_KEY IPN authentication are required")
	}
	u, err := url.Parse(c.ReturnURL)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || strings.ContainsAny(c.ReturnURL, "\r\n,") || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"))) {
		return errors.New("SEPAY_RETURN_URL must use HTTPS (HTTP loopback is allowed for sandbox frontend development), without fragment or comma")
	}
	return nil
}

func (c Config) Provider() string { return "SEPAY_" + strings.ToUpper(c.Environment) }
func (c Config) VerifyIPN(secret string) bool {
	return c.IPNSecret != "" && hmac.Equal([]byte(c.IPNSecret), []byte(secret))
}

// Fields is ordered: SePay signs the comma-separated name=value pairs in form order.
type Field struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type Checkout struct {
	AttemptID string    `json:"attempt_id"`
	Invoice   string    `json:"invoice"`
	ExpiresAt time.Time `json:"expires_at"`
	Action    string    `json:"action"`
	Method    string    `json:"method"`
	Fields    []Field   `json:"fields"`
}

func (c Config) Checkout(attempt, invoice string, amount int64, expiry time.Time) Checkout {
	fields := []Field{{"merchant", c.MerchantID}, {"operation", "PURCHASE"}, {"payment_method", "BANK_TRANSFER"}, {"order_invoice_number", invoice}, {"order_amount", strconv.FormatInt(amount, 10)}, {"currency", "VND"}}
	// The live sandbox rejects a checkout without a description, although the
	// SDK reference marks it optional. Include and sign it before return URLs.
	fields = append(fields, Field{"order_description", "Thanh toan don hang " + invoice})
	for _, outcome := range []string{"success", "error", "cancel"} {
		u, _ := url.Parse(c.ReturnURL)
		q := u.Query()
		q.Set("attempt_id", attempt)
		q.Set("result", outcome)
		u.RawQuery = q.Encode()
		fields = append(fields, Field{outcome + "_url", u.String()})
	}
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = f.Name + "=" + f.Value
	}
	mac := hmac.New(sha256.New, []byte(c.SecretKey))
	mac.Write([]byte(strings.Join(parts, ",")))
	fields = append(fields, Field{"signature", base64.StdEncoding.EncodeToString(mac.Sum(nil))})
	return Checkout{attempt, invoice, expiry, "https://pay-sandbox.sepay.vn/v1/checkout/init", "POST", fields}
}

// Scalar accepts provider IDs and money encoded as JSON strings or integers,
// without ever converting money through float64.
type Scalar string

func (s *Scalar) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*s = Scalar(v)
		return nil
	}
	if string(b) == "null" {
		return errors.New("required scalar is null")
	}
	var number json.Number
	if err := json.Unmarshal(b, &number); err != nil {
		return errors.New("invalid scalar")
	}
	*s = Scalar(number.String())
	return nil
}

var integerVND = regexp.MustCompile(`^[0-9]{1,13}(?:\.0{1,2})?$`)

func VND(v Scalar) (int64, error) {
	if !integerVND.MatchString(string(v)) {
		return 0, errors.New("not integer VND")
	}
	n, err := strconv.ParseInt(strings.SplitN(string(v), ".", 2)[0], 10, 64)
	if err != nil || n <= 0 || n > 1_000_000_000_000 {
		return 0, errors.New("VND out of range")
	}
	return n, nil
}

type Transaction struct {
	ID       Scalar `json:"id"`
	Method   string `json:"payment_method"`
	Type     string `json:"transaction_type"`
	Status   string `json:"transaction_status"`
	Amount   Scalar `json:"transaction_amount"`
	Currency string `json:"transaction_currency"`
	Date     string `json:"transaction_date"`
}
type Order struct {
	ID           Scalar        `json:"id"`
	OrderID      string        `json:"order_id"`
	Invoice      string        `json:"order_invoice_number"`
	Status       string        `json:"order_status"`
	Amount       Scalar        `json:"order_amount"`
	Currency     string        `json:"order_currency"`
	Transactions []Transaction `json:"transactions"`
}
type Notification struct {
	Timestamp   int64       `json:"timestamp"`
	Type        string      `json:"notification_type"`
	Order       Order       `json:"order"`
	Transaction Transaction `json:"transaction"`
}
type Payment struct {
	TransactionID, Invoice, OrderID string
	Amount                          int64
	PaidAt                          time.Time
}

func Normalize(o Order, t Transaction) (Payment, error) {
	if o.Currency != "VND" || t.Currency != "VND" || t.Status != "APPROVED" || t.Method != "BANK_TRANSFER" || t.Type != "PAYMENT" {
		return Payment{}, errors.New("unsupported or unconfirmed payment")
	}
	if len(t.ID) == 0 || len(t.ID) > 128 || len(o.Invoice) == 0 || len(o.Invoice) > 128 || o.OrderID == "" || len(o.OrderID) > 128 {
		return Payment{}, errors.New("missing payment identity")
	}
	amount, err := VND(t.Amount)
	if err != nil {
		return Payment{}, err
	}
	// Transaction amount is the actual receipt, not the order's requested amount.
	zone := time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)
	paidAt, err := time.ParseInLocation("2006-01-02 15:04:05", t.Date, zone)
	if err != nil {
		return Payment{}, errors.New("invalid transaction date")
	}
	return Payment{string(t.ID), o.Invoice, o.OrderID, amount, paidAt.UTC()}, nil
}

type Client struct {
	Config  Config
	HTTP    *http.Client
	BaseURL string
}

func New(c Config) *Client {
	return &Client{c, &http.Client{Timeout: 15 * time.Second}, "https://pgapi-sandbox.sepay.vn/v1"}
}
func (c *Client) get(ctx context.Context, path string, out any) error {
	r, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	r.SetBasicAuth(c.Config.MerchantID, c.Config.SecretKey)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("User-Agent", "order-service-sepay/1.0")
	resp, err := c.HTTP.Do(r)
	if err != nil {
		return errors.New("SePay query unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("SePay query returned HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if err != nil || len(b) > 2<<20 {
		return errors.New("SePay response too large")
	}
	return json.Unmarshal(b, out)
}
func (c *Client) Detail(ctx context.Context, invoice, providerOrderID string) (Order, error) {
	var response struct {
		Data Order `json:"data"`
	}
	err := c.get(ctx, "/order/detail/"+url.PathEscape(invoice), &response)
	if err == nil && (response.Data.OrderID != providerOrderID || response.Data.Invoice != invoice) {
		err = errors.New("SePay detail identity mismatch")
	}
	return response.Data, err
}

// Orders returns one stable page at a time. No status filter: cancelled orders
// and late money must remain visible to reconciliation.
func (c *Client) Orders(ctx context.Context, page int) ([]Order, int, error) {
	var response struct {
		Data []Order `json:"data"`
		Meta struct {
			Current int  `json:"current_page"`
			Last    int  `json:"page_count"`
			HasMore bool `json:"has_more"`
		} `json:"meta"`
	}
	err := c.get(ctx, "/order?per_page=10&page="+strconv.Itoa(page)+"&sort=created_at%3Aasc", &response)
	if err != nil {
		return nil, 0, err
	}
	if response.Meta.Current != page || response.Meta.Last < 0 || (response.Meta.Last == 0 && len(response.Data) != 0) || (response.Meta.Last > 0 && response.Meta.Last < page) || response.Meta.HasMore != (page < response.Meta.Last) {
		return nil, 0, errors.New("SePay pagination evidence missing")
	}
	return response.Data, response.Meta.Last, nil
}
