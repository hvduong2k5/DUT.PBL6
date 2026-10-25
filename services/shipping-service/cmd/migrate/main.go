package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/shipping-service/migrations"
)

func main() {
	databaseURL := os.Getenv("SHIPPING_DATABASE_URL")
	if databaseURL == "" {
		fmt.Fprintln(os.Stderr, "SHIPPING_DATABASE_URL is required")
		os.Exit(1)
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer pool.Close()
	applied, err := migrations.Apply(context.Background(), pool)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, name := range applied {
		fmt.Println(name)
	}
}
