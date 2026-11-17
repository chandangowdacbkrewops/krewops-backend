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
	ErrProfileAlreadyExists  = errors.New("profile already exists")
	ErrProfileNotFound       = errors.New("profile not found")
	ErrWorkerProfileNotFound = errors.New("worker profile not found")
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
			gender,
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
		&profile.Gender,
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
			gender,
			user_type,
			country,
			state,
			city,
			postal_code,
			profile_completed,
			onboarding_completed
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE, TRUE)
		ON CONFLICT (user_id) DO NOTHING
		RETURNING
			id,
			user_id,
			first_name,
			last_name,
			gender,
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
	`, authUserID, req.FirstName, req.LastName, req.Gender, req.UserType, req.Country, req.State, req.City, req.PostalCode).Scan(
		&profile.ID,
		&profile.AuthUserID,
		&profile.FirstName,
		&profile.LastName,
		&profile.Gender,
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
			gender = $4,
			country = $5,
			state = $6,
			city = $7,
			postal_code = $8,
			updated_at = NOW()
		WHERE user_id = $1
		RETURNING
			id, user_id, first_name, last_name, gender, user_type, date_of_birth,
			email, country, state, city, postal_code, preferred_language,
			profile_completed, onboarding_completed, created_at, updated_at
	`, authUserID, req.FirstName, req.LastName, req.Gender, req.Country, req.State, req.City, req.PostalCode).Scan(
		&profile.ID,
		&profile.AuthUserID,
		&profile.FirstName,
		&profile.LastName,
		&profile.Gender,
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
			expected_rate, expected_rate_min, expected_rate_max, rate_basis, rate_unit,
			availability_status, bio, profile_completed
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, COALESCE($11, 'available'), $12, TRUE)
		ON CONFLICT (user_id) DO UPDATE SET
			worker_type = EXCLUDED.worker_type,
			crew_name = EXCLUDED.crew_name,
			crew_size = EXCLUDED.crew_size,
			experience_years = EXCLUDED.experience_years,
			expected_rate = EXCLUDED.expected_rate,
			expected_rate_min = EXCLUDED.expected_rate_min,
			expected_rate_max = EXCLUDED.expected_rate_max,
			rate_basis = EXCLUDED.rate_basis,
			rate_unit = EXCLUDED.rate_unit,
			availability_status = EXCLUDED.availability_status,
			bio = EXCLUDED.bio,
			profile_completed = TRUE,
			updated_at = NOW()
		RETURNING id
	`, authUserID, req.WorkerType, req.CrewName, req.CrewSize, req.ExperienceYears, req.ExpectedRate, req.ExpectedRateMin, req.ExpectedRateMax, req.RateBasis, req.RateUnit, req.AvailabilityStatus, req.Bio).Scan(&workerProfileID)
	if err != nil {
		return nil, err
	}

	if err := replaceWorkerSelections(ctx, tx, workerProfileID, req.Selections); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.FindWorkerProfileByUserID(ctx, authUserID)
}

func (r *ProfileRepository) UpdateWorkerSelections(
	ctx context.Context,
	authUserID string,
	req model.UpdateWorkerProfileRequest,
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

	if req.RateBasis != nil || req.RateUnit != nil {
		if _, err = tx.Exec(ctx, `
			UPDATE worker_profiles
			SET rate_basis = COALESCE($2, rate_basis),
			    rate_unit = $3,
			    updated_at = NOW()
			WHERE id = $1
		`, profile.ID, req.RateBasis, req.RateUnit); err != nil {
			return nil, err
		}
	}

	if err := replaceWorkerSelections(ctx, tx, profile.ID, req.Selections); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.FindWorkerProfileByUserID(ctx, authUserID)
}

type workerCategoryInsert struct {
	CategoryID string
	IsPrimary  bool
}

func flattenWorkerSelections(selections []model.WorkerCategorySelection) ([]workerCategoryInsert, []string) {
	categories := make([]workerCategoryInsert, 0)
	skillIDs := make([]string, 0)
	primarySet := false

	for _, selection := range selections {
		if selection.WorkCategoryID != "" {
			categories = append(categories, workerCategoryInsert{CategoryID: selection.WorkCategoryID})
		}
		for _, segment := range selection.Segments {
			if segment.WorkSegmentID != "" {
				categories = append(categories, workerCategoryInsert{CategoryID: segment.WorkSegmentID})
			}
			for _, workType := range segment.WorkTypes {
				if workType.WorkTypeID == "" {
					continue
				}
				isPrimary := !primarySet
				if isPrimary {
					primarySet = true
				}
				categories = append(categories, workerCategoryInsert{
					CategoryID: workType.WorkTypeID,
					IsPrimary:  isPrimary,
				})
				skillIDs = append(skillIDs, workType.SkillIDs...)
			}
		}
	}

	return categories, skillIDs
}

func replaceWorkerSelections(
	ctx context.Context,
	tx pgx.Tx,
	workerProfileID string,
	selections []model.WorkerCategorySelection,
) error {
	if _, err := tx.Exec(ctx, `DELETE FROM worker_skills WHERE worker_profile_id = $1`, workerProfileID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM worker_categories WHERE worker_profile_id = $1`, workerProfileID); err != nil {
		return err
	}

	categories, skillIDs := flattenWorkerSelections(selections)
	for _, category := range categories {
		if _, err := tx.Exec(ctx, `
			INSERT INTO worker_categories (worker_profile_id, category_id, is_primary)
			VALUES ($1, $2, $3)
		`, workerProfileID, category.CategoryID, category.IsPrimary); err != nil {
			return err
		}
	}
	for _, skillID := range skillIDs {
		if skillID == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO worker_skills (worker_profile_id, skill_id)
			VALUES ($1, $2)
		`, workerProfileID, skillID); err != nil {
			return err
		}
	}
	return nil
}

func (r *ProfileRepository) FindWorkerProfileByUserID(ctx context.Context, userID string) (*model.WorkerProfile, error) {
	profile := &model.WorkerProfile{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, worker_type, crew_name, crew_size, experience_years,
		       expected_rate, expected_rate_min, expected_rate_max, rate_basis, rate_unit,
		       availability_status, verification_status,
		       bio, profile_completed, created_at, updated_at
		FROM worker_profiles
		WHERE user_id = $1
	`, userID).Scan(
		&profile.ID, &profile.UserID, &profile.WorkerType, &profile.CrewName,
		&profile.CrewSize, &profile.ExperienceYears, &profile.ExpectedRate,
		&profile.ExpectedRateMin, &profile.ExpectedRateMax, &profile.RateBasis, &profile.RateUnit,
		&profile.AvailabilityStatus, &profile.VerificationStatus,
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
	return profile, nil
}

