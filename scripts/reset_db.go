//go:build ignore

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, "postgres://postgres:root@localhost:5432/krewops?sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	_, err = pool.Exec(ctx, `
		DROP SCHEMA public CASCADE;
		CREATE SCHEMA public;
		GRANT ALL ON SCHEMA public TO postgres;
		GRANT ALL ON SCHEMA public TO public;
	`)
	if err != nil {
		panic(err)
	}

	fmt.Println("reset krewops database schema")
}
