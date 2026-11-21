package main

import (
	"context"
	"database/sql"
	"dut-pbl6/inventory-service/internal/seed"
	"github.com/lib/pq"
	"log"
	"os"
	"time"
)

func main() {
	if len(os.Args) > 2 || (len(os.Args) == 2 && os.Args[1] != "down") {
		log.Fatal("usage: inventory-seed-demo [down]")
	}
	if os.Getenv("INVENTORY_SEED_DEMO") != "true" {
		log.Print("Demo seed disabled; set INVENTORY_SEED_DEMO=true for this job only")
		return
	}
	if os.Getenv("DB_URL") == "" {
		log.Fatal("DB_URL is required")
	}
	connector, err := pq.NewConnector(os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal("open database failed")
	}
	db := sql.OpenDB(pq.ConnectorWithNoticeHandler(connector, func(notice *pq.Error) { log.Print(notice.Message) }))
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if len(os.Args) == 2 {
		err = seed.Rollback(ctx, db)
	} else {
		err = seed.Apply(ctx, db, true)
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Demo seed job completed; existing SKUs skipped (no stock reset)")
}
