package application

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

type DraftStore interface {
	CreateDraft(context.Context, domain.CommandMetadata, domain.CreateDraftRequest) (domain.CommandResult, error)
	ReplayCreateDraft(context.Context, domain.CommandMetadata, domain.CreateDraftRequest) (domain.CommandResult, bool, error)
	EditDraft(context.Context, domain.CommandMetadata, string, int, domain.EditDraftRequest) (domain.CommandResult, error)
	Get(context.Context, string) (domain.Profile, error)
	List(context.Context, domain.ListQuery) (domain.ProfilePage, error)
}
type DependencyVerifier interface {
	Snapshot(context.Context, domain.Binding) (domain.DependencySnapshot, error)
}

type Drafts struct {
	Store        DraftStore
	Dependencies DependencyVerifier
	Clock        func() time.Time
	VideoHosts   []string
	PublicWebURL string
}

func authorize(p domain.Principal, permission string) error {
	if !domain.ValidUUID(p.ID) {
		return domain.ErrUnauthenticated
	}
	if !p.Can(permission) || !p.Can(domain.ReadInternal) {
		return domain.ErrForbidden
	}
	return nil
}

func (s *Drafts) Create(ctx context.Context, p domain.Principal, meta domain.CommandMetadata, request domain.CreateDraftRequest) (domain.CommandResult, error) {
	if err := authorize(p, domain.Create); err != nil {
		return domain.CommandResult{}, err
	}
	if s == nil || s.Store == nil || s.Dependencies == nil || s.Clock == nil {
		return domain.CommandResult{}, domain.ErrUnavailable
	}
	meta.PrincipalID = p.ID
	request.ProductID = strings.ToLower(request.ProductID)
	request.Content = request.Content.Normalize()
	if err := request.Binding.Validate(); err != nil {
		return domain.CommandResult{}, err
	}
	if err := request.Content.ValidateDraft(s.VideoHosts); err != nil {
		return domain.CommandResult{}, err
	}
	// Replay stays available when a peer is down, without replaying a side effect.
	result, found, err := s.Store.ReplayCreateDraft(ctx, meta, request)
	if err != nil {
		return domain.CommandResult{}, err
	}
	if found {
		return redactResult(result, p.Can(domain.ReadEvidence), s.PublicWebURL, meta.Scoped)
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	snapshot, err := s.Dependencies.Snapshot(checkCtx, request.Binding)
	if err != nil {
		return domain.CommandResult{}, err
	}
	if snapshot.Binding != request.Binding {
		return domain.CommandResult{}, domain.ErrConflict
	}
	if meta.Scoped {
		if !domain.ValidUUID(snapshot.WarehouseID) {
			return domain.CommandResult{}, domain.ErrUnavailable
		}
		if meta.WarehouseID != snapshot.WarehouseID {
			return domain.CommandResult{}, domain.ErrWarehouseConflict
		}
	}
	if domain.ValidUUID(snapshot.WarehouseID) {
		meta.WarehouseID = snapshot.WarehouseID
	}
	now := s.Clock()
	if snapshot.CheckedAt.After(now) || now.Sub(snapshot.CheckedAt) > 30*time.Second {
		return domain.CommandResult{}, domain.ErrUnavailable
	}
	result, err = s.Store.CreateDraft(ctx, meta, request)
	if err != nil {
		return domain.CommandResult{}, err
	}
	return redactResult(result, p.Can(domain.ReadEvidence), s.PublicWebURL, meta.Scoped)
}

func (s *Drafts) Edit(ctx context.Context, p domain.Principal, meta domain.CommandMetadata, id string, revision int, request domain.EditDraftRequest) (domain.CommandResult, error) {
	if err := authorize(p, domain.Edit); err != nil {
		return domain.CommandResult{}, err
	}
	if s == nil || s.Store == nil {
		return domain.CommandResult{}, domain.ErrUnavailable
	}
	meta.PrincipalID = p.ID
	if !meta.Scoped {
		if err := request.Content.ValidateDraft(s.VideoHosts); err != nil {
			return domain.CommandResult{}, err
		}
	}
	result, err := s.Store.EditDraft(ctx, meta, id, revision, request)
	if err != nil {
		return domain.CommandResult{}, err
	}
	return redactResult(result, p.Can(domain.ReadEvidence), s.PublicWebURL, meta.Scoped)
}

func (s *Drafts) Get(ctx context.Context, p domain.Principal, id string) (domain.Profile, error) {
	if err := authorize(p, domain.ReadInternal); err != nil {
		return domain.Profile{}, err
	}
	if s == nil || s.Store == nil {
		return domain.Profile{}, domain.ErrUnavailable
	}
	profile, err := s.Store.Get(ctx, id)
	if err != nil {
		return domain.Profile{}, err
	}
	profile, err = profile.Redact(p.Can(domain.ReadEvidence))
	if err != nil {
		return domain.Profile{}, err
	}
	linkProfile(&profile, s.PublicWebURL)
	profile.WarehouseID = nil
	return profile, nil
}

func redactResult(result domain.CommandResult, evidence bool, webBase string, scoped bool) (domain.CommandResult, error) {
	var profile domain.Profile
	if err := json.Unmarshal(result.Body, &profile); err != nil {
		return domain.CommandResult{}, err
	}
	redacted, err := profile.Redact(evidence)
	if err != nil {
		return domain.CommandResult{}, err
	}
	linkProfile(&redacted, webBase)
	if !scoped {
		redacted.WarehouseID = nil
	}
	result.Body, err = json.Marshal(redacted)
	return result, err
}

func (s *Drafts) List(ctx context.Context, p domain.Principal, query domain.ListQuery) (domain.ProfilePage, error) {
	if err := authorize(p, domain.ReadInternal); err != nil {
		return domain.ProfilePage{}, err
	}
	if s == nil || s.Store == nil {
		return domain.ProfilePage{}, domain.ErrUnavailable
	}
	page, err := s.Store.List(ctx, query)
	if err != nil {
		return domain.ProfilePage{}, err
	}
	for i := range page.Items {
		page.Items[i], err = page.Items[i].Redact(p.Can(domain.ReadEvidence))
		if err != nil {
			return domain.ProfilePage{}, err
		}
		linkProfile(&page.Items[i], s.PublicWebURL)
		page.Items[i].WarehouseID = nil
	}
	return page, nil
}

func linkProfile(profile *domain.Profile, base string) {
	if base == "" {
		return
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/trace/" + profile.QRCode
	parsed.RawPath = ""
	profile.QRURL = parsed.String()
}
