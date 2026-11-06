package domain

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

type Binding struct {
	ProductID       string `json:"product_id"`
	SKUCode         string `json:"sku_code"`
	SourceNamespace string `json:"source_namespace"`
	BatchID         string `json:"batch_id"`
}

type Content struct {
	ProductName      string        `json:"product_name"`
	Producer         string        `json:"producer"`
	WorkshopLocation string        `json:"workshop_location"`
	ArtisanName      *string       `json:"artisan_name"`
	Process          string        `json:"process"`
	Ingredients      []Ingredient  `json:"ingredients"`
	Certificates     []Certificate `json:"certificates"`
	Evidence         []Evidence    `json:"evidence"`
	CraftingVideoURL *string       `json:"crafting_video_url"`
}
type Ingredient struct {
	Name   string `json:"name"`
	Origin string `json:"origin"`
}

func (c Content) Normalize() Content {
	if c.Ingredients == nil {
		c.Ingredients = []Ingredient{}
	}
	if c.Certificates == nil {
		c.Certificates = []Certificate{}
	}
	if c.Evidence == nil {
		c.Evidence = []Evidence{}
	}
	return c
}

// Typed Go clients emit arrays rather than null for empty Draft collections.
func (c Content) MarshalJSON() ([]byte, error) {
	type wire Content
	return json.Marshal(wire(c.Normalize()))
}

type Evidence struct {
	ID          string `json:"id"`
	Reference   string `json:"reference,omitempty"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
}
type Certificate struct {
	ID                 string     `json:"id"`
	Type               string     `json:"type"`
	Number             string     `json:"number"`
	Issuer             string     `json:"issuer"`
	Scope              string     `json:"scope"`
	ScopeID            string     `json:"scope_id"`
	IssuedOn           string     `json:"issued_on"`
	ValidUntil         *string    `json:"valid_until"`
	NoExpiryStated     bool       `json:"no_expiry_stated"`
	ReviewDueAt        *time.Time `json:"review_due_at"`
	Stars              *int       `json:"stars"`
	VerificationMethod string     `json:"verification_method"`
	VerifiedAt         time.Time  `json:"verified_at"`
	Revoked            bool       `json:"revoked"`
	EvidenceIDs        []string   `json:"evidence_ids"`
}
type Revision struct {
	Number             int                 `json:"number"`
	Status             string              `json:"status"`
	Content            Content             `json:"content"`
	DependencySnapshot *DependencySnapshot `json:"dependency_snapshot,omitempty"`
	Contributors       []string            `json:"contributors"`
	CreatedBy          string              `json:"created_by"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}
