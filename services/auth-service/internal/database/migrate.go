package database

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func migrationSourceURL(migrationsPath string) string {
	return "file://" + filepath.ToSlash(migrationsPath)
}

func migrationDatabaseURL(databaseURL string, migrationsTable string) string {
	url := databaseURL
	if strings.HasPrefix(url, "postgres://") {
		url = "pgx5://" + strings.TrimPrefix(url, "postgres://")
	}

	if migrationsTable != "" && !strings.Contains(url, "x-migrations-table=") {
		separator := "?"
		if strings.Contains(url, "?") {
			separator = "&"
		}
		url = fmt.Sprintf("%s%sx-migrations-table=%s", url, separator, migrationsTable)
	}

	return url
}

func RunMigrations(databaseURL string, migrationsPath string, migrationsTable string) error {
	m, err := migrate.New(
		migrationSourceURL(migrationsPath),
		migrationDatabaseURL(databaseURL, migrationsTable),
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
