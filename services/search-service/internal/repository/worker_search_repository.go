package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/model"
)

// WorkerSearchRepository issues read-only search queries against the
// profiles / worker_profiles / worker_work_types tables (owned by
// user-service) and the shared work_types lookup table. It intentionally
// does not write to any of these tables.
type WorkerSearchRepository struct {
	db *pgxpool.Pool
}

func NewWorkerSearchRepository(db *pgxpool.Pool) *WorkerSearchRepository {
	return &WorkerSearchRepository{db: db}
}

func (r *WorkerSearchRepository) SearchWorkers(
	ctx context.Context,
	filters model.WorkerSearchFilters,
) (*model.WorkerSearchResponse, error) {
	conditions := []string{"1 = 1"}
	args := make([]interface{}, 0, 8)

	addCondition := func(clause string, value interface{}) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(clause, len(args)))
	}

	if v := trimmed(filters.WorkTypeID); v != "" {
		addCondition(`EXISTS (
			SELECT 1 FROM worker_work_types wwt
			WHERE wwt.worker_profile_id = wp.id AND wwt.work_type_id = $%d AND wwt.is_active = TRUE
		)`, v)
	}
	if v := trimmed(filters.City); v != "" {
		addCondition("LOWER(p.city) = LOWER($%d)", v)
	}
	if v := trimmed(filters.State); v != "" {
		addCondition("LOWER(p.state) = LOWER($%d)", v)
	}
	if v := trimmed(filters.WorkerType); v != "" {
		addCondition("wp.worker_type = $%d", v)
	}
	if v := trimmed(filters.AvailabilityStatus); v != "" {
		addCondition("wp.availability_status = $%d", v)
	}
	if filters.MinExperienceYears != nil {
		addCondition("wp.experience_years >= $%d", *filters.MinExperienceYears)
	}
	if v := trimmed(filters.Keyword); v != "" {
		args = append(args, "%"+v+"%")
		placeholder := len(args)
		conditions = append(conditions, fmt.Sprintf(
			"(p.first_name ILIKE $%d OR p.last_name ILIKE $%d OR wp.crew_name ILIKE $%d OR wp.bio ILIKE $%d)",
			placeholder, placeholder, placeholder, placeholder,
		))
	}

	page, pageSize := model.NormalizePagination(filters.Page, filters.PageSize)
	offset := (page - 1) * pageSize

	limitPlaceholder := len(args) + 1
	offsetPlaceholder := len(args) + 2
	args = append(args, pageSize, offset)

	query := fmt.Sprintf(`
		SELECT
			wp.id,
			wp.user_id,
			p.first_name,
			p.last_name,
			p.city,
			p.state,
			wp.worker_type,
			wp.crew_name,
			wp.crew_size,
			wp.experience_years,
			wp.expected_rate,
			wp.rate_type,
			wp.availability_status,
			wp.verification_status,
			wp.bio,
			COUNT(*) OVER() AS total_count
		FROM worker_profiles wp
		JOIN profiles p ON p.user_id = wp.user_id
		WHERE %s
		ORDER BY wp.created_at DESC
		LIMIT $%d OFFSET $%d
	`, strings.Join(conditions, " AND "), limitPlaceholder, offsetPlaceholder)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	response := &model.WorkerSearchResponse{
		Results:  []model.WorkerSearchResult{},
		Page:     page,
		PageSize: pageSize,
	}

	workerIDs := make([]string, 0)
	byID := make(map[string]*model.WorkerSearchResult)

	for rows.Next() {
		var item model.WorkerSearchResult
		var total int

		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.FirstName,
			&item.LastName,
			&item.City,
			&item.State,
			&item.WorkerType,
			&item.CrewName,
			&item.CrewSize,
			&item.ExperienceYears,
			&item.ExpectedRate,
			&item.RateType,
			&item.AvailabilityStatus,
			&item.VerificationStatus,
			&item.Bio,
			&total,
		); err != nil {
			rows.Close()
			return nil, err
		}

		item.Skills = []model.WorkerSkillResult{}
		response.Total = total
		response.Results = append(response.Results, item)
		workerIDs = append(workerIDs, item.ID)
	}
	rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range response.Results {
		byID[response.Results[i].ID] = &response.Results[i]
	}

	if len(workerIDs) > 0 {
		if err := r.attachSkills(ctx, workerIDs, byID); err != nil {
			return nil, err
		}
	}

	return response, nil
}

func (r *WorkerSearchRepository) attachSkills(
	ctx context.Context,
	workerProfileIDs []string,
	byID map[string]*model.WorkerSearchResult,
) error {
	rows, err := r.db.Query(ctx, `
		SELECT wwt.worker_profile_id, wt.id, wt.name, wwt.years_experience
		FROM worker_work_types wwt
		JOIN work_types wt ON wt.id = wwt.work_type_id
		WHERE wwt.worker_profile_id = ANY($1) AND wwt.is_active = TRUE
		ORDER BY wwt.is_primary DESC, wwt.created_at
	`, workerProfileIDs)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var workerProfileID string
		var skill model.WorkerSkillResult

		if err := rows.Scan(
			&workerProfileID,
			&skill.WorkTypeID,
			&skill.WorkTypeName,
			&skill.ExperienceYears,
		); err != nil {
			return err
		}

		if result, ok := byID[workerProfileID]; ok {
			result.Skills = append(result.Skills, skill)
		}
	}

	return rows.Err()
}
