package database

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	sqlmigrations "github.com/sofi-io/sofi-dashboard/migrations"
)

func NewMigrator(databaseURL string) (*migrate.Migrate, error) {
	source, err := iofs.New(sqlmigrations.Files, ".")
	if err != nil {
		return nil, fmt.Errorf("initialize migration source: %w", err)
	}

	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		_ = source.Close()
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	db := stdlib.OpenDB(*config)

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		_ = db.Close()
		_ = source.Close()
		return nil, fmt.Errorf("initialize postgres migration driver: %w", err)
	}

	migrator, err := migrate.NewWithInstance(
		"iofs",
		source,
		"postgres",
		driver,
	)
	if err != nil {
		_ = driver.Close()
		_ = source.Close()
		return nil, fmt.Errorf("create migrator: %w", err)
	}

	return migrator, nil
}
