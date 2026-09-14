//go:build ignore

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, "postgres://postgres:root@localhost:5432/krewops?sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	tables := []string{}
	rows, err := pool.Query(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		ORDER BY table_name
	`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			panic(err)
		}
		tables = append(tables, name)
	}
	fmt.Println("tables:", tables)

	for _, table := range []string{"auth_schema_migrations", "user_schema_migrations", "schema_migrations"} {
		var version int64
		var dirty bool
		err := pool.QueryRow(ctx, fmt.Sprintf(`SELECT version, dirty FROM %s LIMIT 1`, table)).Scan(&version, &dirty)
		if err != nil {
			fmt.Printf("%s: not found or empty (%v)\n", table, err)
			continue
		}
		fmt.Printf("%s: version=%d dirty=%v\n", table, version, dirty)
	}
}
