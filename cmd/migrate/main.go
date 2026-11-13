package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"

	"github.com/sofi-io/sofi-dashboard/internal/config"
	"github.com/sofi-io/sofi-dashboard/internal/database"
)

func main() {
	if err := run(); err != nil {
		log.Printf("migration failed: %v", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		return fmt.Errorf(
			"usage: migrate [up|down|version|force <version>]",
		)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	m, err := database.NewMigrator(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() {
		sourceErr, databaseErr := m.Close()
		if sourceErr != nil {
			log.Printf("close migration source: %v", sourceErr)
		}
		if databaseErr != nil {
			log.Printf("close migration database: %v", databaseErr)
		}
	}()

	switch os.Args[1] {
	case "up":
		err = m.Up()
		if err == migrate.ErrNoChange {
			fmt.Println("Database is already up to date.")
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Println("Migrations applied successfully.")

	case "down":
		err = m.Steps(-1)
		if err == migrate.ErrNoChange {
			fmt.Println("No migration to roll back.")
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Println("Migration rolled back successfully.")

	case "version":
		version, dirty, err := m.Version()
		if err == migrate.ErrNilVersion {
			fmt.Println("No migration version recorded.")
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Printf("Version: %d\nDirty: %t\n", version, dirty)

	case "force":
		if len(os.Args) != 3 {
			return fmt.Errorf("usage: migrate force <version>")
		}

		version, err := strconv.Atoi(os.Args[2])
		if err != nil || version < 0 {
			return fmt.Errorf("invalid migration version: %s", os.Args[2])
		}

		if err := m.Force(version); err != nil {
			return fmt.Errorf("force migration version: %w", err)
		}

		fmt.Printf("Migration version forced to %d.\n", version)

	default:
		return fmt.Errorf("unsupported command: %s", os.Args[1])
	}

	return nil
}
