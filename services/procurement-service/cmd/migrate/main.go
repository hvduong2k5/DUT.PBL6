package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/procurement-service/migrations"
)

func main() {
	databaseURL := os.Getenv("PROCUREMENT_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://procurement:procurement@localhost:5449/procurement_db?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	applied, err := migrations.Apply(ctx, pool)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, name := range applied {
		fmt.Println(name)
	}
}
