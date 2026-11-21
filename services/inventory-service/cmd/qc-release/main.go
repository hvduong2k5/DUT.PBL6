package main

import (
	"context"
	"database/sql"
	"dut-pbl6/inventory-service/internal/qc"
	"dut-pbl6/inventory-service/migrations"
	"encoding/json"
	"flag"
	_ "github.com/lib/pq"
	"log"
	"os"
	"time"
)

func main() {
	var r qc.Request
	flag.StringVar(&r.Actor, "actor", "", "audited employee UUID")
	flag.StringVar(&r.Batch, "batch", "", "batch UUID")
	flag.StringVar(&r.Warehouse, "warehouse", "", "warehouse UUID")
	flag.StringVar(&r.Key, "idempotency-key", "", "unique job key, 16-128 characters")
	flag.Int64Var(&r.Version, "expected-version", 0, "expected batch version")
	flag.StringVar(&r.Manufactured, "manufactured", "", "verified YYYY-MM-DD")
	flag.StringVar(&r.Expires, "expires", "", "verified YYYY-MM-DD")
	flag.StringVar(&r.Reason, "reason", "", "QC evidence reference/reason")
	flag.Parse()
	if os.Getenv("DB_URL") == "" {
		log.Fatal("DB_URL required")
	}
	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal("open database failed")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = migrations.Verify(ctx, db); err != nil {
		log.Fatal(err)
	}
	result, err := qc.Release(ctx, db, r)
	if err != nil {
		log.Fatal(err)
	}
	json.NewEncoder(os.Stdout).Encode(result)
}
