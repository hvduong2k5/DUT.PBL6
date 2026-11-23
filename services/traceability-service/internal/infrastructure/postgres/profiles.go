package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Profiles struct {
	DB         *pgxpool.Pool
	Mode       string
	Clock      func() time.Time
	ID         func() (string, error)
	QR         func() (string, error)
	VideoHosts []string
}

func NewProfiles(db *pgxpool.Pool, mode string) (*Profiles, error) {
	if db == nil || (mode != "sandbox" && mode != "integration") {
		return nil, ErrInvalidCommand
	}
	return &Profiles{DB: db, Mode: mode, Clock: time.Now, ID: domain.NewUUID, QR: domain.NewQRCode}, nil
}

func (s *Profiles) ReplayCreateDraft(ctx context.Context, meta domain.CommandMetadata, request domain.CreateDraftRequest) (domain.CommandResult, bool, error) {
	request.Content = request.Content.Normalize()
	request.ProductID = strings.ToLower(request.ProductID)
	cmd, err := draftCommand(meta, "profile.create", "", request)
	if err != nil {
		return domain.CommandResult{}, false, err
	}
	result, found, err := (Commands{DB: s.DB}).Lookup(ctx, cmd)
	return commandResult(result), found, err
}

// The application authorizes and validates dependencies before calling this port.
// The repository repeats local content checks and owns all transactional writes.
func (s *Profiles) CreateDraft(ctx context.Context, meta domain.CommandMetadata, request domain.CreateDraftRequest) (domain.CommandResult, error) {
	request.Content = request.Content.Normalize()
	if err := request.Binding.Validate(); err != nil {
		return domain.CommandResult{}, err
	}
	if err := request.Content.ValidateDraft(s.VideoHosts); err != nil {
		return domain.CommandResult{}, err
	}
	request.ProductID = strings.ToLower(request.ProductID)
	cmd, err := draftCommand(meta, "profile.create", "", request)
	if err != nil {
		return domain.CommandResult{}, err
	}
	result, err := (Commands{DB: s.DB}).Run(ctx, cmd, func(ctx context.Context, tx pgx.Tx) (Outcome, error) {
		now := s.Clock().UTC()
		content, err := json.Marshal(request.Content)
		if err != nil {
			return Outcome{}, err
		}
		var id string
		for attempt := 0; attempt < 4; attempt++ {
			id, err = s.ID()
			if err != nil {
				return Outcome{}, err
			}
			qr, err := s.QR()
			if err != nil {
				return Outcome{}, err
			}
			nested, err := tx.Begin(ctx)
			if err != nil {
				return Outcome{}, err
			}
			_, err = nested.Exec(ctx, `INSERT INTO traceability_profiles(id,mode,product_id,sku_code,source_namespace,batch_id,qr_code,created_at,updated_at)
    VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8)`, id, s.Mode, request.ProductID, request.SKUCode, request.SourceNamespace, request.BatchID, qr, now)
			if err != nil {
				if rollbackErr := nested.Rollback(ctx); rollbackErr != nil {
					return Outcome{}, rollbackErr
				}
				var pg *pgconn.PgError
				if errors.As(err, &pg) && pg.Code == "23505" && pg.ConstraintName == "traceability_profiles_qr_code_key" {
					continue
				}
				return Outcome{}, mapWriteError(err)
			}
			if err = nested.Commit(ctx); err != nil {
				return Outcome{}, err
			}
			_, err = tx.Exec(ctx, `INSERT INTO traceability_revisions(profile_id,number,content,contributors,created_by,created_at,updated_at)
    VALUES($1,1,$2,ARRAY[$3::uuid],$3,$4,$4)`, id, content, meta.PrincipalID, now)
			if err != nil {
				return Outcome{}, err
			}
			if err = s.audit(ctx, tx, meta, id, 1, "profile.create", 0, 1, now); err != nil {
				return Outcome{}, err
			}
			return profileOutcome(ctx, tx, id, 201)
		}
		return Outcome{}, domain.ErrConflict
	})
	return commandResult(result), err
}

