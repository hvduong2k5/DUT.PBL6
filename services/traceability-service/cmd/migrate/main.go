package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/migrations"
	"github.com/jackc/pgx/v5"
)

func main() {
	role := flag.String("runtime-role", "", "existing unprivileged login to grant service DML (optional)")
	flag.Parse()
	if err := run(*role); err != nil {
		// Connection errors can embed credentials/host details. Never log raw errors.
		slog.Error("migration failed; check mode, database access, schema checksum and runtime role")
		os.Exit(1)
	}
	slog.Info("migration completed")
}

func run(role string) error {
	if os.Getenv("DATABASE_URL") == "" {
		return errors.New("DATABASE_URL is required")
	}
	if mode := os.Getenv("APP_MODE"); mode != "sandbox" && mode != "integration" {
		return errors.New("explicit sandbox or integration APP_MODE is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	if err = migrations.Apply(ctx, conn, os.Getenv("APP_MODE")); err != nil {
		return err
	}
	if role != "" {
		return migrations.GrantRuntime(ctx, conn, role)
	}
	return nil
}
