package ghn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	Code               = "GHN"
	defaultMaxResponse = int64(2 << 20)
)

type Config struct {
	BaseURL          string
	Token            string
	ShopID           int64
	FromDistrictID   int
	FromWardCode     string
	Timeout          time.Duration
	ReconcileTimeout time.Duration
	QuoteTTL         time.Duration
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	baseURL          *url.URL
	token            string
	shopID           int64
	fromDistrictID   int
	fromWardCode     string
	quoteTTL         time.Duration
	reconcileTimeout time.Duration
	httpClient       HTTPDoer
	now              func() time.Time
}

type APIError struct {
	HTTPStatus     int
	Code           int
	CodeMessage    string
	Message        string
	MessageForUser string
}

func (e *APIError) Error() string {
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = "GHN request failed"
	}
	if e.CodeMessage != "" {
		return fmt.Sprintf("GHN %s: %s", e.CodeMessage, message)
	}
	return fmt.Sprintf("GHN HTTP %d: %s", e.HTTPStatus, message)
}

func (e *APIError) Temporary() bool {
	if e.HTTPStatus > 0 {
		return e.HTTPStatus == http.StatusRequestTimeout || e.HTTPStatus == http.StatusTooManyRequests || e.HTTPStatus >= http.StatusInternalServerError
	}
	return e.Code == http.StatusRequestTimeout || e.Code == http.StatusTooManyRequests || e.Code >= http.StatusInternalServerError
}

func New(cfg Config) (*Client, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return newClient(cfg, &http.Client{Timeout: timeout})
}

func NewWithHTTPClient(cfg Config, httpClient HTTPDoer) (*Client, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("GHN HTTP client is required")
	}
	return newClient(cfg, httpClient)
}

func newClient(cfg Config, httpClient HTTPDoer) (*Client, error) {
	baseURL, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil || baseURL.Scheme != "https" && baseURL.Scheme != "http" || baseURL.Host == "" {
		return nil, fmt.Errorf("valid GHN base URL is required")
	}
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, fmt.Errorf("GHN token is required")
	}
	if cfg.ShopID <= 0 {
		return nil, fmt.Errorf("positive GHN shop ID is required")
	}
	if cfg.FromDistrictID <= 0 || strings.TrimSpace(cfg.FromWardCode) == "" {
		return nil, fmt.Errorf("GHN pickup district ID and ward code are required")
	}
	quoteTTL := cfg.QuoteTTL
	if quoteTTL <= 0 {
		quoteTTL = 15 * time.Minute
	}
	reconcileTimeout := cfg.ReconcileTimeout
	if reconcileTimeout <= 0 {
		reconcileTimeout = 3 * time.Second
	}
	return &Client{
		baseURL: baseURL, token: strings.TrimSpace(cfg.Token), shopID: cfg.ShopID,
		fromDistrictID: cfg.FromDistrictID, fromWardCode: strings.TrimSpace(cfg.FromWardCode),
		quoteTTL: quoteTTL, reconcileTimeout: reconcileTimeout, httpClient: httpClient, now: time.Now,
	}, nil
}

type apiEnvelope struct {
	Code             int             `json:"code"`
	Message          string          `json:"message"`
	MessageDisplay   string          `json:"message_display"`
	CodeMessage      string          `json:"code_message"`
	CodeMessageValue string          `json:"code_message_value"`
	Data             json.RawMessage `json:"data"`
}

func (c *Client) do(ctx context.Context, method, path string, input, output any) error {
	endpoint := *c.baseURL
	relative, err := url.Parse(path)
	if err != nil {
		return fmt.Errorf("parse GHN request path: %w", err)
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + strings.TrimLeft(relative.Path, "/")
	endpoint.RawQuery = relative.RawQuery

	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode GHN request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return fmt.Errorf("create GHN request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Token", c.token)
	req.Header.Set("ShopId", strconv.FormatInt(c.shopID, 10))
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call GHN: %w", err)
	}
	defer response.Body.Close()

	var envelope apiEnvelope
	decoder := json.NewDecoder(io.LimitReader(response.Body, defaultMaxResponse))
	if err := decoder.Decode(&envelope); err != nil {
		return fmt.Errorf("decode GHN response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices || envelope.Code != http.StatusOK {
		return &APIError{
			HTTPStatus: response.StatusCode, Code: envelope.Code, CodeMessage: envelope.CodeMessage,
			Message: envelope.Message, MessageForUser: envelope.CodeMessageValue,
		}
	}
	if output == nil {
		return nil
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return fmt.Errorf("GHN response data is empty")
	}
	if err := json.Unmarshal(envelope.Data, output); err != nil {
		return fmt.Errorf("decode GHN response data: %w", err)
	}
	return nil
}