// loadWorkerSelections reconstructs the worker's SECTOR -> GROUP -> WORK_TYPE
// selection tree from the single generic `worker_categories` join table
// (replacing the old worker_work_categories/segments/types tables), using
// `categories.node_type` + the denormalized `category_id`/`segment_id`
// ancestor pointers to regroup rows by level.
func (r *ProfileRepository) loadWorkerSelections(
	ctx context.Context,
	workerProfileID string,
) ([]model.WorkerWorkCategorySummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.node_type, c.code, c.name, c.category_id, c.segment_id, wc.is_primary
		FROM worker_categories wc
		JOIN categories c ON c.id = wc.category_id
		WHERE wc.worker_profile_id = $1 AND wc.is_active = TRUE
		ORDER BY c.node_type ASC, wc.is_primary DESC, wc.created_at ASC
	`, workerProfileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categoryOrder := []string{}
	categoriesByID := map[string]model.WorkerWorkCategorySummary{}
	segmentOrder := map[string][]string{}
	segmentsByID := map[string]model.WorkerWorkSegmentSummary{}
	workTypeOrder := map[string][]string{}
	workTypesByID := map[string]model.WorkerWorkTypeSummary{}
	workTypeIDs := []string{}

	for rows.Next() {
		var (
			id, nodeType, code, name string
			categoryID, segmentID    *string
			isPrimary                bool
		)
		if err := rows.Scan(&id, &nodeType, &code, &name, &categoryID, &segmentID, &isPrimary); err != nil {
			return nil, err
		}

		switch nodeType {
		case "SECTOR":
			item := model.WorkerWorkCategorySummary{
				WorkCategoryID: id,
				Code:           code,
				Name:           name,
				Segments:       []model.WorkerWorkSegmentSummary{},
				WorkTypes:      []model.WorkerWorkTypeSummary{},
			}
			categoryOrder = append(categoryOrder, id)
			categoriesByID[id] = item
		case "GROUP":
			item := model.WorkerWorkSegmentSummary{
				WorkSegmentID: id,
				Code:          code,
				Name:          name,
				WorkTypes:     []model.WorkerWorkTypeSummary{},
			}
			if categoryID != nil {
				item.CategoryID = *categoryID
				segmentOrder[*categoryID] = append(segmentOrder[*categoryID], id)
			}
			segmentsByID[id] = item
		case "WORK_TYPE":
			item := model.WorkerWorkTypeSummary{
				WorkTypeID: id,
				Name:       name,
				IsPrimary:  isPrimary,
				Skills:     []model.WorkerSkillSummary{},
			}
			if categoryID != nil {
				item.CategoryID = *categoryID
			}
			if segmentID != nil {
				item.SegmentID = *segmentID
				workTypeOrder[*segmentID] = append(workTypeOrder[*segmentID], id)
			}
			workTypesByID[id] = item
			workTypeIDs = append(workTypeIDs, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(workTypeIDs) > 0 {
		// Skills are global master data; reconstruct which selected work
		// type a worker's skill belongs to via the work_type_skills join
		// table (a skill may legitimately map to more than one of the
		// worker's selected work types, in which case it's attached to
		// each).
		skillRows, err := r.db.Query(ctx, `
			SELECT wts.work_type_id, s.id, s.code, s.name
			FROM worker_skills ws
			JOIN skills s ON s.id = ws.skill_id
			JOIN work_type_skills wts ON wts.skill_id = s.id
			WHERE ws.worker_profile_id = $1 AND ws.is_active = TRUE
				AND wts.work_type_id = ANY($2)
			ORDER BY s.name ASC
		`, workerProfileID, workTypeIDs)
		if err != nil {
			return nil, err
		}
		defer skillRows.Close()

		for skillRows.Next() {
			var workTypeID string
			var skill model.WorkerSkillSummary
			if err := skillRows.Scan(&workTypeID, &skill.ID, &skill.Code, &skill.Name); err != nil {
				return nil, err
			}
			skill.WorkTypeID = workTypeID
			workType, ok := workTypesByID[workTypeID]
			if !ok {
				continue
			}
			workType.Skills = append(workType.Skills, skill)
			workTypesByID[workTypeID] = workType
		}
		if err := skillRows.Err(); err != nil {
			return nil, err
		}
	}

	categories := make([]model.WorkerWorkCategorySummary, 0, len(categoryOrder))
	for _, categoryID := range categoryOrder {
		category := categoriesByID[categoryID]
		for _, segmentID := range segmentOrder[categoryID] {
			segment := segmentsByID[segmentID]
			for _, workTypeID := range workTypeOrder[segmentID] {
				workType := workTypesByID[workTypeID]
				segment.WorkTypes = append(segment.WorkTypes, workType)
				category.WorkTypes = append(category.WorkTypes, workType)
			}
			category.Segments = append(category.Segments, segment)
		}
		categories = append(categories, category)
	}
	return categories, nil
}
