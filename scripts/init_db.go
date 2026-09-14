//go:build ignore

package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	adminURL := "postgres://postgres:root@localhost:5432/postgres?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), adminURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	var exists bool
	err = pool.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = 'krewops')",
	).Scan(&exists)
	if err != nil {
		panic(err)
	}

	if !exists {
		_, err = pool.Exec(context.Background(), "CREATE DATABASE krewops")
		if err != nil {
			panic(err)
		}
		fmt.Println("created database krewops")
	} else {
		fmt.Println("database krewops already exists")
	}

	krewopsURL := "postgres://postgres:root@localhost:5432/krewops?sslmode=disable"
	kpool, err := pgxpool.New(context.Background(), krewopsURL)
	if err != nil {
		panic(err)
	}
	defer kpool.Close()

	if err := kpool.Ping(context.Background()); err != nil {
		panic(err)
	}
	fmt.Println("connected to krewops OK")
}
