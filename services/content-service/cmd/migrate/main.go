// Command migrate applies (up, default) or reverts one step of (down) the
// service migrations against DATABASE_URL.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/config"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/infrastructure/migrate"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/internal/infrastructure/postgres"
	"github.com/hvduong2k5/DUT.PBL6/services/content-service/migrations"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}
func run() error {
	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	if cmd != "up" && cmd != "down" {
		return fmt.Errorf("usage: migrate [up|down]")
	}
	c, err := config.LoadConfig()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	pool, err := postgres.Open(ctx, c)
	if err != nil {
		return err
	}
	defer pool.Close()
	if cmd == "down" {
		v, err := migrate.Down(ctx, pool, migrations.FS)
		if err != nil {
			return err
		}
		fmt.Printf("reverted migration %06d\n", v)
		return nil
	}
	done, err := migrate.Up(ctx, pool, migrations.FS)
	if err != nil {
		return err
	}
	fmt.Printf("applied %d migration(s) %v\n", len(done), done)
	return nil
}
