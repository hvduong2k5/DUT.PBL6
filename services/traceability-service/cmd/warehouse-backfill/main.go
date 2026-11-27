// Controlled one-profile maintenance; never guess warehouses for legacy rows.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/dependency"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/postgres"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/jsonutil"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/migrations"
	traceabilityv1 "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/traceability/v1"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	id := flag.String("profile-id", "", "exact UUID of one legacy profile; reruns are idempotent")
	flag.Parse()
	if err := run(*id); err != nil {
		slog.Error("warehouse backfill failed; profile retained without guessed metadata; check source, schema-owner access and version")
		os.Exit(1)
	}
}

func run(id string) error {
	mode := os.Getenv("APP_MODE")
	if !domain.ValidUUID(id) || (mode != "sandbox" && mode != "integration") || os.Getenv("DATABASE_URL") == "" {
		return errors.New("explicit profile UUID, mode and migrator database required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer pool.Close()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	err = migrations.Verify(ctx, conn.Conn(), mode)
	conn.Release()
	if err != nil {
		return err
	}
	store, err := postgres.NewProfiles(pool, mode)
	if err != nil {
		return err
	}
	profile, err := store.Get(ctx, id)
	if err != nil {
		return err
	}
	var batchSource interface {
		Batch(context.Context, string, string, string) (domain.DependencySnapshot, error)
	}
	if mode == "sandbox" {
		file, err := os.Open(os.Getenv("SANDBOX_FIXTURE_FILE"))
		if err != nil {
			return err
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, 256*1024+1))
		if err != nil || len(data) > 256*1024 {
			return errors.New("invalid fixture size")
		}
		var document struct {
			Batches []domain.DependencySnapshot `json:"batches"`
		}
		if jsonutil.Decode(data, &document) != nil {
			return errors.New("invalid fixture")
		}
		batchSource, err = dependency.NewFixture(mode, document.Batches, time.Now)
		if err != nil {
			return err
		}
	} else {
		token, target := os.Getenv("TRACEABILITY_INVENTORY_TOKEN"), os.Getenv("TRACEABILITY_INVENTORY_ADDR")
		if len(token) < 32 || target == "" {
			return errors.New("explicit Inventory credential and TLS target required")
		}
		tls, err := credentials.NewClientTLSFromFile(os.Getenv("TRACEABILITY_INVENTORY_CA_FILE"), "")
		if err != nil {
			return err
		}
		client, err := grpc.NewClient(target, grpc.WithTransportCredentials(tls))
		if err != nil {
			return err
		}
		defer client.Close()
		batchSource = dependency.Live{InventoryToken: token, Inventory: traceabilityv1.NewBatchSourceClient(client)}
	}
	sourceCtx, done := context.WithTimeout(ctx, 2*time.Second)
	defer done()
	snapshot, err := batchSource.Batch(sourceCtx, profile.SourceNamespace, profile.BatchID, profile.SKUCode)
	if err != nil {
		return err
	}
	if snapshot.SourceNamespace != profile.SourceNamespace || snapshot.BatchID != profile.BatchID || snapshot.SKUCode != profile.SKUCode {
		return domain.ErrConflict
	}
	snapshot.Binding = profile.Binding
	changed, err := store.BackfillWarehouse(ctx, id, profile.Version, snapshot)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"profile_id": id, "warehouse_id": snapshot.WarehouseID, "changed": changed, "batch_version": snapshot.BatchVersion, "source_verified_at": snapshot.CheckedAt})
}
