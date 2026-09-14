//go:build ignore

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func fixMigrationTable(ctx context.Context, pool *pgxpool.Pool, table string) error {
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version bigint NOT NULL PRIMARY KEY,
			dirty boolean NOT NULL
		)
	`, table))
	if err != nil {
		return err
	}

	var exists bool
	err = pool.QueryRow(ctx, fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s)`, table)).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		_, err = pool.Exec(ctx, fmt.Sprintf(`
			UPDATE %s SET version = 1, dirty = false
		`, table))
	} else {
		_, err = pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s (version, dirty) VALUES (1, false)
		`, table))
	}
	return err
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, "postgres://postgres:root@localhost:5432/krewops?sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	var usersExists bool
	err = pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'users')",
	).Scan(&usersExists)
	if err != nil {
		panic(err)
	}

	if usersExists {
		if err := fixMigrationTable(ctx, pool, "auth_schema_migrations"); err != nil {
			panic(err)
		}
		fmt.Println("fixed auth_schema_migrations")
	}

	var profilesExists bool
	err = pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'profiles')",
	).Scan(&profilesExists)
	if err != nil {
		panic(err)
	}

	if profilesExists {
		if err := fixMigrationTable(ctx, pool, "user_schema_migrations"); err != nil {
			panic(err)
		}
		fmt.Println("fixed user_schema_migrations")
	}
}
