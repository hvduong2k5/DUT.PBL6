package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omamx/shipping-service/internal/application"
)

type Directory struct {
	base   string
	client *http.Client
}

func NewDirectory(base string, local bool) (*Directory, error) {
	u, err := url.Parse(base)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, fmt.Errorf("Identity base URL must be an absolute origin")
	}
	ip := net.ParseIP(u.Hostname())
	loopback := u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())
	if u.Scheme != "https" && !(local && loopback && u.Scheme == "http") {
		return nil, fmt.Errorf("Identity requires HTTPS; local HTTP is restricted to loopback")
	}
	return &Directory{base: strings.TrimRight(base, "/"), client: &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

type principal struct {
	ID      string `json:"principalId"`
	Subject string `json:"idpSubject"`
	Type    string `json:"principalType"`
	Status  string `json:"provisioningStatus"`
}

func (d *Directory) read(ctx context.Context, path, authorization string, driver bool) (principal, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.base+path, nil)
	if err != nil {
		return principal{}, application.ErrEmployeeDirectoryUnavailable
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Accept", "application/json")
	response, err := d.client.Do(req)
	if err != nil {
		return principal{}, application.ErrEmployeeDirectoryUnavailable
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case 401:
		return principal{}, application.ErrEmployeeUnauthenticated
	case 403:
		return principal{}, application.ErrEmployeeForbidden
	case 400, 404:
		// Identity's getUserDetail throws IllegalArgumentException for a missing UUID.
		if driver {
			return principal{}, application.ErrDriverIneligible
		}
		return principal{}, application.ErrEmployeeForbidden
	case 200:
	default:
		return principal{}, application.ErrEmployeeDirectoryUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return principal{}, application.ErrEmployeeDirectoryUnavailable
	}
	var envelope struct {
		Success bool      `json:"success"`
		Data    principal `json:"data"`
	}
	if json.Unmarshal(body, &envelope) != nil || !envelope.Success {
		return principal{}, application.ErrEmployeeDirectoryUnavailable
	}
	return envelope.Data, nil
}

func (d *Directory) ValidateActor(ctx context.Context, authorization string, id uuid.UUID, subject string) error {
	p, err := d.read(ctx, "/api/v1/users/me", authorization, false)
	if err != nil {
		return err
	}
	if p.ID != id.String() || p.Subject != subject || subject == "" {
		return application.ErrEmployeeDirectoryUnavailable
	}
	if p.Type != "EMPLOYEE" || p.Status != "ACTIVE" {
		return application.ErrEmployeeForbidden
	}
	return nil
}

func (d *Directory) ValidateDriver(ctx context.Context, authorization string, id uuid.UUID) error {
	if id == uuid.Nil {
		return application.ErrDriverIneligible
	}
	p, err := d.read(ctx, "/api/v1/admin/users/"+id.String(), authorization, true)
	if err != nil {
		return err
	}
	if p.ID != id.String() {
		return application.ErrEmployeeDirectoryUnavailable
	}
	if p.Type != "EMPLOYEE" || p.Status != "ACTIVE" {
		return application.ErrDriverIneligible
	}
	return nil
}
