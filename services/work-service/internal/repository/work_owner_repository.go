package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkOwnerRepository currently validates onboarding completion for the
// shared profiles table; the legacy owner_profile model has been removed.
type WorkOwnerRepository struct {
	db *pgxpool.Pool
}

func NewWorkOwnerRepository(db *pgxpool.Pool) *WorkOwnerRepository {
	return &WorkOwnerRepository{db: db}
}

func (r *WorkOwnerRepository) HasCompletedOnboarding(
	ctx context.Context,
	userID string,
) (bool, error) {
	var exists int

	err := r.db.QueryRow(ctx, `
		SELECT 1
		FROM profiles
		WHERE user_id = $1
			AND onboarding_completed = TRUE
	`, userID).Scan(&exists)

	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *WorkOwnerRepository) HasCompletedWorkerProfile(
	ctx context.Context,
	userID string,
) (bool, error) {
	var exists int

	err := r.db.QueryRow(ctx, `
		SELECT 1
		FROM worker_profiles
		WHERE user_id = $1
			AND profile_completed = TRUE
	`, userID).Scan(&exists)

	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
