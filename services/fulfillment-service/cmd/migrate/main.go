package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/fulfillment-service/migrations"
	"log"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	dsn := os.Getenv("FULFILLMENT_DATABASE_URL")
	if dsn == "" {
		log.Fatal("FULFILLMENT_DATABASE_URL required")
	}
	pool, e := pgxpool.New(ctx, dsn)
	if e != nil {
		log.Fatal(e)
	}
	defer pool.Close()
	if e = migrations.Apply(ctx, pool); e != nil {
		log.Fatal(e)
	}
	log.Print("Fulfillment migrations applied")
}
