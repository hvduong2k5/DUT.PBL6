package sepay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Pagination struct {
	Total int  `json:"total"`
	Page  int  `json:"current_page"`
	Last  int  `json:"last_page"`
	Size  int  `json:"per_page"`
	More  bool `json:"has_more"`
}
type Page struct {
	Rows       []json.RawMessage
	Pagination Pagination
}
type Window struct {
	Bank     string
	From, To time.Time
	Page     int
}
type Reader interface {
	List(context.Context, Window) (Page, error)
}
type APIError struct {
	Status     int
	RetryAfter time.Duration
}

func (e *APIError) Error() string { return fmt.Sprintf("SEPAY_HTTP_%d", e.Status) }

type Client struct {
	base, token string
	http        *http.Client
}

func NewSandboxClient(token string) (*Client, error) {
	if len(token) != 64 {
		return nil, ErrAuth
	}
	return &Client{"https://userapi-sandbox.sepay.vn/v2", token, &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

// Injecting a transport preserves the pinned Sandbox URL and client safeguards;
// integration harnesses can route this real HTTP client to a private simulator.
func NewSandboxClientWithTransport(token string, transport http.RoundTripper) (*Client, error) {
	c, e := NewSandboxClient(token)
	if e != nil {
		return nil, e
	}
	if transport == nil {
		return nil, ErrInvalid
	}
	c.http.Transport = transport
	return c, nil
}
func (c *Client) List(ctx context.Context, w Window) (Page, error) {
	// Provider date filters use Vietnamese wall time. Keep coverage instants in
	// UTC internally, but send the documented local datetime wire format. Sandbox
	// ignores the Z offset in these filters and otherwise misses seven hours.
	vietnam := time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)
	wireDate := func(at time.Time) string { return at.In(vietnam).Format("2006-01-02 15:04:05") }
	q := url.Values{"bank_account_id": {w.Bank}, "page": {strconv.Itoa(w.Page)}, "per_page": {"100"}, "timestamp_format": {"iso8601"}, "transaction_date_sort": {"asc"}, "transaction_date_from": {wireDate(w.From)}, "transaction_date_to": {wireDate(w.To)}}
	r, e := http.NewRequestWithContext(ctx, "GET", c.base+"/transactions?"+q.Encode(), nil)
	if e != nil {
		return Page{}, e
	}
	r.Header.Set("Authorization", "Bearer "+c.token)
	r.Header.Set("Accept", "application/json")
	resp, e := c.http.Do(r)
	if e != nil {
		return Page{}, ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		delay, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
		if delay < 1 {
			delay = 1
		}
		if delay > 60 {
			delay = 60
		}
		return Page{}, &APIError{resp.StatusCode, time.Duration(delay) * time.Second}
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if e != nil || len(b) > 1<<20 {
		return Page{}, ErrInvalid
	}
	var result struct {
		Status string            `json:"status"`
		Data   []json.RawMessage `json:"data"`
		Meta   struct {
			Pagination Pagination `json:"pagination"`
		} `json:"meta"`
	}
	if Decode(b, &result) != nil || result.Status != "success" || result.Data == nil {
		return Page{}, ErrInvalid
	}
	p := result.Meta.Pagination
	// Canonical Sandbox returns last_page=0 for an empty first page. Accept
	// only that exact shape; a missing/truncated/nonempty page is still invalid.
	empty := w.Page == 1 && p.Page == 1 && p.Size == 100 && p.Last == 0 && p.Total == 0 && !p.More && len(result.Data) == 0
	if !empty && (p.Page != w.Page || p.Size != 100 || p.Last < 1 || p.Total < 0 || len(result.Data) > 100 || p.More != (p.Page < p.Last) || p.Page > p.Last) {
		return Page{}, ErrInvalid
	}
	return Page{result.Data, p}, nil
}
