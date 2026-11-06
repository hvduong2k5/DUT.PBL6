package main

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/application"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/config"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/domain"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/dependency"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/identity"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/infrastructure/postgres"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/jsonutil"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/observability"
	httptransport "github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/transport/http"
	"github.com/jackc/pgx/v5/pgxpool"
)

func assemble(ctx context.Context, cfg config.Config, getenv func(string) string, logger *slog.Logger) (http.Handler, func(), error) {
	if cfg.Mode != "sandbox" {
		return nil, nil, errors.New("integration adapters are not installed")
	}
	operations, err := config.LoadOperations(getenv)
	if err != nil {
		return nil, nil, err
	}
	keyPath, fixturePath := getenv("SANDBOX_JWT_PUBLIC_KEY_FILE"), getenv("SANDBOX_FIXTURE_FILE")
	if keyPath == "" || fixturePath == "" {
		return nil, nil, errors.New("sandbox requires explicit verification key and synthetic fixture files")
	}
	keyData, err := readBounded(keyPath, 16*1024)
	if err != nil {
		return nil, nil, errors.New("cannot read sandbox verification key")
	}
	block, rest := pem.Decode(keyData)
	if block == nil || len(rest) != 0 || block.Type != "PUBLIC KEY" {
		return nil, nil, errors.New("sandbox requires one PKIX PUBLIC KEY PEM block")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, nil, errors.New("invalid sandbox public key")
	}
	public, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return nil, nil, errors.New("sandbox public key must be Ed25519")
	}
	verifier, err := identity.NewSandboxVerifier(cfg.Mode, map[string]ed25519.PublicKey{"sandbox-local": public}, time.Now)
	if err != nil {
		return nil, nil, err
	}
	fixtureData, err := readBounded(fixturePath, 256*1024)
	if err != nil {
		return nil, nil, errors.New("cannot read synthetic source fixture")
	}
	var sources struct {
		Batches []domain.DependencySnapshot `json:"batches"`
	}
	if jsonutil.Decode(fixtureData, &sources) != nil {
		return nil, nil, errors.New("invalid synthetic fixture document")
	}
	dependencies, err := dependency.NewFixture(cfg.Mode, sources.Batches, time.Now)
	if err != nil {
		return nil, nil, err
	}
	dbConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, nil, errors.New("invalid runtime database configuration")
	}
	dbConfig.MaxConns = 10
	dbConfig.ConnConfig.ConnectTimeout = 3 * time.Second
	dbConfig.ConnConfig.RuntimeParams["application_name"] = "traceability-service"
	pool, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		return nil, nil, errors.New("cannot initialize runtime database")
	}
	ready := func(ctx context.Context) error { return postgres.CheckRuntime(ctx, pool, cfg.Mode) }
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err = ready(checkCtx); err != nil {
		pool.Close()
		return nil, nil, errors.New("runtime database is not ready: verify explicit migrations, mode and unprivileged runtime grants")
	}
	store, err := postgres.NewProfiles(pool, cfg.Mode)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	store.VideoHosts = cfg.VideoHosts
	queue, err := postgres.NewOutbox(pool, postgres.OutboxOptions{Lease: 30 * time.Second, MaxAttempts: 10, BaseRetry: time.Second, MaxRetry: time.Minute})
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	metrics := observability.New(queue, operations.RelayEnabled)
	source := observability.Dependency{Next: dependencies, Metrics: metrics}
	service := &application.Drafts{Store: store, Dependencies: source, Clock: time.Now, VideoHosts: cfg.VideoHosts, PublicWebURL: cfg.PublicWebURL}
	workflow := &application.Workflow{Store: store, Dependencies: source, Clock: time.Now, VideoHosts: cfg.VideoHosts, PublicWebURL: cfg.PublicWebURL}
	stopWorkers, err := background(ctx, operations, queue, store, metrics, logger)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	return httptransport.NewHandler(httptransport.HandlerConfig{Readiness: ready, Logger: logger, Identity: verifier, Drafts: service, Workflow: workflow, Metrics: metrics}), func() { stopWorkers(); pool.Close() }, nil
}

func readBounded(path string, max int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil || !stat.Mode().IsRegular() || stat.Size() > max {
		return nil, errors.New("invalid configuration file")
	}
	// A bounded reader also protects against the file growing after stat.
	data, err := io.ReadAll(io.LimitReader(file, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, errors.New("configuration file too large")
	}
	return data, nil
}
