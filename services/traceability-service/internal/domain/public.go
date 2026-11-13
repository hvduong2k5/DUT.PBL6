package domain

import (
	"regexp"
	"strings"
	"time"
)

type GoneError struct{ Reason string }

func (e *GoneError) Error() string { return e.Reason }

var qrPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{22,160}$`)

func ValidQRCode(code string) bool { return qrPattern.MatchString(code) }

// PublicRecord is an explicit allowlist; never marshal Profile on this route.
type PublicRecord struct {
	QRCode             string    `json:"qr_code"`
	ProductName        string    `json:"product_name"`
	BatchCode          string    `json:"batch_code"`
	Scope              string    `json:"scope"`
	Revision           int       `json:"revision"`
	Mode               string    `json:"mode"`
	VerificationMethod string    `json:"verification_method"`
	VerifiedAt         time.Time `json:"verified_at"`
	SourceVerifiedAt   time.Time `json:"source_verified_at"`
	OCOPStar           int       `json:"ocop_star"`
	OCOPCertificateNo  string    `json:"ocop_certificate_no"`
	ProductionDate     string    `json:"production_date"`
	ExpiryDate         string    `json:"expiry_date"`
	Producer           string    `json:"producer"`
	WorkshopLocation   string    `json:"workshop_location"`
	ArtisanName        *string   `json:"artisan_name"`
	FoodSafetyCert     *string   `json:"food_safety_cert"`
	RawMaterialsOrigin string    `json:"raw_materials_origin"`
	Process            string    `json:"process"`
	CraftingVideoURL   *string   `json:"crafting_video_url"`
}

func BuildPublic(p Profile, now time.Time) (PublicRecord, error) {
	fail := func(reason string) (PublicRecord, error) { return PublicRecord{}, &GoneError{Reason: reason} }
	if !p.EverPublished {
		return PublicRecord{}, ErrNotFound
	}
	switch p.Status {
	case "SUSPENDED":
		return fail("PROFILE_SUSPENDED")
	case "ARCHIVED":
		return fail("PROFILE_ARCHIVED")
	case "ACTIVE":
	default:
		return PublicRecord{}, ErrUnavailable
	}
	if p.PublicRevision == nil {
		return PublicRecord{}, ErrUnavailable
	}
	var published *Revision
	for i := range p.Revisions {
		if p.Revisions[i].Number == *p.PublicRevision {
			published = &p.Revisions[i]
		}
	}
	if published == nil || published.Status != "PUBLISHED" || published.DependencySnapshot == nil {
		return PublicRecord{}, ErrUnavailable
	}
	snapshot := published.DependencySnapshot
	c := published.Content
	if snapshot.Binding != p.Binding || snapshot.BestBefore.IsZero() {
		return PublicRecord{}, ErrUnavailable
	}
	if snapshot.Restricted {
		return fail("BATCH_RESTRICTED")
	}
	if !now.Before(snapshot.BestBefore) {
		return fail("BATCH_EXPIRED")
	}
	if reason := c.Eligibility(p.Binding, now); reason != "" {
		return fail(reason)
	}
	var ocop *Certificate
	var attp *string
	for i := range c.Certificates {
		cert := &c.Certificates[i]
		if cert.Type == "OCOP" && ocop == nil {
			ocop = cert
		}
		if cert.Type == "ATTP" && attp == nil {
			number := cert.Number
			attp = &number
		}
	}
	if ocop == nil || ocop.Stars == nil {
		return PublicRecord{}, ErrUnavailable
	}
	origins := make([]string, 0, len(c.Ingredients))
	for _, material := range c.Ingredients {
		origins = append(origins, material.Name+": "+material.Origin)
	}
	artisan := c.ArtisanName
	if artisan != nil && strings.TrimSpace(*artisan) == "" {
		artisan = nil
	}
	return PublicRecord{QRCode: p.QRCode, ProductName: c.ProductName, BatchCode: snapshot.BatchCode, Scope: "BATCH", Revision: published.Number, Mode: p.Mode, VerificationMethod: "MANUAL", VerifiedAt: ocop.VerifiedAt, SourceVerifiedAt: snapshot.CheckedAt, OCOPStar: *ocop.Stars, OCOPCertificateNo: ocop.Number, ProductionDate: snapshot.ProductionDate, ExpiryDate: snapshot.ExpiryDate, Producer: c.Producer, WorkshopLocation: c.WorkshopLocation, ArtisanName: artisan, FoodSafetyCert: attp, RawMaterialsOrigin: strings.Join(origins, "; "), Process: c.Process, CraftingVideoURL: c.CraftingVideoURL}, nil
}
