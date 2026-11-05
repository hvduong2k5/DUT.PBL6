package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

var _ domain.Repository = (*Repository)(nil)

type querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func dbError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var p *pgconn.PgError
	if errors.As(err, &p) && p.Code == "23505" && p.ConstraintName == "slug_history_pkey" {
		return domain.ErrSlug
	}
	return err
}

// Cancellation must not prevent releasing a transaction after a failed request.
func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}
func (r *Repository) Create(ctx context.Context, a *domain.Article) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	_, err = tx.Exec(ctx, `INSERT INTO articles(id,version,current_revision_id,visible,created_at,updated_at) VALUES($1,$2,$3,false,$4,$5)`, a.ID, a.Version, a.CurrentRevisionID, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return dbError(err)
	}
	if err = saveChildren(ctx, tx, a); err != nil {
		return dbError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return dbError(err)
	}
	a.Slugs = []domain.SlugHistory{{Slug: a.Current().Content.Slug, CreatedAt: a.CreatedAt}}
	return nil
}
func (r *Repository) Mutate(ctx context.Context, id uuid.UUID, version int64, apply func(*domain.Article) error) (*domain.Article, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	var actual int64
	if err = tx.QueryRow(ctx, `SELECT version FROM articles WHERE id=$1 FOR UPDATE`, id).Scan(&actual); err != nil {
		return nil, dbError(err)
	}
	if actual != version {
		return nil, domain.ErrConflict
	}
	a, err := load(ctx, tx, id, true, false)
	if err != nil {
		return nil, err
	}
	if err = apply(a); err != nil {
		return nil, err
	}
	a.Version++
	a.UpdatedAt = time.Now().UTC()
	if err = saveChildren(ctx, tx, a); err != nil {
		return nil, dbError(err)
	}
	tag, err := tx.Exec(ctx, `UPDATE articles SET version=$2,current_revision_id=$3,published_revision_id=$4,visible=$5,updated_at=$6 WHERE id=$1 AND version=$7`, a.ID, a.Version, a.CurrentRevisionID, a.PublishedRevisionID, a.Visible, a.UpdatedAt, version)
	if err != nil {
		return nil, dbError(err)
	}
	if tag.RowsAffected() != 1 {
		return nil, domain.ErrConflict
	}
	// Return authoritative history, including a newly reserved slug.
	a, err = load(ctx, tx, id, true, false)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, dbError(err)
	}
	return a, nil
}
func saveChildren(ctx context.Context, q querier, a *domain.Article) error {
	for _, rev := range a.Revisions {
		seo, err := json.Marshal(rev.Content.SEO)
		if err != nil {
			return err
		}
		media := rev.Content.Media
		if media == nil {
			media = []domain.MediaReference{}
		}
		raw, err := json.Marshal(media)
		if err != nil {
			return err
		}
		_, err = q.Exec(ctx, `INSERT INTO article_revisions(id,article_id,revision_number,status,title,slug,body,category,seo,media,created_by,created_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status`, rev.ID, a.ID, rev.Number, rev.Status, rev.Content.Title, rev.Content.Slug, rev.Content.Body, rev.Content.Category, seo, raw, rev.CreatedBy, rev.CreatedAt)
		if err != nil {
			return err
		}
		// ON CONFLICT is safe only for a slug already owned by this article.
		var owner uuid.UUID
		err = q.QueryRow(ctx, `INSERT INTO slug_history(slug,article_id,created_at) VALUES($1,$2,$3)
ON CONFLICT(slug) DO UPDATE SET slug=EXCLUDED.slug WHERE slug_history.article_id=EXCLUDED.article_id RETURNING article_id`, rev.Content.Slug, a.ID, rev.CreatedAt).Scan(&owner)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrSlug
		}
		if err != nil {
			return err
		}
	}
	for _, d := range a.Decisions {
		_, err := q.Exec(ctx, `INSERT INTO approval_decisions(id,article_id,revision_id,status,reason,actor_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(id) DO NOTHING`, d.ID, a.ID, d.RevisionID, d.Status, d.Reason, d.ActorID, d.CreatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func load(ctx context.Context, q querier, id uuid.UUID, history, public bool) (*domain.Article, error) {
	a := &domain.Article{}
	err := q.QueryRow(ctx, `SELECT id,version,current_revision_id,published_revision_id,visible,created_at,updated_at FROM articles WHERE id=$1`, id).Scan(&a.ID, &a.Version, &a.CurrentRevisionID, &a.PublishedRevisionID, &a.Visible, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, dbError(err)
	}
	selected := a.CurrentRevisionID
	if public {
		if !a.Visible || a.PublishedRevisionID == nil {
			return nil, domain.ErrNotFound
		}
		selected = *a.PublishedRevisionID
	}
	rows, err := q.Query(ctx, `SELECT id,revision_number,status,title,slug,body,category,seo,media,created_by,created_at FROM article_revisions WHERE article_id=$1 AND ($2 OR id=$3) ORDER BY revision_number`, id, history, selected)
	if err != nil {
		return nil, err
	}
	a.Revisions = []domain.Revision{}
	for rows.Next() {
		var rev domain.Revision
		var seo, media []byte
		if err = rows.Scan(&rev.ID, &rev.Number, &rev.Status, &rev.Content.Title, &rev.Content.Slug, &rev.Content.Body, &rev.Content.Category, &seo, &media, &rev.CreatedBy, &rev.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal(seo, &rev.Content.SEO); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal(media, &rev.Content.Media); err != nil {
			rows.Close()
			return nil, err
		}
		a.Revisions = append(a.Revisions, rev)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if len(a.Revisions) == 0 || (public && a.Revision(selected).Status != domain.Published) {
		return nil, domain.ErrNotFound
	}
	if !history {
		return a, nil
	}
	a.Decisions = []domain.Decision{}
	a.Slugs = []domain.SlugHistory{}
	rows, err = q.Query(ctx, `SELECT id,revision_id,status,reason,actor_id,created_at FROM approval_decisions WHERE article_id=$1 ORDER BY created_at,id`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var d domain.Decision
		if err = rows.Scan(&d.ID, &d.RevisionID, &d.Status, &d.Reason, &d.ActorID, &d.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		a.Decisions = append(a.Decisions, d)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = q.Query(ctx, `SELECT slug,created_at FROM slug_history WHERE article_id=$1 ORDER BY created_at,slug`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var h domain.SlugHistory
		if err = rows.Scan(&h.Slug, &h.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		a.Slugs = append(a.Slugs, h)
	}
	err = rows.Err()
	rows.Close()
	return a, err
}

// Multi-query reads share a repeatable-read snapshot so pointer/content/history
// cannot come from different committed versions during publication.
func (r *Repository) read(ctx context.Context, fn func(pgx.Tx) (*domain.Article, error)) (*domain.Article, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	a, err := fn(tx)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return a, nil
}
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*domain.Article, error) {
	return r.read(ctx, func(tx pgx.Tx) (*domain.Article, error) { return load(ctx, tx, id, true, false) })
}
func (r *Repository) GetPublished(ctx context.Context, slug string) (*domain.Article, error) {
	return r.read(ctx, func(tx pgx.Tx) (*domain.Article, error) {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `SELECT a.id FROM articles a JOIN article_revisions r ON r.id=a.published_revision_id AND r.article_id=a.id WHERE a.visible AND r.status='PUBLISHED' AND r.slug=$1`, slug).Scan(&id)
		if err != nil {
			return nil, dbError(err)
		}
		return load(ctx, tx, id, false, true)
	})
}
func (r *Repository) List(ctx context.Context, f domain.Filter) ([]domain.Article, int64, error) {
	if err := f.Validate(); err != nil {
		return nil, 0, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, 0, err
	}
	defer rollback(tx)
	const from = ` FROM articles a JOIN article_revisions r ON r.article_id=a.id AND r.id=CASE WHEN $1 THEN a.published_revision_id ELSE a.current_revision_id END
WHERE (NOT $1 OR (a.visible AND r.status='PUBLISHED')) AND ($2='' OR strpos(lower(r.title || ' ' || r.body),lower($2))>0) AND ($3='' OR r.category=$3) AND ($4='' OR r.status=$4)`
	var total int64
	if err = tx.QueryRow(ctx, `SELECT count(*)`+from, f.Public, f.Query, f.Category, f.Status).Scan(&total); err != nil {
		return nil, 0, err
	}
	// A page is one joined query, without loading article bodies or full history.
	// Detail endpoints still return the full immutable snapshot.
	rows, err := tx.Query(ctx, `SELECT a.id,a.version,a.current_revision_id,a.published_revision_id,a.visible,a.created_at,a.updated_at,
r.id,r.revision_number,r.status,r.title,r.slug,r.category,r.seo,r.media,r.created_by,r.created_at`+from+` ORDER BY a.created_at DESC,a.id DESC LIMIT $5 OFFSET $6`, f.Public, f.Query, f.Category, f.Status, f.Limit, (f.Page-1)*f.Limit)
	if err != nil {
		return nil, 0, err
	}
	items := make([]domain.Article, 0)
	for rows.Next() {
		var a domain.Article
		var rev domain.Revision
		var seo, media []byte
		if err = rows.Scan(&a.ID, &a.Version, &a.CurrentRevisionID, &a.PublishedRevisionID, &a.Visible, &a.CreatedAt, &a.UpdatedAt,
			&rev.ID, &rev.Number, &rev.Status, &rev.Content.Title, &rev.Content.Slug, &rev.Content.Category, &seo, &media, &rev.CreatedBy, &rev.CreatedAt); err != nil {
			rows.Close()
			return nil, 0, err
		}
		if err = json.Unmarshal(seo, &rev.Content.SEO); err != nil {
			rows.Close()
			return nil, 0, err
		}
		if err = json.Unmarshal(media, &rev.Content.Media); err != nil {
			rows.Close()
			return nil, 0, err
		}
		a.Revisions = []domain.Revision{rev}
		items = append(items, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
