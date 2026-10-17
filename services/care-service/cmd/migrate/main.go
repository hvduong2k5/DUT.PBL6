package main

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/care-service/migrations"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Str("service", "care-migrate").Logger()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal().Msg("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("parse database configuration")
	}
	config.MaxConns = 1
	config.MinConns = 0
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatal().Err(err).Msg("open PostgreSQL pool")
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("connect to PostgreSQL")
	}

	applied, err := migrations.Apply(ctx, pool)
	if err != nil {
		log.Fatal().Err(err).Msg("apply Care schema migrations")
	}
	for _, item := range applied {
		log.Info().Int64("version", item.Version).Str("migration", item.Name).Msg("migration applied")
	}
	log.Info().Int("applied", len(applied)).Msg("Care schema is current")
}
