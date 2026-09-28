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
	ErrProfileAlreadyExists   = errors.New("profile already exists")
	ErrProfileNotFound        = errors.New("profile not found")
	ErrWorkerProfileNotFound  = errors.New("worker profile not found")
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

	if _, err = tx.Exec(ctx, `DELETE FROM worker_skills WHERE worker_profile_id = $1`, workerProfileID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM worker_work_types WHERE worker_profile_id = $1`, workerProfileID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM worker_work_categories WHERE worker_profile_id = $1`, workerProfileID); err != nil {
		return nil, err
	}

	primarySet := false
	for _, selection := range req.Selections {
		if _, err = tx.Exec(ctx, `
			INSERT INTO worker_work_categories (worker_profile_id, work_category_id)
			VALUES ($1, $2)
		`, workerProfileID, selection.WorkCategoryID); err != nil {
			return nil, err
		}

		for _, workType := range selection.WorkTypes {
			isPrimary := !primarySet
			if isPrimary {
				primarySet = true
			}
			if _, err = tx.Exec(ctx, `
				INSERT INTO worker_work_types (worker_profile_id, work_type_id, is_primary)
				VALUES ($1, $2, $3)
			`, workerProfileID, workType.WorkTypeID, isPrimary); err != nil {
				return nil, err
			}
			for _, skillID := range workType.SkillIDs {
				if _, err = tx.Exec(ctx, `
					INSERT INTO worker_skills (worker_profile_id, skill_id)
					VALUES ($1, $2)
				`, workerProfileID, skillID); err != nil {
					return nil, err
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.FindWorkerProfileByUserID(ctx, authUserID)
}

func (r *ProfileRepository) UpdateWorkerSelections(
	ctx context.Context,
	authUserID string,
	selections []model.WorkerCategorySelection,
) (*model.WorkerProfile, error) {
	profile, err := r.FindWorkerProfileByUserID(ctx, authUserID)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx, `DELETE FROM worker_skills WHERE worker_profile_id = $1`, profile.ID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM worker_work_types WHERE worker_profile_id = $1`, profile.ID); err != nil {
		return nil, err
	}

	primarySet := false
	for _, selection := range selections {
		for _, workType := range selection.WorkTypes {
			isPrimary := !primarySet
			if isPrimary {
				primarySet = true
			}
			if _, err = tx.Exec(ctx, `
				INSERT INTO worker_work_types (worker_profile_id, work_type_id, is_primary)
				VALUES ($1, $2, $3)
			`, profile.ID, workType.WorkTypeID, isPrimary); err != nil {
				return nil, err
			}
			for _, skillID := range workType.SkillIDs {
				if _, err = tx.Exec(ctx, `
					INSERT INTO worker_skills (worker_profile_id, skill_id)
					VALUES ($1, $2)
				`, profile.ID, skillID); err != nil {
					return nil, err
				}
			}
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
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWorkerProfileNotFound
	}
	if err != nil {
		return nil, err
	}

	categories, err := r.loadWorkerSelections(ctx, profile.ID)
	if err != nil {
		return nil, err
	}
	profile.WorkCategories = categories
	if len(categories) > 0 {
		profile.WorkCategory = &model.WorkerProfileCategory{
			WorkerProfileID: profile.ID,
			WorkCategoryID:  categories[0].WorkCategoryID,
			Code:            categories[0].Code,
			Name:            categories[0].Name,
			IsActive:        true,
		}
	}
	return profile, nil
}

func (r *ProfileRepository) loadWorkerSelections(
	ctx context.Context,
	workerProfileID string,
) ([]model.WorkerWorkCategorySummary, error) {
	categoryRows, err := r.db.Query(ctx, `
		SELECT wwc.work_category_id, wc.code, wc.name
		FROM worker_work_categories wwc
		JOIN work_categories wc ON wc.id = wwc.work_category_id
		WHERE wwc.worker_profile_id = $1 AND wwc.is_active = TRUE
		ORDER BY wwc.created_at ASC
	`, workerProfileID)
	if err != nil {
		return nil, err
	}
	defer categoryRows.Close()

	categoryOrder := []string{}
	categoriesByID := map[string]model.WorkerWorkCategorySummary{}
	for categoryRows.Next() {
		var item model.WorkerWorkCategorySummary
		if err := categoryRows.Scan(&item.WorkCategoryID, &item.Code, &item.Name); err != nil {
			return nil, err
		}
		item.WorkTypes = []model.WorkerWorkTypeSummary{}
		categoryOrder = append(categoryOrder, item.WorkCategoryID)
		categoriesByID[item.WorkCategoryID] = item
	}
	if err := categoryRows.Err(); err != nil {
		return nil, err
	}

	typeRows, err := r.db.Query(ctx, `
		SELECT wwt.work_type_id, wt.name, wt.category_id, wwt.is_primary
		FROM worker_work_types wwt
		JOIN work_types wt ON wt.id = wwt.work_type_id
		WHERE wwt.worker_profile_id = $1 AND wwt.is_active = TRUE
		ORDER BY wwt.is_primary DESC, wwt.created_at ASC
	`, workerProfileID)
	if err != nil {
		return nil, err
	}
	defer typeRows.Close()

	workTypeOrder := map[string][]string{}
	workTypesByID := map[string]model.WorkerWorkTypeSummary{}
	for typeRows.Next() {
		var item model.WorkerWorkTypeSummary
		if err := typeRows.Scan(&item.WorkTypeID, &item.Name, &item.CategoryID, &item.IsPrimary); err != nil {
			return nil, err
		}
		item.Skills = []model.WorkerSkillSummary{}
		workTypesByID[item.WorkTypeID] = item
		workTypeOrder[item.CategoryID] = append(workTypeOrder[item.CategoryID], item.WorkTypeID)
	}
	if err := typeRows.Err(); err != nil {
		return nil, err
	}

	skillRows, err := r.db.Query(ctx, `
		SELECT s.id, s.work_type_id, s.code, s.name
		FROM worker_skills ws
		JOIN skills s ON s.id = ws.skill_id
		WHERE ws.worker_profile_id = $1 AND ws.is_active = TRUE
		ORDER BY s.name ASC
	`, workerProfileID)
	if err != nil {
		return nil, err
	}
	defer skillRows.Close()

	for skillRows.Next() {
		var skill model.WorkerSkillSummary
		if err := skillRows.Scan(&skill.ID, &skill.WorkTypeID, &skill.Code, &skill.Name); err != nil {
			return nil, err
		}
		workType, ok := workTypesByID[skill.WorkTypeID]
		if !ok {
			continue
		}
		workType.Skills = append(workType.Skills, skill)
		workTypesByID[skill.WorkTypeID] = workType
	}
	if err := skillRows.Err(); err != nil {
		return nil, err
	}

	categories := make([]model.WorkerWorkCategorySummary, 0, len(categoryOrder))
	for _, categoryID := range categoryOrder {
		category := categoriesByID[categoryID]
		for _, workTypeID := range workTypeOrder[categoryID] {
			category.WorkTypes = append(category.WorkTypes, workTypesByID[workTypeID])
		}
		categories = append(categories, category)
	}
	return categories, nil
}
