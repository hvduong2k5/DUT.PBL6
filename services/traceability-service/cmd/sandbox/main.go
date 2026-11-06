package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/config"
	"github.com/hvduong2k5/DUT.PBL6/services/traceability-service/internal/sandbox"
)

func main() {
	action := flag.String("action", "init", "init, tokens or smoke")
	dir := flag.String("dir", ".sandbox", "dedicated local sandbox directory")
	api := flag.String("api", "http://127.0.0.1:8003", "loopback sandbox API origin for smoke")
	flag.Parse()
	absolute, err := filepath.Abs(*dir)
	if err == nil {
		switch *action {
		case "init":
			var cfg config.Config
			cfg, err = config.Load(os.Getenv)
			if err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				_, err = sandbox.Initialize(ctx, cfg.Mode, cfg.DatabaseURL, absolute, cfg.PublicWebURL, time.Now())
			}
		case "tokens":
			err = sandbox.RefreshTokens(os.Getenv("APP_MODE"), absolute, time.Now())
		case "smoke":
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			var report sandbox.SmokeReport
			report, err = sandbox.Smoke(ctx, os.Getenv("APP_MODE"), absolute, *api)
			if err == nil {
				fmt.Println("Smoke outcome:", report.Outcome, "profile:", report.ProfileID)
			}
		default:
			err = fmt.Errorf("unknown sandbox action")
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "sandbox setup failed:", err)
		os.Exit(1)
	}
	fmt.Println("Sandbox files ready:", absolute, "(credentials omitted)")
}
