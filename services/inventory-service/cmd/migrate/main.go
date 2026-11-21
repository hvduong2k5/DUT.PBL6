package main

import (
	"context"
	"database/sql"
	"dut-pbl6/inventory-service/migrations"
	_ "github.com/lib/pq"
	"log"
	"os"
	"time"
)

func main() {
	if len(os.Args) > 2 || (len(os.Args) == 2 && os.Args[1] != "baseline") {
		log.Fatal("usage: inventory-migrate [baseline]")
	}
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatal("DB_URL is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("open migration database failed")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if len(os.Args) == 2 {
		err = migrations.Baseline(ctx, db)
	} else {
		err = migrations.Apply(ctx, db)
	}
	if err != nil {
		log.Fatalf("Inventory migration failed: %v", err)
	}
	log.Print("Inventory migration job completed")
}
