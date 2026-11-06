package application

import (
	"context"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
)

type WorkflowStore interface {
	Get(context.Context, string) (domain.Profile, error)
	NewRevision(context.Context, domain.CommandMetadata, string, domain.NewRevisionRequest) (domain.CommandResult, error)
	ReplayRevisionCommand(context.Context, domain.CommandMetadata, string, int, string, domain.RevisionCommand) (domain.CommandResult, bool, error)
	SubmitRevision(context.Context, domain.CommandMetadata, string, int, domain.RevisionCommand, domain.DependencySnapshot) (domain.CommandResult, error)
	DecideRevision(context.Context, domain.CommandMetadata, string, int, string, domain.RevisionCommand) (domain.CommandResult, error)
	PublishRevision(context.Context, domain.CommandMetadata, string, int, domain.RevisionCommand, domain.DependencySnapshot) (domain.CommandResult, error)
	SetStatus(context.Context, domain.CommandMetadata, string, string, domain.RevisionCommand) (domain.CommandResult, error)
	Public(context.Context, string) (domain.PublicRecord, error)
}
type Workflow struct {
	Store        WorkflowStore
	Dependencies DependencyVerifier
	Clock        func() time.Time
	VideoHosts   []string
	PublicWebURL string
}

func (s *Workflow) NewRevision(ctx context.Context, p domain.Principal, meta domain.CommandMetadata, id string, request domain.NewRevisionRequest) (domain.CommandResult, error) {
	if err := authorize(p, domain.Edit); err != nil {
		return domain.CommandResult{}, err
	}
	if s == nil || s.Store == nil {
		return domain.CommandResult{}, domain.ErrUnavailable
	}
	meta.PrincipalID = p.ID
	result, err := s.Store.NewRevision(ctx, meta, id, request)
	if err != nil {
		return domain.CommandResult{}, err
	}
	return redactResult(result, p.Can(domain.ReadEvidence), s.PublicWebURL)
}

