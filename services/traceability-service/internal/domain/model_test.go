package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

const productID = "44444444-4444-4444-8444-444444444444"
const evidenceID = "55555555-5555-4555-8555-555555555555"

func fixture() (Binding, Content, time.Time) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	until := "2026-10-17"
	stars := 4
	return Binding{ProductID: productID, SKUCode: "SKU-1", SourceNamespace: "fixture", BatchID: "batch-1"}, Content{
		ProductName: "Sản phẩm mẫu", Producer: "Xưởng mẫu", WorkshopLocation: "Huế", Process: "Quy trình mẫu",
		Ingredients:  []Ingredient{{Name: "Nguyên liệu mẫu", Origin: "Huế"}},
		Evidence:     []Evidence{{ID: evidenceID, Reference: "private/document-1", Kind: "CERTIFICATE", Description: "Bằng chứng mẫu"}},
		Certificates: []Certificate{{ID: "66666666-6666-4666-8666-666666666666", Type: "OCOP", Number: "OCOP-1", Issuer: "Synthetic authority", Scope: "PRODUCT", ScopeID: productID, IssuedOn: "2026-01-01", ValidUntil: &until, Stars: &stars, VerificationMethod: "MANUAL", VerifiedAt: now.Add(-time.Hour), EvidenceIDs: []string{evidenceID}}},
	}, now
}

func TestDraftAndSubmission(t *testing.T) {
	b, c, now := fixture()
	if err := (Content{}).ValidateDraft(nil); err != nil {
		t.Fatal(err)
	}
	if err := (Content{}).ValidateSubmission(b, now, nil); err == nil {
		t.Fatal("incomplete draft submitted")
	}
	if err := c.ValidateSubmission(b, now, nil); err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*Content){
		"name limit":                   func(c *Content) { c.ProductName = strings.Repeat("ế", 251) },
		"null byte":                    func(c *Content) { c.Producer = "bad\x00name" },
		"ingredient missing origin":    func(c *Content) { c.Ingredients[0].Origin = "" },
		"duplicate evidence":           func(c *Content) { c.Evidence = append(c.Evidence, c.Evidence[0]) },
		"duplicate certificate":        func(c *Content) { c.Certificates = append(c.Certificates, c.Certificates[0]) },
		"unknown evidence":             func(c *Content) { c.Certificates[0].EvidenceIDs = []string{productID} },
		"duplicate evidence reference": func(c *Content) { c.Certificates[0].EvidenceIDs = append(c.Certificates[0].EvidenceIDs, evidenceID) },
		"invalid type":                 func(c *Content) { c.Certificates[0].Type = "EXTERNAL" },
		"invalid scope":                func(c *Content) { c.Certificates[0].Scope = "SUPPLIER" },
		"external method":              func(c *Content) { c.Certificates[0].VerificationMethod = "EXT10" },
		"missing stars":                func(c *Content) { c.Certificates[0].Stars = nil },
		"bad date":                     func(c *Content) { c.Certificates[0].IssuedOn = "2026-02-30" },
		"missing expiry":               func(c *Content) { c.Certificates[0].ValidUntil = nil },
		"expiry before issue":          func(c *Content) { s := "2025-12-31"; c.Certificates[0].ValidUntil = &s },
		"no expiry missing review":     func(c *Content) { c.Certificates[0].NoExpiryStated = true; c.Certificates[0].ValidUntil = nil },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			_, content, _ := fixture()
			mutate(&content)
			if err := content.ValidateDraft(nil); err == nil {
				t.Fatal("invalid draft accepted")
			}
		})
	}
}

