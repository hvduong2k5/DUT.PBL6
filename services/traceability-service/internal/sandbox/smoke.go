package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/jsonutil"
)

type SmokeReport struct {
	Mode        string    `json:"mode"`
	Outcome     string    `json:"outcome"`
	ProfileID   string    `json:"profile_id"`
	QRCode      string    `json:"qr_code"`
	QRURL       string    `json:"qr_url"`
	CompletedAt time.Time `json:"completed_at"`
}

func Smoke(ctx context.Context, mode, dir, apiURL string) (SmokeReport, error) {
	if mode != "sandbox" {
		return SmokeReport{}, errors.New("smoke is sandbox only")
	}
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return SmokeReport{}, errors.New("invalid local API URL")
	}
	ip := net.ParseIP(parsed.Hostname())
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || (parsed.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback())) || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return SmokeReport{}, errors.New("smoke API URL must be an explicit loopback HTTP(S) origin")
	}
	state, err := Load(dir)
	if err != nil {
		return SmokeReport{}, err
	}
	if err = RefreshTokens(mode, dir, time.Now()); err != nil {
		return SmokeReport{}, err
	}
	editor, err := os.ReadFile(filepath.Join(dir, "editor.jwt"))
	if err != nil {
		return SmokeReport{}, errors.New("editor token missing")
	}
	approver, err := os.ReadFile(filepath.Join(dir, "approver.jwt"))
	if err != nil {
		return SmokeReport{}, errors.New("approver token missing")
	}
	seed, err := os.ReadFile(filepath.Join(dir, "create.json"))
	if err != nil {
		return SmokeReport{}, errors.New("synthetic create input missing")
	}
	var request domain.CreateDraftRequest
	if jsonutil.Decode(seed, &request) != nil {
		return SmokeReport{}, errors.New("synthetic create input invalid")
	}
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return errors.New("smoke does not follow redirects with credentials")
	}}
	call := func(method, path, key string, body any, token []byte, want int) ([]byte, error) {
		var data []byte
		var err error
		if body != nil {
			data, err = json.Marshal(body)
			if err != nil {
				return nil, err
			}
		}
		req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(apiURL, "/")+path, bytes.NewReader(data))
		if err != nil {
			return nil, errors.New("cannot create smoke request")
		}
		if len(token) > 0 {
			req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
		}
		if key != "" {
			req.Header.Set("X-Idempotency-Key", state.ID+"-"+key)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		response, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("smoke %s request failed", key)
		}
		defer response.Body.Close()
		data, err = io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
		if err != nil || len(data) > 1<<20 {
			return nil, errors.New("smoke response too large or unreadable")
		}
		if response.StatusCode != want {
			return nil, fmt.Errorf("smoke %s expected HTTP %d, got %d (response omitted)", key, want, response.StatusCode)
		}
		if response.Header.Get("Cache-Control") != "no-store" {
			return nil, errors.New("smoke response is missing no-store")
		}
		return data, nil
	}
	data, err := call("POST", "/api/v1/admin/trace/records", "create", request, editor, 201)
	if err != nil {
		return SmokeReport{}, err
	}
	var created domain.Profile
	if json.Unmarshal(data, &created) != nil || created.ID == "" {
		return SmokeReport{}, errors.New("create response invalid")
	}
	root := "/api/v1/admin/trace/records/" + created.ID
	public := "/api/v1/trace/" + created.QRCode
	data, err = call("GET", root, "inspect", nil, editor, 200)
	if err != nil {
		return SmokeReport{}, err
	}
	var current domain.Profile
	if json.Unmarshal(data, &current) != nil {
		return SmokeReport{}, errors.New("profile response invalid")
	}
	report := SmokeReport{Mode: "sandbox", Outcome: "full_flow_passed", ProfileID: created.ID, QRCode: created.QRCode, QRURL: current.QRURL}
	if current.EverPublished && (current.Status == "SUSPENDED" || current.Status == "ARCHIVED") {
		if _, err = call("GET", public, "existing-terminal-qr", nil, nil, 410); err != nil {
			return SmokeReport{}, err
		}
		report.Outcome = "existing_terminal_fixture_verified"
	} else {
		for _, step := range []struct {
			suffix, key string
			body        any
			token       []byte
			want        int
		}{
			{"/revisions/1/submit", "submit", domain.RevisionCommand{ExpectedVersion: 1}, editor, 200},
			{"/revisions/1/approve", "approve", domain.RevisionCommand{ExpectedVersion: 2, Reason: "Synthetic independent evidence checked"}, approver, 200},
			{"/revisions/1/publish", "publish", domain.RevisionCommand{ExpectedVersion: 3}, approver, 200},
		} {
			if _, err = call("POST", root+step.suffix, step.key, step.body, step.token, step.want); err != nil {
				return SmokeReport{}, err
			}
		}
		before, err := call("GET", public, "public-first", nil, nil, 200)
		if err != nil {
			return SmokeReport{}, err
		}
		var record domain.PublicRecord
		if json.Unmarshal(before, &record) != nil || record.Mode != "sandbox" || record.VerificationMethod != "MANUAL" || record.Revision != 1 || record.QRCode != created.QRCode || bytes.Contains(before, []byte("sandbox/evidence/")) {
			return SmokeReport{}, errors.New("public DTO metadata or privacy failed")
		}
		if _, err = call("POST", root+"/revisions", "new-revision", domain.NewRevisionRequest{ExpectedVersion: 4, FromRevision: 1}, editor, 201); err != nil {
			return SmokeReport{}, err
		}
		edited := request.Content
		edited.ProductName += " — next draft"
		if _, err = call("PATCH", root+"/revisions/2", "edit-next", domain.EditDraftRequest{ExpectedVersion: 5, Content: edited}, editor, 200); err != nil {
			return SmokeReport{}, err
		}
		after, err := call("GET", public, "public-unchanged", nil, nil, 200)
		if err != nil {
			return SmokeReport{}, err
		}
		if !bytes.Equal(before, after) {
			return SmokeReport{}, errors.New("draft edit changed the public snapshot")
		}
		if _, err = call("POST", root+"/suspend", "suspend", domain.RevisionCommand{ExpectedVersion: 6, Reason: "Synthetic review scenario complete"}, approver, 200); err != nil {
			return SmokeReport{}, err
		}
		if _, err = call("GET", public, "suspended-qr", nil, nil, 410); err != nil {
			return SmokeReport{}, err
		}
	}
	report.CompletedAt = time.Now().UTC()
	data, err = json.MarshalIndent(report, "", "  ")
	if err != nil {
		return SmokeReport{}, err
	}
	if err = os.WriteFile(filepath.Join(dir, "smoke-report.json"), data, 0600); err != nil {
		return SmokeReport{}, errors.New("cannot save smoke report")
	}
	return report, nil
}
