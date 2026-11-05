package postgres

import (
	"context"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

func Open(ctx context.Context, c config.Config) (*pgxpool.Pool, error) {
	p, err := pgxpool.ParseConfig(c.DatabaseURL)
	if err != nil {
		return nil, err
	}
	p.MaxConns = c.DBMaxConns
	p.MinConns = 0
	p.MaxConnIdleTime = 5 * time.Minute
	p.ConnConfig.ConnectTimeout = 3 * time.Second
	return pgxpool.NewWithConfig(ctx, p)
}

// Check both connectivity and the service-owned migration, rather than a ping
// that could report ready against an empty or wrong database.
func Ready(ctx context.Context, p *pgxpool.Pool) error {
	_, err := p.Exec(ctx, `SELECT a.id,r.id,d.id,h.slug FROM articles a LEFT JOIN article_revisions r ON false LEFT JOIN approval_decisions d ON false LEFT JOIN slug_history h ON false LIMIT 0`)
	return err
}