func TestCertificateEligibilityBoundaries(t *testing.T) {
	b, c, now := fixture()
	midnight := time.Date(2026, 10, 17, 17, 0, 0, 0, time.UTC)
	if got := c.Eligibility(b, midnight.Add(-time.Nanosecond)); got != "" {
		t.Fatal(got)
	}
	if got := c.Eligibility(b, midnight); got != "CERTIFICATE_EXPIRED" {
		t.Fatal(got)
	}
	tests := map[string]struct {
		mutate func(*Content)
		reason string
	}{
		"revoked":             {func(c *Content) { c.Certificates[0].Revoked = true }, "CERTIFICATE_REVOKED"},
		"wrong product":       {func(c *Content) { c.Certificates[0].ScopeID = evidenceID }, "CERTIFICATE_SCOPE_MISMATCH"},
		"future issue":        {func(c *Content) { c.Certificates[0].IssuedOn = "2026-10-18" }, "CERTIFICATE_NOT_VALID"},
		"future verification": {func(c *Content) { c.Certificates[0].VerifiedAt = now.Add(time.Second) }, "CERTIFICATE_NOT_VALID"},
		"no OCOP":             {func(c *Content) { c.Certificates = nil }, "OCOP_CERTIFICATE_REQUIRED"},
		"invalid method":      {func(c *Content) { c.Certificates[0].VerificationMethod = "EXT10" }, "CERTIFICATE_NOT_VALID"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, c, _ := fixture()
			tc.mutate(&c)
			if got := c.Eligibility(b, now); got != tc.reason {
				t.Fatalf("got %s want %s", got, tc.reason)
			}
		})
	}
	c.Certificates[0].ValidUntil = nil
	c.Certificates[0].NoExpiryStated = true
	c.Certificates[0].ReviewDueAt = &now
	if got := c.Eligibility(b, now.Add(-time.Nanosecond)); got != "" {
		t.Fatal(got)
	}
	if got := c.Eligibility(b, now); got != "CERTIFICATE_REVIEW_DUE" {
		t.Fatal(got)
	}
}

func TestVideoAllowlistAndRedaction(t *testing.T) {
	for _, raw := range []string{"http://video.example/v", "https://video.example.attacker.test/v", "https://user:pass@video.example/v", "https://video.example:8443/v", "https://video.example/v#fragment"} {
		_, c, _ := fixture()
		c.CraftingVideoURL = &raw
		if err := c.ValidateDraft([]string{"video.example"}); err == nil {
			t.Errorf("unsafe URL accepted: %s", raw)
		}
	}
	_, c, _ := fixture()
	raw := "https://video.example/v"
	c.CraftingVideoURL = &raw
	if err := c.ValidateDraft([]string{"video.example"}); err != nil {
		t.Fatal(err)
	}
	if err := c.ValidateDraft(nil); err == nil {
		t.Fatal("implicit video allowlist")
	}
	p := Profile{Revisions: []Revision{{Content: c}}}
	redacted, err := p.Redact(false)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(redacted)
	if strings.Contains(string(body), "private/document-1") || p.Revisions[0].Content.Evidence[0].Reference != "private/document-1" {
		t.Fatal("redaction leaked or changed original")
	}
	allowed, err := p.Redact(true)
	if err != nil || allowed.Revisions[0].Content.Evidence[0].Reference != "private/document-1" {
		t.Fatal("authorized evidence missing")
	}
}

func TestBindingAndSnapshot(t *testing.T) {
	b, _, now := fixture()
	if err := b.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "not-uuid", "00000000-0000-0000-0000-000000000000"} {
		bad := b
		bad.ProductID = id
		if err := bad.Validate(); err == nil {
			t.Fatal("invalid product")
		}
	}
	for _, field := range []string{"sku", "namespace", "batch"} {
		bad := b
		switch field {
		case "sku":
			bad.SKUCode = " "
		case "namespace":
			bad.SourceNamespace = ""
		case "batch":
			bad.BatchID = "batch "
		}
		if err := bad.Validate(); err == nil {
			t.Fatal("invalid binding")
		}
	}
	snapshot := DependencySnapshot{Binding: b, BatchCode: "LOT-1", ProductionDate: "2026-10-01", ExpiryDate: "2027-01-01", BestBefore: now.Add(time.Hour), CatalogVersion: "1", BatchVersion: "1", CheckedAt: now}
	if err := snapshot.Validate(b, now); err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []func(*DependencySnapshot){func(s *DependencySnapshot) { s.CheckedAt = now.Add(-31 * time.Second) }, func(s *DependencySnapshot) { s.CheckedAt = now.Add(time.Second) }, func(s *DependencySnapshot) { s.BatchID = "other" }, func(s *DependencySnapshot) { s.BatchVersion = "" }, func(s *DependencySnapshot) { s.ExpiryDate = "bad" }, func(s *DependencySnapshot) { s.Restricted = true }, func(s *DependencySnapshot) { s.BestBefore = now }} {
		bad := snapshot
		mutation(&bad)
		if err := bad.Validate(b, now); err == nil {
			t.Fatal("invalid snapshot accepted")
		}
	}
	snapshot.Depleted = true
	if err := snapshot.Validate(b, now); err != nil {
		t.Fatal("depleted batch wrongly blocked")
	}
}
