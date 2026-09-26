package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
)

type WorkTypeRepository struct {
	db *pgxpool.Pool
}

func NewWorkTypeRepository(db *pgxpool.Pool) *WorkTypeRepository {
	return &WorkTypeRepository{db: db}
}

func (r *WorkTypeRepository) FindByID(ctx context.Context, id string) (*model.WorkType, error) {
	wt := &model.WorkType{}
	err := r.db.QueryRow(ctx, `
		SELECT id, category_id, name
		FROM work_types
		WHERE id = $1 AND is_active = TRUE
	`, id).Scan(&wt.ID, &wt.CategoryID, &wt.Name)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return wt, nil
}
