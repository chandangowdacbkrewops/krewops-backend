package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
)

type WorkCategoryRepository struct {
	db *pgxpool.Pool
}

func NewWorkCategoryRepository(db *pgxpool.Pool) *WorkCategoryRepository {
	return &WorkCategoryRepository{db: db}
}

func (r *WorkCategoryRepository) ListAll(ctx context.Context) ([]model.WorkCategory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, code, name, description, display_order, is_active, created_at, updated_at
		FROM work_categories
		WHERE is_active = TRUE
		ORDER BY display_order ASC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []model.WorkCategory{}
	for rows.Next() {
		var wc model.WorkCategory
		if err := rows.Scan(
			&wc.ID,
			&wc.Code,
			&wc.Name,
			&wc.Description,
			&wc.DisplayOrder,
			&wc.IsActive,
			&wc.CreatedAt,
			&wc.UpdatedAt,
		); err != nil {
			return nil, err
		}
		categories = append(categories, wc)
	}

	return categories, rows.Err()
}
