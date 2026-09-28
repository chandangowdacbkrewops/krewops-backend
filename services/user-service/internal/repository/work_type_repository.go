package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
)

type WorkTypeRepository struct {
	db *pgxpool.Pool
}

func NewWorkTypeRepository(db *pgxpool.Pool) *WorkTypeRepository {
	return &WorkTypeRepository{db: db}
}

func (r *WorkTypeRepository) ListAll(ctx context.Context) ([]model.WorkType, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, category_id, code, name, description, display_order, is_active, created_at, updated_at
		FROM work_types
		WHERE is_active = TRUE
		ORDER BY display_order ASC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanWorkTypes(rows)
}

// ListByCategoryID returns the active work types belonging to a single
// category, ordered by display_order then name.
func (r *WorkTypeRepository) ListByCategoryID(ctx context.Context, categoryID string) ([]model.WorkType, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, category_id, code, name, description, display_order, is_active, created_at, updated_at
		FROM work_types
		WHERE category_id = $1 AND is_active = TRUE
		ORDER BY display_order ASC, name ASC
	`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanWorkTypes(rows)
}

func scanWorkTypes(rows pgx.Rows) ([]model.WorkType, error) {
	types := []model.WorkType{}
	for rows.Next() {
		var wt model.WorkType
		if err := rows.Scan(
			&wt.ID,
			&wt.CategoryID,
			&wt.Code,
			&wt.Name,
			&wt.Description,
			&wt.DisplayOrder,
			&wt.IsActive,
			&wt.CreatedAt,
			&wt.UpdatedAt,
		); err != nil {
			return nil, err
		}
		types = append(types, wt)
	}

	return types, rows.Err()
}

func (r *WorkTypeRepository) FindByIDs(ctx context.Context, ids []string) (map[string]model.WorkType, error) {
	results := make(map[string]model.WorkType, len(ids))
	if len(ids) == 0 {
		return results, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, category_id, code, name, description, display_order, is_active, created_at, updated_at
		FROM work_types
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	types, err := scanWorkTypes(rows)
	if err != nil {
		return nil, err
	}
	for _, wt := range types {
		results[wt.ID] = wt
	}
	return results, nil
}
