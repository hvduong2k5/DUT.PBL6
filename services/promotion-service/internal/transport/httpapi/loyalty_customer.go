package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var customerUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type customerProfileResolver struct {
	endpoint string
	client   *http.Client
}

type customerProfileErrorKind uint8

const (
	customerProfileAuthError customerProfileErrorKind = iota + 1
	customerProfileForbiddenError
	customerProfileUnavailableError
)

type customerProfileError struct {
	kind customerProfileErrorKind
}

func (e customerProfileError) Error() string { return "customer profile resolution failed" }

func newCustomerProfileResolver(endpoint string) *customerProfileResolver {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil
	}
	return &customerProfileResolver{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 2 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Resolve delegates JWT verification, customer status checks, and user-to-profile
// ID resolution to Profile. The browser cannot choose the customer ID used by the
// Promotion ledger.
func (r *customerProfileResolver) Resolve(ctx context.Context, headers http.Header) (string, error) {
	authorization := headers.Values("Authorization")
	if len(authorization) != 1 {
		return "", customerProfileError{kind: customerProfileAuthError}
	}
	parts := strings.Fields(authorization[0])
	if len(authorization[0]) > 8192 || len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", customerProfileError{kind: customerProfileAuthError}
	}
	if r == nil || r.endpoint == "" {
		return "", customerProfileError{kind: customerProfileUnavailableError}
	}
	u, err := url.ParseRequestURI(r.endpoint)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", customerProfileError{kind: customerProfileUnavailableError}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", customerProfileError{kind: customerProfileUnavailableError}
	}
	request.Header.Set("Authorization", authorization[0])
	request.Header.Set("Accept", "application/json")
	response, err := r.client.Do(request)
	if err != nil {
		return "", customerProfileError{kind: customerProfileUnavailableError}
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusUnauthorized:
		return "", customerProfileError{kind: customerProfileAuthError}
	case http.StatusForbidden, http.StatusNotFound:
		return "", customerProfileError{kind: customerProfileForbiddenError}
	case http.StatusOK:
	default:
		return "", customerProfileError{kind: customerProfileUnavailableError}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return "", customerProfileError{kind: customerProfileUnavailableError}
	}
	var profile struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &profile); err != nil || !customerUUIDPattern.MatchString(profile.ID) || profile.Status == "" {
		return "", customerProfileError{kind: customerProfileUnavailableError}
	}
	if profile.Status != "ACTIVE" {
		return "", customerProfileError{kind: customerProfileForbiddenError}
	}
	return strings.ToLower(profile.ID), nil
}

func customerProfileHTTPError(err error) (int, string, string) {
	var profileErr customerProfileError
	if !errors.As(err, &profileErr) {
		return http.StatusServiceUnavailable, "CUSTOMER_PROFILE_UNAVAILABLE", "customer profile is unavailable"
	}
	switch profileErr.kind {
	case customerProfileAuthError:
		return http.StatusUnauthorized, "UNAUTHENTICATED", "a valid customer bearer token is required"
	case customerProfileForbiddenError:
		return http.StatusForbidden, "CUSTOMER_FORBIDDEN", "customer is not allowed to view loyalty points"
	default:
		return http.StatusServiceUnavailable, "CUSTOMER_PROFILE_UNAVAILABLE", "customer profile is unavailable"
	}
}
