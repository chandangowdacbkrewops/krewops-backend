package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
)

type WorkCategoryRepository struct {
	db *pgxpool.Pool
}

func NewWorkCategoryRepository(db *pgxpool.Pool) *WorkCategoryRepository {
	return &WorkCategoryRepository{db: db}
}

func (r *WorkCategoryRepository) FindByID(ctx context.Context, id string) (*model.WorkCategory, error) {
	category := &model.WorkCategory{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name
		FROM work_categories
		WHERE id = $1 AND is_active = TRUE
	`, id).Scan(&category.ID, &category.Name)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return category, nil
}
