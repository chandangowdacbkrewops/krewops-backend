package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
)

type WorkTypeFieldRepository struct {
	db *pgxpool.Pool
}

func NewWorkTypeFieldRepository(db *pgxpool.Pool) *WorkTypeFieldRepository {
	return &WorkTypeFieldRepository{db: db}
}

func (r *WorkTypeFieldRepository) ListActiveByWorkTypeID(
	ctx context.Context,
	workTypeID string,
) ([]model.WorkTypeField, error) {
	rows, err := r.db.Query(ctx, `
		SELECT field_key, is_required
		FROM work_type_fields
		WHERE work_type_id = $1 AND is_active = TRUE
		ORDER BY display_order ASC, label ASC
	`, workTypeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fields := []model.WorkTypeField{}
	for rows.Next() {
		var field model.WorkTypeField
		if err := rows.Scan(&field.FieldKey, &field.IsRequired); err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}

	return fields, rows.Err()
}
