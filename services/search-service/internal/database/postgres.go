package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool connects search-service to the shared KrewOps database.
// search-service does not own any tables or migrations - it only issues
// read queries against tables owned by user-service (profiles,
// worker_profiles, worker_profile_skills) and work-service (work_postings),
// plus the shared work_types lookup table.
func NewPostgresPool(
	ctx context.Context,
	databaseURL string,
) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return pool, nil
}
