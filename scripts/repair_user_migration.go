//go:build ignore

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
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	var columnExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = 'profiles'
			  AND column_name = 'onboarding_completed'
		)
	`).Scan(&columnExists)
	if err != nil {
		panic(err)
	}

	forceVersion := 5
	if columnExists {
		forceVersion = 6
	}

	commandTag, err := pool.Exec(ctx, `
		UPDATE user_schema_migrations
		SET version = $1, dirty = FALSE
		WHERE version = 6
	`, forceVersion)
	if err != nil {
		panic(err)
	}

	if commandTag.RowsAffected() != 1 {
		panic("expected exactly one dirty user migration row")
	}

	fmt.Printf("repaired user_schema_migrations at version %d; onboarding_completed exists: %t\n", forceVersion, columnExists)
}
