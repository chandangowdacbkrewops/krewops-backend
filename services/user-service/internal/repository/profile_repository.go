package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
)

type ProfileRepository struct {
	db *pgxpool.Pool
}

var (
	ErrProfileAlreadyExists = errors.New("profile already exists")
	ErrProfileNotFound      = errors.New("profile not found")
)

func NewProfileRepository(db *pgxpool.Pool) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) FindByAuthUserID(
	ctx context.Context,
	authUserID string,
) (*model.Profile, error) {
	query := `
		SELECT
			id,
			user_id,
			first_name,
			last_name,
			user_type,
			date_of_birth,
			email,
			country,
			state,
			city,
			postal_code,
			preferred_language,
			profile_completed,
			onboarding_completed,
			created_at,
			updated_at
		FROM profiles
		WHERE user_id = $1
	`

	profile := &model.Profile{}

	err := r.db.QueryRow(ctx, query, authUserID).Scan(
		&profile.ID,
		&profile.AuthUserID,
		&profile.FirstName,
		&profile.LastName,
		&profile.UserType,
		&profile.DateOfBirth,
		&profile.Email,
		&profile.Country,
		&profile.State,
		&profile.City,
		&profile.PostalCode,
		&profile.PreferredLanguage,
		&profile.ProfileCompleted,
		&profile.OnboardingCompleted,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (r *ProfileRepository) CreateProfile(
	ctx context.Context,
	authUserID string,
	req model.CreateProfileRequest,
) (*model.Profile, error) {
	profile := &model.Profile{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO profiles (
			user_id,
			first_name,
			last_name,
			user_type,
			country,
			state,
			city,
			postal_code,
			profile_completed,
			onboarding_completed
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, TRUE, TRUE)
		ON CONFLICT (user_id) DO NOTHING
		RETURNING
			id,
			user_id,
			first_name,
			last_name,
			user_type,
			date_of_birth,
			email,
			country,
			state,
			city,
			postal_code,
			preferred_language,
			profile_completed,
			onboarding_completed,
			created_at,
			updated_at
	`, authUserID, req.FirstName, req.LastName, req.UserType, req.Country, req.State, req.City, req.PostalCode).Scan(
		&profile.ID,
		&profile.AuthUserID,
		&profile.FirstName,
		&profile.LastName,
		&profile.UserType,
		&profile.DateOfBirth,
		&profile.Email,
		&profile.Country,
		&profile.State,
		&profile.City,
		&profile.PostalCode,
		&profile.PreferredLanguage,
		&profile.ProfileCompleted,
		&profile.OnboardingCompleted,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileAlreadyExists
	}
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (r *ProfileRepository) UpdateProfile(
	ctx context.Context,
	authUserID string,
	req model.UpdateProfileRequest,
) (*model.Profile, error) {
	profile := &model.Profile{}
	err := r.db.QueryRow(ctx, `
		UPDATE profiles
		SET first_name = $2,
			last_name = $3,
			country = $4,
			state = $5,
			city = $6,
			postal_code = $7,
			updated_at = NOW()
		WHERE user_id = $1
		RETURNING
			id, user_id, first_name, last_name, user_type, date_of_birth,
			email, country, state, city, postal_code, preferred_language,
			profile_completed, onboarding_completed, created_at, updated_at
	`, authUserID, req.FirstName, req.LastName, req.Country, req.State, req.City, req.PostalCode).Scan(
		&profile.ID,
		&profile.AuthUserID,
		&profile.FirstName,
		&profile.LastName,
		&profile.UserType,
		&profile.DateOfBirth,
		&profile.Email,
		&profile.Country,
		&profile.State,
		&profile.City,
		&profile.PostalCode,
		&profile.PreferredLanguage,
		&profile.ProfileCompleted,
		&profile.OnboardingCompleted,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (r *ProfileRepository) CreateWorkerProfile(
	ctx context.Context,
	authUserID string,
	req model.CreateWorkerProfileRequest,
) (*model.WorkerProfile, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, user_type, profile_completed, onboarding_completed)
		VALUES ($1, 'worker', TRUE, TRUE)
		ON CONFLICT (user_id) DO UPDATE SET
			user_type = 'worker',
			profile_completed = TRUE,
			onboarding_completed = TRUE,
			updated_at = NOW()
	`, authUserID); err != nil {
		return nil, err
	}

	var workerProfileID string
	err = tx.QueryRow(ctx, `
		INSERT INTO worker_profiles (
			user_id, worker_type, crew_name, crew_size, experience_years,
			expected_rate, rate_type, availability_status, bio, profile_completed
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, 'available'), $9, TRUE)
		ON CONFLICT (user_id) DO UPDATE SET
			worker_type = EXCLUDED.worker_type,
			crew_name = EXCLUDED.crew_name,
			crew_size = EXCLUDED.crew_size,
			experience_years = EXCLUDED.experience_years,
			expected_rate = EXCLUDED.expected_rate,
			rate_type = EXCLUDED.rate_type,
			availability_status = EXCLUDED.availability_status,
			bio = EXCLUDED.bio,
			profile_completed = TRUE,
			updated_at = NOW()
		RETURNING id
	`, authUserID, req.WorkerType, req.CrewName, req.CrewSize, req.ExperienceYears, req.ExpectedRate, req.RateType, req.AvailabilityStatus, req.Bio).Scan(&workerProfileID)
	if err != nil {
		return nil, err
	}

	if _, err = tx.Exec(ctx, `DELETE FROM worker_profile_skills WHERE worker_profile_id = $1`, workerProfileID); err != nil {
		return nil, err
	}
	for _, workTypeID := range req.WorkTypeIDs {
		if _, err = tx.Exec(ctx, `
			INSERT INTO worker_profile_skills (worker_profile_id, work_type_id, experience_years)
			VALUES ($1, $2, $3)
		`, workerProfileID, workTypeID, req.ExperienceYears); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.FindWorkerProfileByUserID(ctx, authUserID)
}

func (r *ProfileRepository) FindWorkerProfileByUserID(ctx context.Context, userID string) (*model.WorkerProfile, error) {
	profile := &model.WorkerProfile{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, worker_type, crew_name, crew_size, experience_years,
		       expected_rate, rate_type, availability_status, verification_status,
		       bio, profile_completed, created_at, updated_at
		FROM worker_profiles
		WHERE user_id = $1
	`, userID).Scan(
		&profile.ID, &profile.UserID, &profile.WorkerType, &profile.CrewName,
		&profile.CrewSize, &profile.ExperienceYears, &profile.ExpectedRate,
		&profile.RateType, &profile.AvailabilityStatus, &profile.VerificationStatus,
		&profile.Bio, &profile.ProfileCompleted, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, worker_profile_id, work_type_id, experience_years, created_at
		FROM worker_profile_skills
		WHERE worker_profile_id = $1
		ORDER BY created_at, id
	`, profile.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profile.Skills = []model.WorkerProfileSkill{}
	for rows.Next() {
		var skill model.WorkerProfileSkill
		if err := rows.Scan(&skill.ID, &skill.WorkerProfileID, &skill.WorkTypeID, &skill.ExperienceYears, &skill.CreatedAt); err != nil {
			return nil, err
		}
		profile.Skills = append(profile.Skills, skill)
	}
	return profile, rows.Err()
}
