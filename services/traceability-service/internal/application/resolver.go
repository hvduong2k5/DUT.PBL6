package application

import (
	"context"
	"errors"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

type ResolutionStore interface {
	ResolveProfiles(context.Context, []domain.LinkBinding) ([]domain.ResolutionProfile, error)
}
type Resolver struct {
	Store        ResolutionStore
	Clock        func() time.Time
	PublicWebURL string
}

func (s *Resolver) Resolve(ctx context.Context, bindings []domain.LinkBinding) (domain.ResolvedLinks, error) {
	if len(bindings) < 1 || len(bindings) > 100 {
		return domain.ResolvedLinks{}, domain.ErrInvalidCommand
	}
	for _, b := range bindings {
		if !b.Valid() {
			return domain.ResolvedLinks{}, domain.ErrInvalidCommand
		}
	}
	if s == nil || s.Store == nil || s.Clock == nil {
		return domain.ResolvedLinks{}, domain.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	profiles, err := s.Store.ResolveProfiles(ctx, bindings)
	if err != nil || len(profiles) != len(bindings) {
		return domain.ResolvedLinks{}, domain.ErrUnavailable
	}
	result := domain.ResolvedLinks{Items: make([]domain.ResolvedLink, len(bindings)), Total: len(bindings)}
	now := s.Clock()
	for i, row := range profiles {
		entry := domain.ResolvedLink{LinkBinding: bindings[i], Status: "NOT_AVAILABLE"}
		switch {
		case row.Unavailable:
			entry.Status = "UNAVAILABLE"
		case row.Profile != nil:
			profile := *row.Profile
			if profile.SourceNamespace != bindings[i].SourceNamespace || profile.BatchID != bindings[i].BatchID || profile.SKUCode != bindings[i].SKUCode {
				break
			}
			_, err := domain.BuildPublic(profile, now)
			var gone *domain.GoneError
			switch {
			case err == nil:
				entry.Status = "AVAILABLE"
			case errors.As(err, &gone) && profile.EverPublished:
				entry.Status = "NO_LONGER_PUBLIC"
			case errors.Is(err, domain.ErrNotFound):
			default:
				entry.Status = "UNAVAILABLE"
			}
			if entry.Status == "AVAILABLE" || entry.Status == "NO_LONGER_PUBLIC" {
				linkProfile(&profile, s.PublicWebURL)
				if !domain.ValidQRCode(profile.QRCode) || profile.QRURL == "" {
					entry.Status = "UNAVAILABLE"
				} else {
					entry.QRURL = &profile.QRURL
				}
			}
		}
		result.Items[i] = entry
	}
	return result, nil
}
