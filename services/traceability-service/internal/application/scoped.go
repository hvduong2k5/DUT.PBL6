package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

type ScopedSources interface {
	Batch(context.Context, string, string, string) (domain.DependencySnapshot, error)
	Catalog(context.Context, string) (string, string, error)
}
type ScopedStore interface {
	Get(context.Context, string) (domain.Profile, error)
	FindBatch(context.Context, string, string) (domain.Profile, error)
	List(context.Context, domain.ListQuery) (domain.ProfilePage, error)
}
type Scoped struct {
	Store     ScopedStore
	Sources   ScopedSources
	Drafts    *Drafts
	Workflow  *Workflow
	Namespace string
}

func scopedPermission(p domain.Principal, permission string) error {
	if !domain.ValidUUID(p.ID) {
		return domain.ErrUnauthenticated
	}
	if !p.Has(domain.ReadInternal) || !p.Has(permission) {
		return domain.ErrForbidden
	}
	return nil
}

func scopedPrincipal(p domain.Principal, warehouse, permission string) (domain.Principal, error) {
	resource := p.At(warehouse)
	if err := authorize(resource, permission); err != nil {
		return domain.Principal{}, err
	}
	return resource, nil
}

func (s *Scoped) resource(ctx context.Context, p domain.Principal, id, permission string) (domain.Profile, domain.Principal, error) {
	if err := scopedPermission(p, permission); err != nil {
		return domain.Profile{}, domain.Principal{}, err
	}
	if !domain.ValidUUID(id) || id != strings.ToLower(id) {
		return domain.Profile{}, domain.Principal{}, domain.ErrInvalidCommand
	}
	if s == nil || s.Store == nil {
		return domain.Profile{}, domain.Principal{}, domain.ErrUnavailable
	}
	profile, err := s.Store.Get(ctx, id)
	if err != nil {
		return profile, domain.Principal{}, scopedError(err)
	}
	actor, err := scopedPrincipal(p, domain.Warehouse(profile), permission)
	return profile, actor, err
}

func (s *Scoped) Get(ctx context.Context, p domain.Principal, id string) (domain.Profile, error) {
	profile, actor, err := s.resource(ctx, p, id, domain.ReadInternal)
	if err != nil {
		return domain.Profile{}, err
	}
	profile, err = profile.Redact(actor.Can(domain.ReadEvidence))
	if err != nil {
		return domain.Profile{}, err
	}
	linkProfile(&profile, s.Drafts.PublicWebURL)
	return profile, nil
}

func (s *Scoped) List(ctx context.Context, p domain.Principal, q domain.ListQuery) (domain.ProfilePage, error) {
	if err := scopedPermission(p, domain.ReadInternal); err != nil {
		return domain.ProfilePage{}, err
	}
	q.Scoped = true
	q.Global, q.Warehouses = p.Warehouses()
	q.ActorID = p.ID
	q.AuthorizationDigest = p.AuthorizationDigest()
	page, err := s.Store.List(ctx, q)
	if err != nil {
		return domain.ProfilePage{}, scopedError(err)
	}
	for i := range page.Items {
		actor, err := scopedPrincipal(p, domain.Warehouse(page.Items[i]), domain.ReadInternal)
		if err != nil {
			return domain.ProfilePage{}, err
		}
		page.Items[i], err = page.Items[i].Redact(actor.Can(domain.ReadEvidence))
		if err != nil {
			return domain.ProfilePage{}, err
		}
		linkProfile(&page.Items[i], s.Drafts.PublicWebURL)
	}
	return page, nil
}

func (s *Scoped) batch(ctx context.Context, p domain.Principal, namespace, batch, sku, permission string) (domain.DependencySnapshot, domain.Principal, error) {
	if err := scopedPermission(p, permission); err != nil {
		return domain.DependencySnapshot{}, domain.Principal{}, err
	}
	if namespace != s.Namespace || !domain.ValidUUID(batch) || batch != strings.ToLower(batch) || sku == "" || strings.TrimSpace(sku) != sku || len([]rune(sku)) > 100 {
		return domain.DependencySnapshot{}, domain.Principal{}, domain.ErrInvalidCommand
	}
	if s.Sources == nil {
		return domain.DependencySnapshot{}, domain.Principal{}, domain.ErrUnavailable
	}
	snapshot, err := s.Sources.Batch(ctx, namespace, batch, sku)
	if err != nil {
		return snapshot, domain.Principal{}, err
	}
	if snapshot.SourceNamespace != namespace || snapshot.BatchID != batch || snapshot.SKUCode != sku {
		return snapshot, domain.Principal{}, domain.ErrConflict
	}
	if !domain.ValidUUID(snapshot.WarehouseID) {
		return snapshot, domain.Principal{}, domain.ErrUnavailable
	}
	actor, err := scopedPrincipal(p, snapshot.WarehouseID, permission)
	return snapshot, actor, err
}

func (s *Scoped) Context(ctx context.Context, p domain.Principal, batch, sku string) (domain.BatchContext, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	snapshot, _, err := s.batch(ctx, p, s.Namespace, batch, sku, domain.ReadInternal)
	if err != nil {
		return domain.BatchContext{}, err
	}
	product, version, err := s.Sources.Catalog(ctx, sku)
	if err != nil {
		return domain.BatchContext{}, err
	}
	if !domain.ValidUUID(product) || product != strings.ToLower(product) || version == "" {
		return domain.BatchContext{}, domain.ErrUnavailable
	}
	snapshot.ProductID = product
	snapshot.CatalogVersion = version
	return domain.ContextFrom(snapshot), nil
}

