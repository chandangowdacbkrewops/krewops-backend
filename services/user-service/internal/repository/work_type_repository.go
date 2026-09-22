package repository

import (
	"context"

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
		SELECT id, name
		FROM work_types
		WHERE is_active = TRUE
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	types := []model.WorkType{}
	for rows.Next() {
		var wt model.WorkType
		if err := rows.Scan(&wt.ID, &wt.Name); err != nil {
			return nil, err
		}
		types = append(types, wt)
	}

	return types, rows.Err()
}
