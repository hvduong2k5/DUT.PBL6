package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/jsonutil"
	"github.com/jackc/pgx/v5"
)

type profileCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

func (s *Profiles) List(ctx context.Context, query domain.ListQuery) (domain.ProfilePage, error) {
	if query.PageSize == 0 {
		query.PageSize = 20
	}
	if query.PageSize < 1 || query.PageSize > 100 || len(query.Cursor) > 256 {
		return domain.ProfilePage{}, ErrInvalidCommand
	}
	var since any
	var id any
	if query.Cursor != "" {
		data, err := base64.RawURLEncoding.Strict().DecodeString(query.Cursor)
		if err != nil {
			return domain.ProfilePage{}, ErrInvalidCommand
		}
		var cursor profileCursor
		if jsonutil.Decode(data, &cursor) != nil || cursor.CreatedAt.IsZero() || !domain.ValidUUID(cursor.ID) {
			return domain.ProfilePage{}, ErrInvalidCommand
		}
		since, id = cursor.CreatedAt, cursor.ID
	}
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return domain.ProfilePage{}, err
	}
	defer func() {
		clean, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(clean)
	}()
	rows, err := tx.Query(ctx, `SELECT id::text,created_at FROM traceability_profiles WHERE ($1::timestamptz IS NULL OR (created_at,id)>($1::timestamptz,$2::uuid)) ORDER BY created_at,id LIMIT $3`, since, id, query.PageSize+1)
	if err != nil {
		return domain.ProfilePage{}, err
	}
	cursors := []profileCursor{}
	for rows.Next() {
		var c profileCursor
		if err = rows.Scan(&c.ID, &c.CreatedAt); err != nil {
			rows.Close()
			return domain.ProfilePage{}, err
		}
		cursors = append(cursors, c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return domain.ProfilePage{}, err
	}
	page := domain.ProfilePage{Items: []domain.Profile{}}
	if len(cursors) > query.PageSize {
		cursors = cursors[:query.PageSize]
		data, err := json.Marshal(cursors[len(cursors)-1])
		if err != nil {
			return page, err
		}
		next := base64.RawURLEncoding.EncodeToString(data)
		page.NextCursor = &next
	}
	for _, cursor := range cursors {
		profile, err := readProfile(ctx, tx, cursor.ID)
		if err != nil {
			return page, err
		}
		page.Items = append(page.Items, profile)
	}
	return page, tx.Commit(ctx)
}
