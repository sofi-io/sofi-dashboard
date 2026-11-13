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

	"github.com/sofi-io/sofi-dashboard/internal/competition"
	"github.com/sofi-io/sofi-dashboard/internal/config"
	"github.com/sofi-io/sofi-dashboard/internal/database"
	"github.com/sofi-io/sofi-dashboard/internal/document"
	"github.com/sofi-io/sofi-dashboard/internal/processing"
	"github.com/sofi-io/sofi-dashboard/internal/sofi"
	"github.com/sofi-io/sofi-dashboard/internal/sources"
	"github.com/sofi-io/sofi-dashboard/internal/storage"
)

const (
	workerBatchSize    = 5
	workerPollInterval = 15 * time.Second
	workerLease        = 5 * time.Minute

	extractionBatchSize = 5
	extractionLease     = 5 * time.Minute
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

	dryRun := flags.Bool(
		"dry-run",
		false,
		"Run discovery without persisting results",
	)

	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	discoverer, err := sources.NewDiscoverer(*source)
	if err != nil {
		return err
	}

	log.Printf("starting %s competition discovery", *source)
	log.Printf("competition_url=%s", *competitionURL)

	result, err := discoverer.DiscoverCompetition(
		ctx,
		*competitionURL,
		competition.DiscoveryOptions{
			MaxConcurrentClubs:   *maxConcurrent,
			MaxConcurrentMatches: *maxConcurrentMatches,
		},
	)
	if err != nil {
		return fmt.Errorf("discover competition: %w", err)
	}

	if !*dryRun {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		db, err := database.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("connect database: %w", err)
		}
		defer db.Close()

		repository := database.NewCompetitionRepository(db)

		if err := repository.SaveDiscovery(ctx, result); err != nil {
			return fmt.Errorf("persist discovery: %w", err)
		}

		log.Printf("discovery persisted successfully")
	} else {
		log.Printf("dry-run enabled: skipping database persistence")
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

	store, err := storage.NewFilesystemStorage(
		cfg.DocumentStoragePath,
	)
	if err != nil {
		return fmt.Errorf("initialize document storage: %w", err)
	}

	downloader, err := document.NewDownloader(
		store,
		document.DownloaderConfig{
			Timeout:     30 * time.Second,
			MaxBytes:    20 << 20,
			MaxAttempts: 3,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize downloader: %w", err)
	}

	repository := database.NewDocumentRepository(db)

	processor, err := processing.NewDocumentProcessor(
		repository,
		downloader,
		processing.ProcessorConfig{
			BatchSize: workerBatchSize,
			Lease:     workerLease,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize processor: %w", err)
	}

	if cfg.SOFIAPIBaseURL == "" {
		return fmt.Errorf("SOFI_API_BASE_URL is required")
	}

	if cfg.SOFIAPIKey == "" {
		return fmt.Errorf("SOFI_API_KEY is required")
	}

	sofiClient, err := sofi.NewClient(
		cfg.SOFIAPIBaseURL,
		cfg.SOFIAPIKey,
		nil,
	)
	if err != nil {
		return fmt.Errorf("initialize SOFI API client: %w", err)
	}

	extractionRepository := database.NewExtractionRepository(db)

	dispatcher, err := processing.NewExtractionDispatcher(
		extractionRepository,
		store,
		sofiClient,
		processing.ExtractionDispatcherConfig{
			BatchSize: extractionBatchSize,
			Lease:     extractionLease,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize extraction dispatcher: %w", err)
	}

	log.Printf(
		"SOFI Dashboard worker started (environment=%s)",
		cfg.Environment,
	)
	log.Printf(
		"document storage: %s",
		cfg.DocumentStoragePath,
	)
	log.Printf(
		"batch_size=%d poll_interval=%s",
		workerBatchSize,
		workerPollInterval,
	)

	for {
		if err := ctx.Err(); err != nil {
			break
		}

		// 1. Download discovered documents.
		downloaded, err := processor.ProcessBatch(ctx)

		if ctx.Err() != nil {
			break
		}

		if err != nil {
			log.Printf("document processing error: %v", err)
		}

		if downloaded > 0 {
			log.Printf(
				"document batch completed: processed=%d",
				downloaded,
			)
		}

		// 2. Create extraction tasks for downloaded documents.
		created, err := extractionRepository.CreatePending(
			ctx,
			extractionBatchSize,
		)

		if ctx.Err() != nil {
			break
		}

		if err != nil {
			log.Printf("create pending extractions error: %v", err)
		}

		if created > 0 {
			log.Printf(
				"extraction tasks created: count=%d",
				created,
			)
		}

		// 3. Submit pending extractions to SOFI API.
		dispatched, err := dispatcher.ProcessBatch(ctx)

		if ctx.Err() != nil {
			break
		}

		if err != nil {
			log.Printf("extraction dispatch error: %v", err)
		}

		if dispatched > 0 {
			log.Printf(
				"extraction dispatch completed: submitted=%d",
				dispatched,
			)
		}

		// 4. Immediately continue if either queue was full.
		if downloaded == workerBatchSize ||
			created == extractionBatchSize ||
			dispatched == extractionBatchSize {
			continue
		}

		// 5. Wait before polling again.
		timer := time.NewTimer(workerPollInterval)

		select {
		case <-ctx.Done():
			timer.Stop()

		case <-timer.C:
		}
	}

	log.Printf("worker shutdown completed")
	return nil
}
