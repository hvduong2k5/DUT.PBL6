package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

func revisionCommand(meta domain.CommandMetadata, id string, revision int, action string, request domain.RevisionCommand) (Command, error) {
	if !domain.ValidUUID(id) || revision < 1 || request.ExpectedVersion < 1 {
		return Command{}, ErrInvalidCommand
	}
	switch action {
	case domain.ActionSubmit, domain.ActionApprove, domain.ActionReject, domain.ActionPublish:
	default:
		return Command{}, ErrInvalidCommand
	}
	if err := domain.ValidateReason(request.Reason, action == domain.ActionApprove || action == domain.ActionReject); err != nil {
		return Command{}, err
	}
	return draftCommand(meta, action, struct {
		ID       string
		Revision int
	}{strings.ToLower(id), revision}, request)
}

func (s *Profiles) ReplayRevisionCommand(ctx context.Context, meta domain.CommandMetadata, id string, revision int, action string, request domain.RevisionCommand) (domain.CommandResult, bool, error) {
	cmd, err := revisionCommand(meta, id, revision, action, request)
	if err != nil {
		return domain.CommandResult{}, false, err
	}
	result, found, err := (Commands{DB: s.DB}).Lookup(ctx, cmd)
	return commandResult(result), found, err
}

func lockProfile(ctx context.Context, tx pgx.Tx, id string, expected int64) (domain.Profile, error) {
	var version int64
	var state string
	err := tx.QueryRow(ctx, `SELECT version,status FROM traceability_profiles WHERE id=$1 FOR UPDATE`, id).Scan(&version, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Profile{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Profile{}, err
	}
	if version != expected || state != "ACTIVE" {
		return domain.Profile{}, domain.ErrConflict
	}
	return readProfile(ctx, tx, id)
}
func findRevision(p *domain.Profile, number int) (*domain.Revision, error) {
	for i := range p.Revisions {
		if p.Revisions[i].Number == number {
			return &p.Revisions[i], nil
		}
	}
	return nil, domain.ErrNotFound
}

func (s *Profiles) NewRevision(ctx context.Context, meta domain.CommandMetadata, id string, request domain.NewRevisionRequest) (domain.CommandResult, error) {
	if !domain.ValidUUID(id) || request.ExpectedVersion < 1 || request.FromRevision < 1 {
		return domain.CommandResult{}, ErrInvalidCommand
	}
	if err := domain.ValidateReason(request.Reason, false); err != nil {
		return domain.CommandResult{}, err
	}
	cmd, err := draftCommand(meta, "revision.create", strings.ToLower(id), request)
	if err != nil {
		return domain.CommandResult{}, err
	}
	result, err := (Commands{DB: s.DB}).Run(ctx, cmd, func(ctx context.Context, tx pgx.Tx) (Outcome, error) {
		p, err := lockProfile(ctx, tx, id, request.ExpectedVersion)
		if err != nil {
			return Outcome{}, err
		}
		source, err := findRevision(&p, request.FromRevision)
		if err != nil {
			return Outcome{}, err
		}
		contributors := map[string]bool{}
		for _, actor := range source.Contributors {
			contributors[strings.ToLower(actor)] = true
		}
		next := 1
		withdrawn := []int{}
		now := s.Clock().UTC()
		for _, r := range p.Revisions {
			if r.Number >= next {
				next = r.Number + 1
			}
			if r.Status == "DRAFT" || r.Status == "IN_REVIEW" || r.Status == "APPROVED" {
				withdrawn = append(withdrawn, r.Number)
				if err = domain.ValidateReason(request.Reason, true); err != nil {
					return Outcome{}, err
				}
				for _, actor := range r.Contributors {
					contributors[strings.ToLower(actor)] = true
				}
				if _, err = tx.Exec(ctx, `UPDATE traceability_revisions SET status='ARCHIVED',updated_at=$3 WHERE profile_id=$1 AND number=$2`, id, r.Number, now); err != nil {
					return Outcome{}, err
				}
			}
		}
		contributors[strings.ToLower(meta.PrincipalID)] = true
		actors := make([]string, 0, len(contributors))
		for actor := range contributors {
			actors = append(actors, actor)
		}
		sort.Strings(actors)
		content, err := json.Marshal(source.Content)
		if err != nil {
			return Outcome{}, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO traceability_revisions(profile_id,number,content,contributors,created_by,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$6)`, id, next, content, actors, meta.PrincipalID, now)
		if err != nil {
			return Outcome{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE traceability_profiles SET version=version+1,updated_at=$2 WHERE id=$1`, id, now); err != nil {
			return Outcome{}, err
		}
		before := map[string]any{"version": p.Version, "public_revision": p.PublicRevision, "working_revisions": withdrawn}
		after := map[string]any{"version": p.Version + 1, "public_revision": p.PublicRevision, "revision": next, "revision_status": "DRAFT", "source_revision": request.FromRevision, "withdrawn_revisions": withdrawn}
		if err = s.auditStates(ctx, tx, meta, id, next, "revision.create", before, after, now, request.Reason); err != nil {
			return Outcome{}, err
		}
		return profileOutcome(ctx, tx, id, 201)
	})
	return commandResult(result), err
}

func (s *Profiles) SubmitRevision(ctx context.Context, meta domain.CommandMetadata, id string, revision int, request domain.RevisionCommand, snapshot domain.DependencySnapshot) (domain.CommandResult, error) {
	cmd, err := revisionCommand(meta, id, revision, domain.ActionSubmit, request)
	if err != nil {
		return domain.CommandResult{}, err
	}
	result, err := (Commands{DB: s.DB}).Run(ctx, cmd, func(ctx context.Context, tx pgx.Tx) (Outcome, error) {
		p, err := lockProfile(ctx, tx, id, request.ExpectedVersion)
		if err != nil {
			return Outcome{}, err
		}
		r, err := findRevision(&p, revision)
		if err != nil {
			return Outcome{}, err
		}
		if r.Status != "DRAFT" {
			return Outcome{}, domain.ErrConflict
		}
		now := s.Clock().UTC()
		if err = r.Content.ValidateSubmission(p.Binding, now, s.VideoHosts); err != nil {
			return Outcome{}, err
		}
		if err = snapshot.Validate(p.Binding, now); err != nil {
			return Outcome{}, err
		}
		encoded, err := json.Marshal(snapshot)
		if err != nil {
			return Outcome{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE traceability_revisions SET status='IN_REVIEW',dependency_snapshot=$3,updated_at=$4 WHERE profile_id=$1 AND number=$2`, id, revision, encoded, now); err != nil {
			return Outcome{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE traceability_profiles SET version=version+1,updated_at=$2 WHERE id=$1`, id, now); err != nil {
			return Outcome{}, err
		}
		if err = s.audit(ctx, tx, meta, id, revision, domain.ActionSubmit, p.Version, p.Version+1, now, request.Reason); err != nil {
			return Outcome{}, err
		}
		return profileOutcome(ctx, tx, id, 200)
	})
	return commandResult(result), err
}

func (s *Profiles) DecideRevision(ctx context.Context, meta domain.CommandMetadata, id string, revision int, action string, request domain.RevisionCommand) (domain.CommandResult, error) {
	if action != domain.ActionApprove && action != domain.ActionReject {
		return domain.CommandResult{}, ErrInvalidCommand
	}
	cmd, err := revisionCommand(meta, id, revision, action, request)
	if err != nil {
		return domain.CommandResult{}, err
	}
	result, err := (Commands{DB: s.DB}).Run(ctx, cmd, func(ctx context.Context, tx pgx.Tx) (Outcome, error) {
		p, err := lockProfile(ctx, tx, id, request.ExpectedVersion)
		if err != nil {
			return Outcome{}, err
		}
		r, err := findRevision(&p, revision)
		if err != nil {
			return Outcome{}, err
		}
		if r.Status != "IN_REVIEW" {
			return Outcome{}, domain.ErrConflict
		}
		for _, actor := range r.Contributors {
			if strings.EqualFold(actor, meta.PrincipalID) {
				return Outcome{}, domain.ErrForbidden
			}
		}
		now := s.Clock().UTC()
		decision, state := "REJECT", "REJECTED"
		if action == domain.ActionApprove {
			decision, state = "APPROVE", "APPROVED"
			if err = r.Content.ValidateSubmission(p.Binding, now, s.VideoHosts); err != nil {
				return Outcome{}, err
			}
		}
		decisionID, err := s.ID()
		if err != nil {
			return Outcome{}, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO review_decisions(id,profile_id,revision_number,actor_id,decision,reason,profile_version,decided_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, decisionID, id, revision, meta.PrincipalID, decision, request.Reason, p.Version+1, now)
		if err != nil {
			return Outcome{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE traceability_revisions SET status=$3,updated_at=$4 WHERE profile_id=$1 AND number=$2`, id, revision, state, now); err != nil {
			return Outcome{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE traceability_profiles SET version=version+1,updated_at=$2 WHERE id=$1`, id, now); err != nil {
			return Outcome{}, err
		}
		if err = s.audit(ctx, tx, meta, id, revision, action, p.Version, p.Version+1, now, request.Reason); err != nil {
			return Outcome{}, err
		}
		return profileOutcome(ctx, tx, id, 200)
	})
	return commandResult(result), err
}
