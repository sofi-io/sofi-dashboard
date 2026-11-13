package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-football-compliance/sofi-dashboard/internal/config"
	"github.com/open-football-compliance/sofi-dashboard/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	log.Printf("SOFI Dashboard worker started (environment=%s)", cfg.Environment)
	log.Printf("worker is idle; discovery and extraction jobs will be added in subsequent steps")

	<-ctx.Done()
	log.Printf("shutdown signal received; stopping worker")
}
