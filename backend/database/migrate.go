package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies all pending up migrations against an already-open
// database connection. migrationsPath should be "file://migrations" locally
// or "file:///app/migrations" in Docker.
func RunMigrations(db *sql.DB, dbName, migrationsPath string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("creating migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsPath, dbName, driver)
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("applying migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("checking migration version: %w", err)
	}

	if dirty {
		return fmt.Errorf("database is in a dirty migration state at version %d — manual intervention required", version)
	}

	log.Printf("Database migrations up to date at version %d", version)
	return nil
}
