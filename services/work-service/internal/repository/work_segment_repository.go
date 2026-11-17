package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
)

type WorkSegmentRepository struct {
	db *pgxpool.Pool
}

func NewWorkSegmentRepository(db *pgxpool.Pool) *WorkSegmentRepository {
	return &WorkSegmentRepository{db: db}
}

func (r *WorkSegmentRepository) FindByID(ctx context.Context, id string) (*model.WorkSegment, error) {
	segment := &model.WorkSegment{}
	err := r.db.QueryRow(ctx, `
		SELECT id, category_id, name
		FROM categories
		WHERE id = $1 AND node_type = 'GROUP' AND is_active = TRUE
	`, id).Scan(&segment.ID, &segment.CategoryID, &segment.Name)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return segment, nil
}
