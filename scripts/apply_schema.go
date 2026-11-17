//go:build ignore

// One-time schema setup for a *fresh* Postgres database (local, or a new
// Cloud SQL instance before first deploy).
//
// This is intentionally NOT a migration tool: it applies scripts/schema.sql
// exactly once and refuses to run again against a database that already has
// the schema, so it's safe to point at a real Cloud SQL instance without
// risking the destructive drop/recreate workflow used for local dev
// (scripts/reset_db.go). Once a database has real data, schema *changes*
// need a proper versioned migration tool (e.g. golang-migrate) instead of
// this script — this only covers the very first "create the tables" step.
//
// Usage:
//
//	# local (reads DATABASE_URL, falls back to the same default used by
//	# the other local dev scripts if unset)
//	go run scripts/apply_schema.go
//
//	# Cloud SQL / any remote Postgres
//	DATABASE_URL="postgres://user:pass@host:5432/krewops?sslmode=disable" \
//	  go run scripts/apply_schema.go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:root@localhost:5432/krewops?sslmode=disable"
		fmt.Println("DATABASE_URL not set; defaulting to local dev database")
	}

	schemaPath := "scripts/schema.sql"
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		panic(fmt.Errorf("failed to read %s (run this from the repo root): %w", schemaPath, err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		panic(fmt.Errorf("failed to connect: %w", err))
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		panic(fmt.Errorf("failed to ping database: %w", err))
	}

	var alreadyApplied bool
	err = pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'users')",
	).Scan(&alreadyApplied)
	if err != nil {
		panic(fmt.Errorf("failed to check existing schema: %w", err))
	}

	if alreadyApplied {
		fmt.Println("schema already applied (table 'users' exists) — refusing to run again; " +
			"use a migration tool for subsequent schema changes")
		return
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to begin transaction: %w", err))
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op if Commit succeeds

	if _, err := tx.Exec(ctx, string(schemaBytes)); err != nil {
		panic(fmt.Errorf("failed to apply schema.sql: %w", err))
	}

	if err := tx.Commit(ctx); err != nil {
		panic(fmt.Errorf("failed to commit schema: %w", err))
	}

	fmt.Println("schema applied successfully")
}
