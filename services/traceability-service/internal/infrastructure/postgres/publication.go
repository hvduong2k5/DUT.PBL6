package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

func (s *Profiles) PublishRevision(ctx context.Context, meta domain.CommandMetadata, id string, revision int, request domain.RevisionCommand, snapshot domain.DependencySnapshot) (domain.CommandResult, error) {
	cmd, err := revisionCommand(meta, id, revision, domain.ActionPublish, request)
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
		if r.Status != "APPROVED" || r.DependencySnapshot == nil {
			return Outcome{}, domain.ErrConflict
		}
		now := s.Clock().UTC()
		if err = r.Content.ValidateSubmission(p.Binding, now, s.VideoHosts); err != nil {
			return Outcome{}, err
		}
		if err = snapshot.Validate(p.Binding, now); err != nil {
			return Outcome{}, err
		}
		previous, err := r.DependencySnapshot.SourceHash()
		if err != nil {
			return Outcome{}, err
		}
		current, err := snapshot.SourceHash()
		if err != nil {
			return Outcome{}, err
		}
		if previous != current {
			return Outcome{}, domain.ErrConflict
		}
		if p.PublicRevision != nil {
			if _, err = tx.Exec(ctx, `UPDATE traceability_revisions SET status='ARCHIVED',updated_at=$3 WHERE profile_id=$1 AND number=$2`, id, *p.PublicRevision, now); err != nil {
				return Outcome{}, err
			}
		}
		if _, err = tx.Exec(ctx, `UPDATE traceability_revisions SET status='PUBLISHED',updated_at=$3 WHERE profile_id=$1 AND number=$2`, id, revision, now); err != nil {
			return Outcome{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE traceability_profiles SET public_revision=$2,ever_published=true,version=version+1,updated_at=$3 WHERE id=$1`, id, revision, now); err != nil {
			return Outcome{}, err
		}
		if err = s.audit(ctx, tx, meta, id, revision, domain.ActionPublish, p.Version, p.Version+1, now, request.Reason); err != nil {
			return Outcome{}, err
		}
		if err = s.event(ctx, tx, id, p.Version+1, revision, "profile.published", "ACTIVE", "", now, meta.TraceParent); err != nil {
			return Outcome{}, err
		}
		return profileOutcome(ctx, tx, id, 200)
	})
	return commandResult(result), err
}

func (s *Profiles) SetStatus(ctx context.Context, meta domain.CommandMetadata, id, status string, request domain.RevisionCommand) (domain.CommandResult, error) {
	if !domain.ValidUUID(id) || request.ExpectedVersion < 1 || (status != "SUSPENDED" && status != "ARCHIVED") {
		return domain.CommandResult{}, ErrInvalidCommand
	}
	if err := domain.ValidateReason(request.Reason, true); err != nil {
		return domain.CommandResult{}, err
	}
	action := "profile.suspend"
	event := "profile.suspended"
	if status == "ARCHIVED" {
		action, event = "profile.archive", "profile.archived"
	}
	cmd, err := draftCommand(meta, action, strings.ToLower(id), request)
	if err != nil {
		return domain.CommandResult{}, err
	}
	result, err := (Commands{DB: s.DB}).Run(ctx, cmd, func(ctx context.Context, tx pgx.Tx) (Outcome, error) {
		// Archive may follow suspension; all content mutations require ACTIVE elsewhere.
		var version int64
		var state string
		err := tx.QueryRow(ctx, `SELECT version,status FROM traceability_profiles WHERE id=$1 FOR UPDATE`, id).Scan(&version, &state)
		if errors.Is(err, pgx.ErrNoRows) {
			return Outcome{}, domain.ErrNotFound
		}
		if err != nil {
			return Outcome{}, err
		}
		if version != request.ExpectedVersion || state == "ARCHIVED" || state == status {
			return Outcome{}, domain.ErrConflict
		}
		p, err := readProfile(ctx, tx, id)
		if err != nil {
			return Outcome{}, err
		}
		now := s.Clock().UTC()
		if _, err = tx.Exec(ctx, `UPDATE traceability_profiles SET status=$2,version=version+1,updated_at=$3 WHERE id=$1`, id, status, now); err != nil {
			return Outcome{}, err
		}
		revision := 0
		if p.PublicRevision != nil {
			revision = *p.PublicRevision
		}
		if err = s.audit(ctx, tx, meta, id, revision, action, version, version+1, now, request.Reason); err != nil {
			return Outcome{}, err
		}
		// A never-published Draft does not generate a public projection event.
		if p.EverPublished {
			if err = s.event(ctx, tx, id, version+1, revision, event, status, "", now, meta.TraceParent); err != nil {
				return Outcome{}, err
			}
		}
		return profileOutcome(ctx, tx, id, 200)
	})
	return commandResult(result), err
}

func (s *Profiles) event(ctx context.Context, tx pgx.Tx, id string, version int64, revision int, eventType, status, reasonCode string, now time.Time, traceContext ...string) error {
	eventID, err := s.ID()
	if err != nil {
		return err
	}
	trace := ""
	if len(traceContext) > 0 {
		trace = traceContext[0]
	}
	if trace == "" {
		trace, err = domain.NewTraceParent()
		if err != nil {
			return err
		}
	}
	if !domain.ValidTraceParent(trace) {
		return domain.ErrInvalidCommand
	}
	// No internal evidence, contributor, actor or free-text operator reason enters
	// the distributed business event. Audit retains the reason locally.
	data := domain.EventData{ProfileID: id, AggregateVersion: version, Revision: revision, Mode: s.Mode, Status: status}
	if eventType == "certification.status.changed" {
		eligible := false
		data.Eligible = &eligible
		data.ReasonCode = reasonCode
	}
	payload, err := json.Marshal(domain.CloudEvent{SpecVersion: "1.0", ID: eventID, Source: "https://omama.vn/services/traceability-service", Type: "vn.omama.traceability." + eventType + ".v1", Subject: "profile:" + id, Time: now, DataContentType: "application/json", TraceParent: trace, Data: data})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(id,profile_id,aggregate_version,event_type,topic,partition_key,payload,created_at,retry_at) VALUES($1,$2,$3,$4,'traceability.events.v1',$2::uuid::text,$5,$6,$6)`, eventID, id, version, "vn.omama.traceability."+eventType+".v1", payload, now)
	return err
}

func (s *Profiles) Public(ctx context.Context, code string) (domain.PublicRecord, error) {
	if !domain.ValidQRCode(code) {
		return domain.PublicRecord{}, domain.ErrNotFound
	}
	var id string
	err := s.DB.QueryRow(ctx, `SELECT id::text FROM traceability_profiles WHERE qr_code=$1`, code).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PublicRecord{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.PublicRecord{}, err
	}
	profile, err := s.Get(ctx, id)
	if err != nil {
		return domain.PublicRecord{}, err
	}
	return domain.BuildPublic(profile, s.Clock())
}
