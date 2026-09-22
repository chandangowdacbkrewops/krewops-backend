package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/model"
)

// WorkSearchRepository issues read-only search queries against the
// work_postings table (owned by work-service) and the shared work_types
// lookup table. It intentionally does not write to either table.
type WorkSearchRepository struct {
	db *pgxpool.Pool
}

func NewWorkSearchRepository(db *pgxpool.Pool) *WorkSearchRepository {
	return &WorkSearchRepository{db: db}
}

func (r *WorkSearchRepository) SearchWork(
	ctx context.Context,
	filters model.WorkSearchFilters,
) (*model.WorkSearchResponse, error) {
	conditions := []string{"wp.status = 'published'"}
	args := make([]interface{}, 0, 8)

	addCondition := func(clause string, value interface{}) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(clause, len(args)))
	}

	if v := trimmed(filters.WorkTypeID); v != "" {
		addCondition("wp.work_type_id = $%d", v)
	}
	if v := trimmed(filters.City); v != "" {
		addCondition("LOWER(wp.city) = LOWER($%d)", v)
	}
	if v := trimmed(filters.State); v != "" {
		addCondition("LOWER(wp.state) = LOWER($%d)", v)
	}
	if v := trimmed(filters.ExperienceLevel); v != "" {
		addCondition("wp.experience_level = $%d", v)
	}
	if v := trimmed(filters.PaymentType); v != "" {
		addCondition("wp.payment_type = $%d", v)
	}
	if filters.MinBudgetRate != nil {
		addCondition("wp.budget_rate >= $%d", *filters.MinBudgetRate)
	}
	if filters.MaxBudgetRate != nil {
		addCondition("wp.budget_rate <= $%d", *filters.MaxBudgetRate)
	}
	if v := trimmed(filters.Keyword); v != "" {
		args = append(args, "%"+v+"%")
		placeholder := len(args)
		conditions = append(conditions, fmt.Sprintf(
			"(wp.title ILIKE $%d OR wp.description ILIKE $%d)",
			placeholder, placeholder,
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
			wp.status,
			wp.title,
			wp.work_type_id,
			wt.name,
			wp.description,
			wp.address,
			wp.city,
			wp.state,
			wp.latitude,
			wp.longitude,
			wp.workers_needed,
			wp.experience_level,
			wp.skills,
			wp.start_date,
			wp.duration_value,
			wp.duration_unit,
			wp.shift_timing,
			wp.payment_type,
			wp.budget_rate,
			wp.published_at,
			wp.created_at,
			COUNT(*) OVER() AS total_count
		FROM work_postings wp
		LEFT JOIN work_types wt ON wt.id = wp.work_type_id
		WHERE %s
		ORDER BY wp.published_at DESC NULLS LAST, wp.created_at DESC
		LIMIT $%d OFFSET $%d
	`, strings.Join(conditions, " AND "), limitPlaceholder, offsetPlaceholder)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	response := &model.WorkSearchResponse{
		Results:  []model.WorkSearchResult{},
		Page:     page,
		PageSize: pageSize,
	}

	for rows.Next() {
		var item model.WorkSearchResult
		var total int

		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Status,
			&item.Title,
			&item.WorkTypeID,
			&item.WorkTypeName,
			&item.Description,
			&item.Address,
			&item.City,
			&item.State,
			&item.Latitude,
			&item.Longitude,
			&item.WorkersNeeded,
			&item.ExperienceLevel,
			&item.Skills,
			&item.StartDate,
			&item.DurationValue,
			&item.DurationUnit,
			&item.ShiftTiming,
			&item.PaymentType,
			&item.BudgetRate,
			&item.PublishedAt,
			&item.CreatedAt,
			&total,
		); err != nil {
			return nil, err
		}

		response.Total = total
		response.Results = append(response.Results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return response, nil
}

func trimmed(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
