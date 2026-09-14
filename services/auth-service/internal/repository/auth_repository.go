package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/model"
)

const maxOTPAttempts = 5

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) FindUserByPhone(
	ctx context.Context,
	phoneNumber string,
) (*model.User, error) {
	query := `
		SELECT
			id,
			phone_number,
			status,
			phone_verified,
			created_at,
			updated_at
		FROM users
		WHERE phone_number = $1
	`

	user := &model.User{}

	err := r.db.QueryRow(ctx, query, phoneNumber).Scan(
		&user.ID,
		&user.PhoneNumber,
		&user.Status,
		&user.PhoneVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *AuthRepository) CreateUser(
	ctx context.Context,
	phoneNumber string,
) (*model.User, error) {
	query := `
		INSERT INTO users (
			phone_number,
			phone_verified
		)
		VALUES ($1, TRUE)
		RETURNING
			id,
			phone_number,
			status,
			phone_verified,
			created_at,
			updated_at
	`

	user := &model.User{}

	err := r.db.QueryRow(ctx, query, phoneNumber).Scan(
		&user.ID,
		&user.PhoneNumber,
		&user.Status,
		&user.PhoneVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *AuthRepository) MarkPhoneVerified(
	ctx context.Context,
	userID string,
) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users
		SET phone_verified = TRUE, updated_at = NOW()
		WHERE id = $1
	`, userID)
	return err
}

func (r *AuthRepository) SaveOTP(
	ctx context.Context,
	phoneNumber string,
	otpHash string,
	expiresAt time.Time,
) error {
	// Start a transaction to ensure atomicity
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Invalidate all previous active OTPs for this phone number
	invalidateQuery := `
		UPDATE otp_verifications
		SET is_active = FALSE
		WHERE phone_number = $1 AND is_active = TRUE
	`
	_, err = tx.Exec(ctx, invalidateQuery, phoneNumber)
	if err != nil {
		return err
	}

	// Insert the new OTP
	insertQuery := `
		INSERT INTO otp_verifications (
			phone_number,
			otp_hash,
			expires_at,
			is_active
		)
		VALUES ($1, $2, $3, TRUE)
	`
	_, err = tx.Exec(ctx, insertQuery, phoneNumber, otpHash, expiresAt)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *AuthRepository) VerifyOTP(
	ctx context.Context,
	phoneNumber string,
	otpHash string,
) (bool, error) {
	query := `
		SELECT id, otp_hash, expires_at, attempts, verified_at
		FROM otp_verifications
		WHERE phone_number = $1 AND is_active = TRUE
		ORDER BY created_at DESC
		LIMIT 1
	`

	var (
		id         string
		storedHash string
		expiresAt  time.Time
		attempts   int
		verifiedAt *time.Time
	)

	err := r.db.QueryRow(ctx, query, phoneNumber).Scan(
		&id,
		&storedHash,
		&expiresAt,
		&attempts,
		&verifiedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	if verifiedAt != nil {
		return false, nil
	}

	if attempts >= maxOTPAttempts {
		return false, nil
	}

	if time.Now().After(expiresAt) {
		return false, nil
	}

	if storedHash != otpHash {
		_, err = r.db.Exec(ctx,
			`UPDATE otp_verifications SET attempts = attempts + 1 WHERE id = $1`,
			id,
		)
		if err != nil {
			return false, err
		}
		return false, nil
	}

	now := time.Now()
	_, err = r.db.Exec(ctx,
		`UPDATE otp_verifications SET verified_at = $1 WHERE id = $2`,
		now,
		id,
	)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetOTPRequestCount returns the count of OTP requests made in the last windowMinutes
func (r *AuthRepository) GetOTPRequestCount(
	ctx context.Context,
	phoneNumber string,
	windowMinutes int,
) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM otp_verifications
		WHERE phone_number = $1
		AND created_at > NOW() - INTERVAL '1 minute' * $2
	`

	var count int
	err := r.db.QueryRow(ctx, query, phoneNumber, windowMinutes).Scan(&count)
	return count, err
}

// GetLastOTPRequestTime returns the creation time of the most recent OTP request
func (r *AuthRepository) GetLastOTPRequestTime(
	ctx context.Context,
	phoneNumber string,
) (time.Time, error) {
	query := `
		SELECT created_at
		FROM otp_verifications
		WHERE phone_number = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var createdAt time.Time
	err := r.db.QueryRow(ctx, query, phoneNumber).Scan(&createdAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, nil
	}

	return createdAt, err
}

func (r *AuthRepository) SaveRefreshToken(
	ctx context.Context,
	userID string,
	tokenHash string,
	expiresAt time.Time,
) error {
	query := `
		INSERT INTO refresh_tokens (
			user_id,
			token_hash,
			expires_at
		)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.Exec(ctx, query, userID, tokenHash, expiresAt)
	return err
}

func (r *AuthRepository) FindRefreshToken(
	ctx context.Context,
	tokenHash string,
) (string, time.Time, *time.Time, error) {
	query := `
		SELECT user_id, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var userID string
	var expiresAt time.Time
	var revokedAt *time.Time

	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&userID,
		&expiresAt,
		&revokedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, nil, pgx.ErrNoRows
	}

	if err != nil {
		return "", time.Time{}, nil, err
	}

	return userID, expiresAt, revokedAt, nil
}

func (r *AuthRepository) RevokeRefreshToken(
	ctx context.Context,
	tokenHash string,
) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`

	_, err := r.db.Exec(ctx, query, tokenHash)
	return err
}

func (r *AuthRepository) FindUserByID(
	ctx context.Context,
	userID string,
) (*model.User, error) {
	query := `
		SELECT
			id,
			phone_number,
			status,
			phone_verified,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
	`

	user := &model.User{}

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.PhoneNumber,
		&user.Status,
		&user.PhoneVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}
