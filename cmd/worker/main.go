package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/open-football-compliance/sofi-dashboard/internal/config"
	"github.com/open-football-compliance/sofi-dashboard/internal/database"
	"github.com/open-football-compliance/sofi-dashboard/internal/sources/cbf"
)

func main() {
	var err error

	if len(os.Args) > 1 && os.Args[1] == "discovery" {
		err = runDiscovery(os.Args[2:])
	} else {
		err = runWorker()
	}

	if err != nil {
		log.Printf("worker failed: %v", err)
		os.Exit(1)
	}
}

func runDiscovery(args []string) error {
	flags := flag.NewFlagSet("discovery", flag.ContinueOnError)

	source := flags.String("source", "", "Discovery source")
	competitionURL := flags.String(
		"competition-url",
		"",
		"Official competition URL",
	)
	maxConcurrent := flags.Int(
		"max-concurrent-clubs",
		3,
		"Maximum concurrent club history requests",
	)

	maxConcurrentMatches := flags.Int(
		"max-concurrent-matches",
		3,
		"Maximum concurrent match document requests",
	)

	if err := flags.Parse(args); err != nil {
		return err
	}

	if *maxConcurrentMatches < 1 || *maxConcurrentMatches > 5 {
		return fmt.Errorf("--max-concurrent-matches must be between 1 and 5")
	}

	if *source != "cbf" {
		return fmt.Errorf("unsupported discovery source: %q", *source)
	}

	if *competitionURL == "" {
		return fmt.Errorf("--competition-url is required")
	}

	if *maxConcurrent < 1 || *maxConcurrent > 5 {
		return fmt.Errorf("--max-concurrent-clubs must be between 1 and 5")
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	client := cbf.NewClient()

	log.Printf("starting CBF competition discovery")
	log.Printf("competition_url=%s", *competitionURL)

	result, err := client.DiscoverCompetition(
		ctx,
		*competitionURL,
		cbf.DiscoveryOptions{
			MaxConcurrentClubFetches:  *maxConcurrent,
			MaxConcurrentMatchFetches: *maxConcurrentMatches,
		},
	)
	if err != nil {
		return fmt.Errorf("discover competition: %w", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("encode discovery result: %w", err)
	}

	log.Printf(
		"discovery completed: clubs=%d matches=%d warnings=%d",
		len(result.Clubs),
		len(result.Matches),
		len(result.Warnings),
	)

	return nil
}

func runWorker() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	log.Printf(
		"SOFI Dashboard worker started (environment=%s)",
		cfg.Environment,
	)
	log.Printf("worker is idle; waiting for jobs")

	<-ctx.Done()

	log.Printf("shutdown signal received; stopping worker")
	return nil
}