func (s *Scoped) Lookup(ctx context.Context, p domain.Principal, b domain.Binding) ([]domain.Profile, error) {
	if b.Validate() != nil || b.ProductID != strings.ToLower(b.ProductID) {
		return nil, domain.ErrInvalidCommand
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	snapshot, _, err := s.batch(ctx, p, b.SourceNamespace, b.BatchID, b.SKUCode, domain.ReadInternal)
	if err != nil {
		return nil, scopedError(err)
	}
	profile, err := s.Store.FindBatch(ctx, b.SourceNamespace, b.BatchID)
	if errors.Is(err, domain.ErrNotFound) {
		return []domain.Profile{}, nil
	}
	if err != nil {
		return nil, scopedError(err)
	}
	if _, err = scopedPrincipal(p, domain.Warehouse(profile), domain.ReadInternal); err != nil {
		return nil, err
	}
	if profile.WarehouseID != nil {
		if err = domain.CheckWarehouse(profile, snapshot); err != nil {
			return nil, err
		}
	}
	if profile.Binding != b {
		return nil, domain.ErrConflict
	}
	return []domain.Profile{profile}, nil
}

func (s *Scoped) Create(ctx context.Context, p domain.Principal, meta domain.CommandMetadata, r domain.CreateDraftRequest) (domain.CommandResult, error) {
	if err := scopedPermission(p, domain.Create); err != nil {
		return domain.CommandResult{}, err
	}
	if r.Binding.Validate() != nil || r.ProductID != strings.ToLower(r.ProductID) || r.SourceNamespace != s.Namespace || !domain.ValidUUID(r.BatchID) || r.BatchID != strings.ToLower(r.BatchID) {
		return domain.CommandResult{}, domain.ErrInvalidCommand
	}
	meta.Scoped = true
	profile, err := s.Store.FindBatch(ctx, r.SourceNamespace, r.BatchID)
	var actor domain.Principal
	if err == nil {
		actor, err = scopedPrincipal(p, domain.Warehouse(profile), domain.Create)
		if err != nil {
			return domain.CommandResult{}, err
		}
		if profile.Binding != r.Binding {
			return domain.CommandResult{}, domain.ErrConflict
		}
		meta.WarehouseID = domain.Warehouse(profile)
	} else if errors.Is(err, domain.ErrNotFound) {
		var snapshot domain.DependencySnapshot
		ctxWithBudget, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		ctx = ctxWithBudget
		snapshot, actor, err = s.batch(ctx, p, r.SourceNamespace, r.BatchID, r.SKUCode, domain.Create)
		if err != nil {
			return domain.CommandResult{}, err
		}
		meta.WarehouseID = snapshot.WarehouseID
	} else {
		return domain.CommandResult{}, scopedError(err)
	}
	return scopedResult(s.Drafts.Create(ctx, actor, meta, r))
}

func (s *Scoped) Edit(ctx context.Context, p domain.Principal, meta domain.CommandMetadata, id string, revision int, r domain.EditDraftRequest) (domain.CommandResult, error) {
	_, actor, err := s.resource(ctx, p, id, domain.Edit)
	if err != nil {
		return domain.CommandResult{}, err
	}
	meta.Scoped = true
	return scopedResult(s.Drafts.Edit(ctx, actor, meta, id, revision, r))
}
func (s *Scoped) NewRevision(ctx context.Context, p domain.Principal, meta domain.CommandMetadata, id string, r domain.NewRevisionRequest) (domain.CommandResult, error) {
	_, actor, err := s.resource(ctx, p, id, domain.Edit)
	if err != nil {
		return domain.CommandResult{}, err
	}
	meta.Scoped = true
	return scopedResult(s.Workflow.NewRevision(ctx, actor, meta, id, r))
}
func (s *Scoped) Command(ctx context.Context, p domain.Principal, meta domain.CommandMetadata, id string, revision int, action string, r domain.RevisionCommand) (domain.CommandResult, error) {
	permission := map[string]string{domain.ActionSubmit: domain.Submit, domain.ActionPublish: domain.Publish, domain.ActionApprove: domain.Approve, domain.ActionReject: domain.Approve, "profile.suspend": domain.Suspend, "profile.archive": domain.Archive}[action]
	if permission == "" {
		return domain.CommandResult{}, domain.ErrInvalidCommand
	}
	_, actor, err := s.resource(ctx, p, id, permission)
	if err != nil {
		return domain.CommandResult{}, err
	}
	meta.Scoped = true
	switch action {
	case domain.ActionSubmit:
		return scopedResult(s.Workflow.Submit(ctx, actor, meta, id, revision, r))
	case domain.ActionPublish:
		return scopedResult(s.Workflow.Publish(ctx, actor, meta, id, revision, r))
	case domain.ActionApprove, domain.ActionReject:
		return scopedResult(s.Workflow.Decide(ctx, actor, meta, id, revision, action, r))
	case "profile.suspend":
		return scopedResult(s.Workflow.SetStatus(ctx, actor, meta, id, "SUSPENDED", r))
	default:
		return scopedResult(s.Workflow.SetStatus(ctx, actor, meta, id, "ARCHIVED", r))
	}
}

func scopedResult(result domain.CommandResult, err error) (domain.CommandResult, error) {
	return result, scopedError(err)
}
func scopedError(err error) error {
	if err == nil {
		return nil
	}
	var validation *domain.ValidationError
	if errors.As(err, &validation) {
		return err
	}
	for _, known := range []error{domain.ErrUnauthenticated, domain.ErrForbidden, domain.ErrNotFound, domain.ErrConflict, domain.ErrWarehouseConflict, domain.ErrIdempotencyConflict, domain.ErrInvalidCommand, domain.ErrInvalidCursor, domain.ErrUnavailable} {
		if errors.Is(err, known) {
			return err
		}
	}
	return domain.ErrUnavailable
}