type Profile struct {
	ID   string `json:"id"`
	Mode string `json:"mode"`
	Binding
	QRCode         string     `json:"qr_code"`
	QRURL          string     `json:"qr_url,omitempty"`
	Status         string     `json:"status"`
	Version        int64      `json:"version"`
	PublicRevision *int       `json:"public_revision"`
	EverPublished  bool       `json:"ever_published"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Revisions      []Revision `json:"revisions"`
}
type DependencySnapshot struct {
	Binding
	BatchCode      string    `json:"batch_code"`
	ProductionDate string    `json:"production_date"`
	ExpiryDate     string    `json:"expiry_date"`
	BestBefore     time.Time `json:"best_before"`
	Restricted     bool      `json:"restricted"`
	Depleted       bool      `json:"depleted"`
	CatalogVersion string    `json:"catalog_version"`
	BatchVersion   string    `json:"batch_version"`
	CheckedAt      time.Time `json:"checked_at"`
}

type ValidationError struct{ Field string }

func (e *ValidationError) Error() string { return "invalid field: " + e.Field }
func invalid(field string) error         { return &ValidationError{Field: field} }
func textWithin(s string, max int, required bool) bool {
	return utf8.ValidString(s) && utf8.RuneCountInString(s) <= max && (!required || strings.TrimSpace(s) != "") && !strings.ContainsRune(s, '\x00')
}

func (b Binding) Validate() error {
	if !ValidUUID(b.ProductID) {
		return invalid("product_id")
	}
	for _, v := range []struct {
		name, value string
		max         int
	}{{"sku_code", b.SKUCode, 100}, {"source_namespace", b.SourceNamespace, 100}, {"batch_id", b.BatchID, 160}} {
		if !textWithin(v.value, v.max, true) || strings.TrimSpace(v.value) != v.value {
			return invalid(v.name)
		}
	}
	return nil
}

func (c Content) ValidateDraft(videoHosts []string) error {
	for _, v := range []struct {
		name, value string
		max         int
	}{{"product_name", c.ProductName, 250}, {"producer", c.Producer, 250}, {"workshop_location", c.WorkshopLocation, 1000}, {"process", c.Process, 8000}} {
		if !textWithin(v.value, v.max, false) {
			return invalid(v.name)
		}
	}
	if c.ArtisanName != nil && !textWithin(*c.ArtisanName, 250, false) {
		return invalid("artisan_name")
	}
	if len(c.Ingredients) > 20 || len(c.Certificates) > 20 || len(c.Evidence) > 50 {
		return invalid("collection_limits")
	}
	for _, i := range c.Ingredients {
		if !textWithin(i.Name, 250, true) || !textWithin(i.Origin, 1000, true) {
			return invalid("ingredients")
		}
	}
	evidence := map[string]bool{}
	for _, e := range c.Evidence {
		id := strings.ToLower(e.ID)
		if !ValidUUID(id) || evidence[id] || !textWithin(e.Reference, 1000, true) || !textWithin(e.Kind, 100, true) || !textWithin(e.Description, 1000, true) {
			return invalid("evidence")
		}
		evidence[id] = true
	}
	certificates := map[string]bool{}
	for _, cert := range c.Certificates {
		id := strings.ToLower(cert.ID)
		if !ValidUUID(id) || certificates[id] {
			return invalid("certificate.id")
		}
		certificates[id] = true
		if err := cert.validateShape(evidence); err != nil {
			return err
		}
	}
	if c.CraftingVideoURL != nil {
		raw := *c.CraftingVideoURL
		u, err := url.Parse(raw)
		if err != nil || !textWithin(raw, 2048, true) || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || (u.Port() != "" && u.Port() != "443") {
			return invalid("crafting_video_url")
		}
		allowed := false
		for _, host := range videoHosts {
			if strings.EqualFold(host, u.Hostname()) {
				allowed = true
			}
		}
		if !allowed {
			return invalid("crafting_video_url")
		}
	}
	return nil
}

// Fixed UTC+7 is the certificate business calendar; timestamps remain instants.
var certificateZone = time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)

func certificateDate(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", value, certificateZone)
}

func (c Certificate) validateShape(evidence map[string]bool) error {
	if c.Type != "OCOP" && c.Type != "ATTP" && c.Type != "OTHER" {
		return invalid("certificate.type")
	}
	if !textWithin(c.Number, 250, true) || !textWithin(c.Issuer, 250, true) || !textWithin(c.ScopeID, 160, true) {
		return invalid("certificate.identity")
	}
	if c.Scope != "PRODUCT" && c.Scope != "SKU" && c.Scope != "BATCH" {
		return invalid("certificate.scope")
	}
	issued, err := certificateDate(c.IssuedOn)
	if err != nil {
		return invalid("certificate.issued_on")
	}
	if c.VerificationMethod != "MANUAL" || c.VerifiedAt.IsZero() {
		return invalid("certificate.manual_verification")
	}
	if c.NoExpiryStated {
		if c.ValidUntil != nil || c.ReviewDueAt == nil || c.ReviewDueAt.IsZero() || c.ReviewDueAt.Before(issued) {
			return invalid("certificate.review_due_at")
		}
	} else {
		if c.ValidUntil == nil || c.ReviewDueAt != nil {
			return invalid("certificate.valid_until")
		}
		end, err := certificateDate(*c.ValidUntil)
		if err != nil || end.Before(issued) {
			return invalid("certificate.valid_until")
		}
	}
	if c.Type == "OCOP" {
		if c.Stars == nil || *c.Stars < 1 || *c.Stars > 5 {
			return invalid("certificate.stars")
		}
	} else if c.Stars != nil {
		return invalid("certificate.stars")
	}
	if len(c.EvidenceIDs) < 1 || len(c.EvidenceIDs) > 50 {
		return invalid("certificate.evidence_ids")
	}
	seen := map[string]bool{}
	for _, ref := range c.EvidenceIDs {
		id := strings.ToLower(ref)
		if !evidence[id] || seen[id] {
			return invalid("certificate.evidence_ids")
		}
		seen[id] = true
	}
	return nil
}

// Eligibility returns a stable safe reason, never evidence or dependency details.
// It is also used on reads, independently of the expiry worker.
func (c Content) Eligibility(b Binding, now time.Time) string {
	ocop := false
	evidence := map[string]bool{}
	for _, e := range c.Evidence {
		evidence[strings.ToLower(e.ID)] = true
	}
	for _, cert := range c.Certificates {
		if err := cert.validateShape(evidence); err != nil {
			return "CERTIFICATE_NOT_VALID"
		}
		expected := ""
		switch cert.Scope {
		case "PRODUCT":
			expected = b.ProductID
		case "SKU":
			expected = b.SKUCode
		case "BATCH":
			expected = b.BatchID
		}
		match := cert.ScopeID == expected
		if cert.Scope == "PRODUCT" {
			match = strings.EqualFold(cert.ScopeID, expected)
		}
		if expected == "" || !match {
			return "CERTIFICATE_SCOPE_MISMATCH"
		}
		if cert.Revoked {
			return "CERTIFICATE_REVOKED"
		}
		issued, err := certificateDate(cert.IssuedOn)
		if err != nil || now.Before(issued) || cert.VerifiedAt.IsZero() || cert.VerifiedAt.After(now) {
			return "CERTIFICATE_NOT_VALID"
		}
		if cert.NoExpiryStated {
			if cert.ReviewDueAt == nil || !now.Before(*cert.ReviewDueAt) {
				return "CERTIFICATE_REVIEW_DUE"
			}
		} else {
			if cert.ValidUntil == nil {
				return "CERTIFICATE_NOT_VALID"
			}
			end, err := certificateDate(*cert.ValidUntil)
			if err != nil {
				return "CERTIFICATE_NOT_VALID"
			}
			if !now.Before(end.AddDate(0, 0, 1)) {
				return "CERTIFICATE_EXPIRED"
			}
		}
		if cert.Type == "OCOP" {
			ocop = true
		}
	}
	if !ocop {
		return "OCOP_CERTIFICATE_REQUIRED"
	}
	return ""
}

func (c Content) ValidateSubmission(b Binding, now time.Time, videoHosts []string) error {
	if err := b.Validate(); err != nil {
		return err
	}
	if err := c.ValidateDraft(videoHosts); err != nil {
		return err
	}
	if strings.TrimSpace(c.ProductName) == "" || strings.TrimSpace(c.Producer) == "" || strings.TrimSpace(c.WorkshopLocation) == "" || strings.TrimSpace(c.Process) == "" || len(c.Ingredients) == 0 {
		return invalid("required_content")
	}
	if reason := c.Eligibility(b, now); reason != "" {
		return invalid(reason)
	}
	return nil
}

// Redact returns a deep copy, preventing response redaction from changing history.
func (p Profile) Redact(evidenceAllowed bool) (Profile, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return Profile{}, err
	}
	var copy Profile
	if err = json.Unmarshal(data, &copy); err != nil {
		return Profile{}, err
	}
	if !evidenceAllowed {
		for i := range copy.Revisions {
			for j := range copy.Revisions[i].Content.Evidence {
				copy.Revisions[i].Content.Evidence[j].Reference = ""
			}
		}
	}
	return copy, nil
}

func (s DependencySnapshot) Validate(b Binding, now time.Time) error {
	if s.Binding != b {
		return ErrConflict
	}
	if s.CheckedAt.After(now) || now.Sub(s.CheckedAt) > 30*time.Second {
		return ErrUnavailable
	}
	if !textWithin(s.BatchCode, 160, true) || s.CatalogVersion == "" || s.BatchVersion == "" {
		return ErrUnavailable
	}
	produced, err := time.Parse("2006-01-02", s.ProductionDate)
	if err != nil {
		return ErrUnavailable
	}
	expires, err := time.Parse("2006-01-02", s.ExpiryDate)
	if err != nil || expires.Before(produced) || s.BestBefore.IsZero() {
		return ErrUnavailable
	}
	if s.Restricted || !now.Before(s.BestBefore) {
		return ErrConflict
	}
	return nil
}
