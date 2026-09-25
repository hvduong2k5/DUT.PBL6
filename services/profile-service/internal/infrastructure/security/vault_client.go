package security

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/omamx/profile-service/internal/domain"
)

type HTTPVaultClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewHTTPVaultClient(baseURL, token string) *HTTPVaultClient {
	return &HTTPVaultClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type vaultEncryptReq struct {
	Plaintext string `json:"plaintext"`
}

type vaultDecryptReq struct {
	Ciphertext string `json:"ciphertext"`
}

type vaultRewrapReq struct {
	Ciphertext string `json:"ciphertext"`
}

type vaultTransitResponse struct {
	Data struct {
		Ciphertext string `json:"ciphertext"`
		Plaintext  string `json:"plaintext"`
		KeyVersion int    `json:"key_version"`
	} `json:"data"`
	Errors []string `json:"errors"`
}

func (c *HTTPVaultClient) EncryptDEK(ctx context.Context, keyName string, dek []byte) ([]byte, int, error) {
	url := fmt.Sprintf("%s/v1/transit/encrypt/%s", c.baseURL, keyName)
	b64Plaintext := base64.StdEncoding.EncodeToString(dek)

	reqBody, _ := json.Marshal(vaultEncryptReq{Plaintext: b64Plaintext})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("X-Vault-Token", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", domain.ErrVaultUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("%w: status %d: %s", domain.ErrVaultUnavailable, resp.StatusCode, string(body))
	}

	var vResp vaultTransitResponse
	if err := json.NewDecoder(resp.Body).Decode(&vResp); err != nil {
		return nil, 0, err
	}

	// Parse version if embedded in vault ciphertext (e.g. vault:v1:...)
	version := vResp.Data.KeyVersion
	if version == 0 {
		version = parseVaultKeyVersion(vResp.Data.Ciphertext)
	}

	return []byte(vResp.Data.Ciphertext), version, nil
}

func (c *HTTPVaultClient) DecryptDEK(ctx context.Context, keyName string, encryptedDEK []byte, version int) ([]byte, error) {
	url := fmt.Sprintf("%s/v1/transit/decrypt/%s", c.baseURL, keyName)

	reqBody, _ := json.Marshal(vaultDecryptReq{Ciphertext: string(encryptedDEK)})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Vault-Token", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrVaultUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", domain.ErrVaultUnavailable, resp.StatusCode, string(body))
	}

	var vResp vaultTransitResponse
	if err := json.NewDecoder(resp.Body).Decode(&vResp); err != nil {
		return nil, err
	}

	dek, err := base64.StdEncoding.DecodeString(vResp.Data.Plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed decoding base64 DEK plaintext: %w", err)
	}

	return dek, nil
}

func (c *HTTPVaultClient) RewrapDEK(ctx context.Context, keyName string, encryptedDEK []byte, version int) ([]byte, int, error) {
	url := fmt.Sprintf("%s/v1/transit/rewrap/%s", c.baseURL, keyName)

	reqBody, _ := json.Marshal(vaultRewrapReq{Ciphertext: string(encryptedDEK)})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("X-Vault-Token", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", domain.ErrVaultUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("%w: status %d: %s", domain.ErrVaultUnavailable, resp.StatusCode, string(body))
	}

	var vResp vaultTransitResponse
	if err := json.NewDecoder(resp.Body).Decode(&vResp); err != nil {
		return nil, 0, err
	}

	newVersion := vResp.Data.KeyVersion
	if newVersion == 0 {
		newVersion = parseVaultKeyVersion(vResp.Data.Ciphertext)
	}

	return []byte(vResp.Data.Ciphertext), newVersion, nil
}

func parseVaultKeyVersion(ciphertext string) int {
	parts := strings.Split(ciphertext, ":")
	if len(parts) >= 2 && strings.HasPrefix(parts[1], "v") {
		var ver int
		if _, err := fmt.Sscanf(parts[1], "v%d", &ver); err == nil && ver > 0 {
			return ver
		}
	}
	return 1
}