func (s *Profiles) EditDraft(ctx context.Context, meta domain.CommandMetadata, id string, revision int, request domain.EditDraftRequest) (domain.CommandResult, error) {
	request.Content = request.Content.Normalize()
	if !domain.ValidUUID(id) || revision < 1 || request.ExpectedVersion < 1 {
		return domain.CommandResult{}, ErrInvalidCommand
	}
	if err := request.Content.ValidateDraft(s.VideoHosts); err != nil {
		return domain.CommandResult{}, err
	}
	resource := struct {
		ID       string
		Revision int
	}{strings.ToLower(id), revision}
	cmd, err := draftCommand(meta, "profile.edit", resource, request)
	if err != nil {
		return domain.CommandResult{}, err
	}
	result, err := (Commands{DB: s.DB}).Run(ctx, cmd, func(ctx context.Context, tx pgx.Tx) (Outcome, error) {
		var version int64
		var status string
		err := tx.QueryRow(ctx, `SELECT version,status FROM traceability_profiles WHERE id=$1 FOR UPDATE`, id).Scan(&version, &status)
		if errors.Is(err, pgx.ErrNoRows) {
			return Outcome{}, domain.ErrNotFound
		}
		if err != nil {
			return Outcome{}, err
		}
		if version != request.ExpectedVersion || status != "ACTIVE" {
			return Outcome{}, domain.ErrConflict
		}
		var state string
		err = tx.QueryRow(ctx, `SELECT status FROM traceability_revisions WHERE profile_id=$1 AND number=$2 FOR UPDATE`, id, revision).Scan(&state)
		if errors.Is(err, pgx.ErrNoRows) {
			return Outcome{}, domain.ErrNotFound
		}
		if err != nil {
			return Outcome{}, err
		}
		if state != "DRAFT" {
			return Outcome{}, domain.ErrConflict
		}
		now := s.Clock().UTC()
		content, err := json.Marshal(request.Content)
		if err != nil {
			return Outcome{}, err
		}
		_, err = tx.Exec(ctx, `UPDATE traceability_revisions SET content=$1,contributors=CASE WHEN $2::uuid=ANY(contributors) THEN contributors ELSE array_append(contributors,$2::uuid) END,updated_at=$3 WHERE profile_id=$4 AND number=$5`, content, meta.PrincipalID, now, id, revision)
		if err != nil {
			return Outcome{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE traceability_profiles SET version=version+1,updated_at=$2 WHERE id=$1`, id, now); err != nil {
			return Outcome{}, err
		}
		if err = s.audit(ctx, tx, meta, id, revision, "profile.edit", version, version+1, now); err != nil {
			return Outcome{}, err
		}
		return profileOutcome(ctx, tx, id, 200)
	})
	return commandResult(result), err
}

func draftCommand(meta domain.CommandMetadata, action string, resource, request any) (Command, error) {
	if !textRequestID(meta.RequestID) {
		return Command{}, ErrInvalidCommand
	}
	// Resource is encoded structurally so revision boundaries cannot collide.
	encoded, err := json.Marshal(resource)
	if err != nil {
		return Command{}, err
	}
	fingerprint, err := Fingerprint(action, string(encoded), request)
	return Command{PrincipalID: meta.PrincipalID, Action: action, Key: meta.Key, Fingerprint: fingerprint}, err
}
func textRequestID(id string) bool {
	return len(id) > 0 && len(id) <= 128 && !strings.ContainsAny(id, "\x00\r\n")
}
func commandResult(o Outcome) domain.CommandResult {
	return domain.CommandResult{Status: o.Status, Body: o.Body, Replayed: o.Replayed}
}
func mapWriteError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return domain.ErrConflict
	}
	return err
}

func (s *Profiles) audit(ctx context.Context, tx pgx.Tx, meta domain.CommandMetadata, id string, revision int, action string, before, after int64, now time.Time, reason ...string) error {
	why := ""
	if len(reason) > 0 {
		why = reason[0]
	}
	return s.auditStates(ctx, tx, meta, id, revision, action, map[string]int64{"version": before}, map[string]int64{"version": after}, now, why)
}

func (s *Profiles) auditStates(ctx context.Context, tx pgx.Tx, meta domain.CommandMetadata, id string, revision int, action string, before, after any, now time.Time, why string) error {
	auditID, err := s.ID()
	if err != nil {
		return err
	}
	previous, err := json.Marshal(before)
	if err != nil {
		return err
	}
	next, err := json.Marshal(after)
	if err != nil {
		return err
	}
	var revisionNumber *int
	if revision > 0 {
		revisionNumber = &revision
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_entries(id,profile_id,revision_number,actor_id,action,before_state,after_state,request_id,occurred_at,reason) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, auditID, id, revisionNumber, meta.PrincipalID, action, previous, next, meta.RequestID, now, why)
	return err
}

type querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func readProfile(ctx context.Context, q querier, id string) (domain.Profile, error) {
	var p domain.Profile
	err := q.QueryRow(ctx, `SELECT id::text,mode,product_id::text,sku_code,source_namespace,batch_id,qr_code,status,version,public_revision,ever_published,created_at,updated_at FROM traceability_profiles WHERE id=$1`, id).Scan(&p.ID, &p.Mode, &p.ProductID, &p.SKUCode, &p.SourceNamespace, &p.BatchID, &p.QRCode, &p.Status, &p.Version, &p.PublicRevision, &p.EverPublished, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, domain.ErrNotFound
	}
	if err != nil {
		return p, err
	}
	rows, err := q.Query(ctx, `SELECT number,status,content,dependency_snapshot,contributors::text[],created_by::text,created_at,updated_at FROM traceability_revisions WHERE profile_id=$1 ORDER BY number`, id)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	p.Revisions = []domain.Revision{}
	for rows.Next() {
		var r domain.Revision
		var content, snapshot []byte
		if err = rows.Scan(&r.Number, &r.Status, &content, &snapshot, &r.Contributors, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return p, err
		}
		if err = json.Unmarshal(content, &r.Content); err != nil {
			return p, err
		}
		if snapshot != nil {
			if err = json.Unmarshal(snapshot, &r.DependencySnapshot); err != nil {
				return p, err
			}
		}
		p.Revisions = append(p.Revisions, r)
	}
	return p, rows.Err()
}
func profileOutcome(ctx context.Context, tx pgx.Tx, id string, status int) (Outcome, error) {
	p, err := readProfile(ctx, tx, id)
	if err != nil {
		return Outcome{}, err
	}
	body, err := json.Marshal(p)
	return Outcome{ResourceID: id, Status: status, Body: body}, err
}

func (s *Profiles) Get(ctx context.Context, id string) (domain.Profile, error) {
	if !domain.ValidUUID(id) {
		return domain.Profile{}, ErrInvalidCommand
	}
	// A repeatable-read transaction prevents mixing root version and later revisions.
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return domain.Profile{}, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	p, err := readProfile(ctx, tx, id)
	if err != nil {
		return p, err
	}
	return p, tx.Commit(ctx)
}
