package database

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
)

const integrationDatabaseName = "sofi_dashboard"

func newIntegrationDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not configured; skipping integration test")
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}

	if cfg.ConnConfig.Database != integrationDatabaseName {
		t.Fatalf(
			"refusing to run integration tests against database %q; expected %q",
			cfg.ConnConfig.Database,
			integrationDatabaseName,
		)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	migrator, err := NewMigrator(databaseURL)
	if err != nil {
		t.Fatalf("initialize test migrations: %v", err)
	}

	if err := migrator.Up(); err != nil &&
		!errors.Is(err, migrate.ErrNoChange) {
		_, _ = migrator.Close()
		t.Fatalf("apply test migrations: %v", err)
	}

	sourceErr, databaseErr := migrator.Close()
	if sourceErr != nil {
		t.Fatalf("close migration source: %v", sourceErr)
	}
	if databaseErr != nil {
		t.Fatalf("close migration database: %v", databaseErr)
	}

	db, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	t.Cleanup(db.Close)

	return db
}
