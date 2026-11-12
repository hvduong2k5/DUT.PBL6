package returns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/omamx/care-service/internal/domain"
)

const orderReturnContextPath = "/api/v1/internal/orders/"

// HTTPOrderClient implements the proposed Order-owned return-context contract:
// GET /api/v1/internal/orders/{orderId}/care-return-context with X-Internal-Token.
type HTTPOrderClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewHTTPOrderClient(baseURL, token string, client *http.Client) (*HTTPOrderClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("Order base URL must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("Order internal token is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return &HTTPOrderClient{baseURL: strings.TrimRight(parsed.String(), "/"), token: strings.TrimSpace(token), client: client}, nil
}

func (c *HTTPOrderClient) GetReturnContext(ctx context.Context, orderID string) (OrderContext, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" || len(orderID) > 100 {
		return OrderContext{}, fmt.Errorf("%w: order ID is invalid", domain.ErrInvalidArgument)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+orderReturnContextPath+url.PathEscape(orderID)+"/care-return-context", nil)
	if err != nil {
		return OrderContext{}, fmt.Errorf("create Order return-context request: %w", err)
	}
	request.Header.Set("X-Internal-Token", c.token)
	request.Header.Set("Accept", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return OrderContext{}, fmt.Errorf("call Order return-context API: %w", domain.ErrDependencyUnavailable)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil {
		return OrderContext{}, fmt.Errorf("read Order return-context response: %w", domain.ErrDependencyUnavailable)
	}
	if len(body) > 1<<20 {
		return OrderContext{}, fmt.Errorf("Order return-context response exceeded size limit: %w", domain.ErrDependencyUnavailable)
	}
	if response.StatusCode == http.StatusNotFound {
		return OrderContext{}, domain.ErrNotFound
	}
	if response.StatusCode != http.StatusOK {
		return OrderContext{}, fmt.Errorf("Order return-context returned HTTP %d: %w", response.StatusCode, domain.ErrDependencyUnavailable)
	}
	var envelope struct {
		Data OrderContext `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Data.OrderID == "" {
		return OrderContext{}, fmt.Errorf("decode Order return-context response: %w", domain.ErrDependencyUnavailable)
	}
	return envelope.Data, nil
}
