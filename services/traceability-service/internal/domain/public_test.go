package domain

import (
	"errors"
	"testing"
	"time"
)

func TestPublicEligibilityAndNoSyntheticDefaults(t *testing.T) {
	b, c, now := fixture()
	n := 1
	base := Profile{Binding: b, Mode: "sandbox", Status: "ACTIVE", EverPublished: true, PublicRevision: &n, Revisions: []Revision{{Number: 1, Status: "PUBLISHED", Content: c, DependencySnapshot: &DependencySnapshot{Binding: b, BestBefore: now.Add(time.Hour), BatchCode: "SYNTHETIC-LOT", ProductionDate: "2026-10-01", ExpiryDate: "2026-12-31", CheckedAt: now}}}}
	record, err := BuildPublic(base, now)
	if err != nil || record.ArtisanName != nil || record.FoodSafetyCert != nil || record.CraftingVideoURL != nil {
		t.Fatalf("public: %+v %v", record, err)
	}
	cases := map[string]struct {
		mutate func(*Profile)
		reason string
	}{
		"suspended":           {func(p *Profile) { p.Status = "SUSPENDED" }, "PROFILE_SUSPENDED"},
		"archived":            {func(p *Profile) { p.Status = "ARCHIVED" }, "PROFILE_ARCHIVED"},
		"expired batch":       {func(p *Profile) { p.Revisions[0].DependencySnapshot.BestBefore = now }, "BATCH_EXPIRED"},
		"restricted":          {func(p *Profile) { p.Revisions[0].DependencySnapshot.Restricted = true }, "BATCH_RESTRICTED"},
		"revoked certificate": {func(p *Profile) { p.Revisions[0].Content.Certificates[0].Revoked = true }, "CERTIFICATE_REVOKED"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p, err := base.Redact(true)
			if err != nil {
				t.Fatal(err)
			}
			tc.mutate(&p)
			var gone *GoneError
			if _, err = BuildPublic(p, now); !errors.As(err, &gone) || gone.Reason != tc.reason {
				t.Fatalf("got %v expected %s", err, tc.reason)
			}
		})
	}
	unpublished, _ := base.Redact(true)
	unpublished.EverPublished = false
	unpublished.Status = "SUSPENDED"
	if _, err = BuildPublic(unpublished, now); !errors.Is(err, ErrNotFound) {
		t.Fatal("unpublished status leaked")
	}
	depleted, _ := base.Redact(true)
	depleted.Revisions[0].DependencySnapshot.Depleted = true
	if _, err = BuildPublic(depleted, now); err != nil {
		t.Fatal("depleted blocked public trace", err)
	}
	for _, mutate := range []func(*Profile){func(p *Profile) { p.PublicRevision = nil }, func(p *Profile) { p.Revisions[0].DependencySnapshot = nil }, func(p *Profile) { p.Revisions[0].Status = "DRAFT" }, func(p *Profile) { p.Revisions[0].DependencySnapshot.Binding.BatchID = "other" }} {
		invalid, _ := base.Redact(true)
		mutate(&invalid)
		if _, err = BuildPublic(invalid, now); !errors.Is(err, ErrUnavailable) {
			t.Fatal("invalid stored state served publicly")
		}
	}
}

func TestSourceHashExcludesOnlyCheckTime(t *testing.T) {
	b, _, now := fixture()
	s := DependencySnapshot{Binding: b, BatchCode: "LOT", CatalogVersion: "1", BatchVersion: "1", CheckedAt: now}
	first, err := s.SourceHash()
	if err != nil {
		t.Fatal(err)
	}
	changed := s
	changed.CheckedAt = now.Add(time.Hour)
	hash, _ := changed.SourceHash()
	if first != hash {
		t.Fatal("verification timestamp invalidated content hash")
	}
	for _, mutate := range []func(*DependencySnapshot){func(s *DependencySnapshot) { s.BatchID = "other" }, func(s *DependencySnapshot) { s.BatchVersion = "2" }, func(s *DependencySnapshot) { s.BestBefore = now }, func(s *DependencySnapshot) { s.Restricted = true }, func(s *DependencySnapshot) { s.CatalogVersion = "2" }} {
		changed := s
		mutate(&changed)
		hash, _ := changed.SourceHash()
		if first == hash {
			t.Fatal("source correction did not invalidate hash")
		}
	}
}
