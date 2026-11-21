package main

import (
	"context"
	"flag"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/notification-service/internal/adapters/identity"
	"github.com/omamx/notification-service/internal/adapters/postgres"
	"github.com/omamx/notification-service/internal/config"
	transport "github.com/omamx/notification-service/internal/transport/http"
	"github.com/omamx/notification-service/internal/usecase"
	"github.com/omamx/notification-service/migrations"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func run() error {
	migrate := flag.Bool("migrate", false, "apply service-owned migrations then exit")
	flag.Parse()
	c, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pc, err := pgxpool.ParseConfig(c.DatabaseURL)
	if err != nil {
		return err
	}
	pc.MaxConns = 10
	pc.ConnConfig.ConnectTimeout = 5 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return err
	}
	defer pool.Close()
	start, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if *migrate {
		return migrations.Apply(start, pool)
	}
	if err = migrations.Check(start, pool); err != nil {
		return err
	}
	api := transport.API{Service: usecase.Service{Store: postgres.Store{Pool: pool}}, Auth: identity.Local{Token: c.LocalToken}, Ready: func(ctx context.Context) error {
		if e := pool.Ping(ctx); e != nil {
			return e
		}
		return migrations.Check(ctx, pool)
	}}
	srv := &http.Server{Addr: c.Address, Handler: api.Router(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	done := make(chan error, 1)
	go func() { done <- srv.ListenAndServe() }()
	select {
	case err = <-done:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	case <-ctx.Done():
	}
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	return srv.Shutdown(shutdown)
}
func main() {
	if run() != nil {
		slog.Error("notification stopped; check mode, database and schema configuration")
		os.Exit(1)
	}
}