func (s *Workflow) Submit(ctx context.Context, p domain.Principal, meta domain.CommandMetadata, id string, revision int, request domain.RevisionCommand) (domain.CommandResult, error) {
	if err := authorize(p, domain.Submit); err != nil {
		return domain.CommandResult{}, err
	}
	if s == nil || s.Store == nil || s.Dependencies == nil || s.Clock == nil {
		return domain.CommandResult{}, domain.ErrUnavailable
	}
	meta.PrincipalID = p.ID
	result, found, err := s.Store.ReplayRevisionCommand(ctx, meta, id, revision, domain.ActionSubmit, request)
	if err != nil {
		return domain.CommandResult{}, err
	}
	if found {
		return redactResult(result, p.Can(domain.ReadEvidence), s.PublicWebURL)
	}
	profile, err := s.Store.Get(ctx, id)
	if err != nil {
		return domain.CommandResult{}, err
	}
	if profile.Version != request.ExpectedVersion || profile.Status != "ACTIVE" {
		return domain.CommandResult{}, domain.ErrConflict
	}
	var current *domain.Revision
	for i := range profile.Revisions {
		if profile.Revisions[i].Number == revision {
			current = &profile.Revisions[i]
		}
	}
	if current == nil {
		return domain.CommandResult{}, domain.ErrNotFound
	}
	if current.Status != "DRAFT" {
		return domain.CommandResult{}, domain.ErrConflict
	}
	if err = current.Content.ValidateSubmission(profile.Binding, s.Clock(), s.VideoHosts); err != nil {
		return domain.CommandResult{}, err
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	snapshot, err := s.Dependencies.Snapshot(checkCtx, profile.Binding)
	if err != nil {
		return domain.CommandResult{}, err
	}
	if err = verifiedSnapshot(snapshot, profile.Binding, s.Clock()); err != nil {
		return domain.CommandResult{}, err
	}
	// Store re-locks and checks expected version, content and source freshness.
	// No network call takes place inside the store's local transaction.
	result, err = s.Store.SubmitRevision(ctx, meta, id, revision, request, snapshot)
	if err != nil {
		return domain.CommandResult{}, err
	}
	return redactResult(result, p.Can(domain.ReadEvidence), s.PublicWebURL)
}

func (s *Workflow) Decide(ctx context.Context, p domain.Principal, meta domain.CommandMetadata, id string, revision int, action string, request domain.RevisionCommand) (domain.CommandResult, error) {
	if err := authorize(p, domain.Approve); err != nil {
		return domain.CommandResult{}, err
	}
	if s == nil || s.Store == nil {
		return domain.CommandResult{}, domain.ErrUnavailable
	}
	if action != domain.ActionApprove && action != domain.ActionReject {
		return domain.CommandResult{}, domain.ErrInvalidCommand
	}
	meta.PrincipalID = p.ID
	result, err := s.Store.DecideRevision(ctx, meta, id, revision, action, request)
	if err != nil {
		return domain.CommandResult{}, err
	}
	return redactResult(result, p.Can(domain.ReadEvidence), s.PublicWebURL)
}

func verifiedSnapshot(snapshot domain.DependencySnapshot, binding domain.Binding, now time.Time) error {
	return snapshot.Validate(binding, now)
}

func (s *Workflow) Publish(ctx context.Context, p domain.Principal, meta domain.CommandMetadata, id string, revision int, request domain.RevisionCommand) (domain.CommandResult, error) {
	if err := authorize(p, domain.Publish); err != nil {
		return domain.CommandResult{}, err
	}
	if s == nil || s.Store == nil || s.Dependencies == nil || s.Clock == nil {
		return domain.CommandResult{}, domain.ErrUnavailable
	}
	meta.PrincipalID = p.ID
	result, found, err := s.Store.ReplayRevisionCommand(ctx, meta, id, revision, domain.ActionPublish, request)
	if err != nil {
		return domain.CommandResult{}, err
	}
	if found {
		return redactResult(result, p.Can(domain.ReadEvidence), s.PublicWebURL)
	}
	profile, err := s.Store.Get(ctx, id)
	if err != nil {
		return domain.CommandResult{}, err
	}
	if profile.Version != request.ExpectedVersion || profile.Status != "ACTIVE" {
		return domain.CommandResult{}, domain.ErrConflict
	}
	var current *domain.Revision
	for i := range profile.Revisions {
		if profile.Revisions[i].Number == revision {
			current = &profile.Revisions[i]
		}
	}
	if current == nil {
		return domain.CommandResult{}, domain.ErrNotFound
	}
	if current.Status != "APPROVED" {
		return domain.CommandResult{}, domain.ErrConflict
	}
	if err = current.Content.ValidateSubmission(profile.Binding, s.Clock(), s.VideoHosts); err != nil {
		return domain.CommandResult{}, err
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	snapshot, err := s.Dependencies.Snapshot(checkCtx, profile.Binding)
	if err != nil {
		return domain.CommandResult{}, err
	}
	if err = verifiedSnapshot(snapshot, profile.Binding, s.Clock()); err != nil {
		return domain.CommandResult{}, err
	}
	result, err = s.Store.PublishRevision(ctx, meta, id, revision, request, snapshot)
	if err != nil {
		return domain.CommandResult{}, err
	}
	return redactResult(result, p.Can(domain.ReadEvidence), s.PublicWebURL)
}

func (s *Workflow) SetStatus(ctx context.Context, p domain.Principal, meta domain.CommandMetadata, id, status string, request domain.RevisionCommand) (domain.CommandResult, error) {
	permission := domain.Suspend
	if status == "ARCHIVED" {
		permission = domain.Archive
	} else if status != "SUSPENDED" {
		return domain.CommandResult{}, domain.ErrInvalidCommand
	}
	if err := authorize(p, permission); err != nil {
		return domain.CommandResult{}, err
	}
	if s == nil || s.Store == nil {
		return domain.CommandResult{}, domain.ErrUnavailable
	}
	meta.PrincipalID = p.ID
	result, err := s.Store.SetStatus(ctx, meta, id, status, request)
	if err != nil {
		return domain.CommandResult{}, err
	}
	return redactResult(result, p.Can(domain.ReadEvidence), s.PublicWebURL)
}

func (s *Workflow) Public(ctx context.Context, code string) (domain.PublicRecord, error) {
	if s == nil || s.Store == nil {
		return domain.PublicRecord{}, domain.ErrUnavailable
	}
	return s.Store.Public(ctx, code)
}
