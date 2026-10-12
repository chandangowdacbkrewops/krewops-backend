package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
)

type WorkSegmentRepository struct {
	db *pgxpool.Pool
}

func NewWorkSegmentRepository(db *pgxpool.Pool) *WorkSegmentRepository {
	return &WorkSegmentRepository{db: db}
}

func (r *WorkSegmentRepository) ListByCategoryID(ctx context.Context, categoryID string) ([]model.WorkSegment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, category_id, code, name, description, display_order, is_active, created_at, updated_at
		FROM categories
		WHERE category_id = $1 AND node_type = 'GROUP' AND is_active = TRUE
		ORDER BY display_order ASC, name ASC
	`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanWorkSegments(rows)
}

func (r *WorkSegmentRepository) FindByID(ctx context.Context, id string) (*model.WorkSegment, error) {
	var segment model.WorkSegment
	err := r.db.QueryRow(ctx, `
		SELECT id, category_id, code, name, description, display_order, is_active, created_at, updated_at
		FROM categories
		WHERE id = $1 AND node_type = 'GROUP'
	`, id).Scan(
		&segment.ID,
		&segment.CategoryID,
		&segment.Code,
		&segment.Name,
		&segment.Description,
		&segment.DisplayOrder,
		&segment.IsActive,
		&segment.CreatedAt,
		&segment.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &segment, nil
}

func (r *WorkSegmentRepository) FindByIDs(ctx context.Context, ids []string) (map[string]model.WorkSegment, error) {
	results := make(map[string]model.WorkSegment, len(ids))
	if len(ids) == 0 {
		return results, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, category_id, code, name, description, display_order, is_active, created_at, updated_at
		FROM categories
		WHERE id = ANY($1) AND node_type = 'GROUP'
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	segments, err := scanWorkSegments(rows)
	if err != nil {
		return nil, err
	}
	for _, segment := range segments {
		results[segment.ID] = segment
	}
	return results, nil
}

func scanWorkSegments(rows pgx.Rows) ([]model.WorkSegment, error) {
	segments := []model.WorkSegment{}
	for rows.Next() {
		var segment model.WorkSegment
		if err := rows.Scan(
			&segment.ID,
			&segment.CategoryID,
			&segment.Code,
			&segment.Name,
			&segment.Description,
			&segment.DisplayOrder,
			&segment.IsActive,
			&segment.CreatedAt,
			&segment.UpdatedAt,
		); err != nil {
			return nil, err
		}
		segments = append(segments, segment)
	}
	return segments, rows.Err()
}
