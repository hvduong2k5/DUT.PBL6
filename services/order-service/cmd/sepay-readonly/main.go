// sepay-readonly exercises the exact runtime API client without a database,
// webhook, ledger write, live host or bank mutation. It never prints credentials
// or financial transaction contents.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/omamx/order-service/internal/sepay"
	"os"
	"strings"
	"time"
)

func main() {
	tokenFile := flag.String("token-file", "", "private Sandbox API token file")
	accountsFile := flag.String("accounts-file", "", "private GET /v2/bank-accounts response file")
	flag.Parse()
	fail := func(code string) {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "FAIL", "mode": "sandbox-read-only", "code": code, "financial_writes": false})
		os.Exit(1)
	}
	token, e := os.ReadFile(*tokenFile)
	if e != nil {
		fail("TOKEN_FILE_UNAVAILABLE")
	}
	raw, e := os.ReadFile(*accountsFile)
	if e != nil {
		fail("ACCOUNT_METADATA_UNAVAILABLE")
	}
	var accounts struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &accounts) != nil || len(accounts.Data) == 0 || !sepay.ValidUUID(accounts.Data[0].ID) {
		fail("INVALID_ACCOUNT_METADATA")
	}
	reader, e := sepay.NewSandboxClient(strings.TrimSpace(string(token)))
	if e != nil {
		fail("INVALID_SANDBOX_CREDENTIAL_SHAPE")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	now := time.Now().UTC()
	page, e := reader.List(ctx, sepay.Window{Bank: accounts.Data[0].ID, From: now.Add(-7 * 24 * time.Hour), To: now, Page: 1})
	if e != nil {
		fail(fmt.Sprint(e))
	}
	valid := 0
	for _, raw := range page.Rows {
		var row sepay.Transaction
		if sepay.Decode(raw, &row) != nil || row.Validate() != nil || row.BankAccountID != accounts.Data[0].ID {
			fail("CANONICAL_ROW_REJECTED")
		}
		valid++
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "PASS", "mode": "sandbox-read-only", "runtime_client": "internal/sepay.Client", "http_endpoint": "https://userapi-sandbox.sepay.vn/v2/transactions", "validated_page": 1, "canonical_rows_validated": valid, "has_more": page.Pagination.More, "financial_writes": false, "paid_flow_acceptance": false})
}
