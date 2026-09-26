package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
)

type WorkTypeFieldRepository struct {
	db *pgxpool.Pool
}

func NewWorkTypeFieldRepository(db *pgxpool.Pool) *WorkTypeFieldRepository {
	return &WorkTypeFieldRepository{db: db}
}

// ListByWorkTypeID returns the active dynamic fields configured for a work
// type, ordered by display_order then label, each with its active options
// attached (populated only for SELECT/MULTI_SELECT/RADIO field types).
func (r *WorkTypeFieldRepository) ListByWorkTypeID(ctx context.Context, workTypeID string) ([]model.WorkTypeField, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, work_type_id, field_key, label, field_type, placeholder, help_text,
		       is_required, unit, min_value, max_value, min_length, max_length,
		       display_order, is_active, created_at, updated_at
		FROM work_type_fields
		WHERE work_type_id = $1 AND is_active = TRUE
		ORDER BY display_order ASC, label ASC
	`, workTypeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fields := []model.WorkTypeField{}
	fieldIDs := make([]string, 0)
	for rows.Next() {
		var f model.WorkTypeField
		if err := rows.Scan(
			&f.ID,
			&f.WorkTypeID,
			&f.FieldKey,
			&f.Label,
			&f.FieldType,
			&f.Placeholder,
			&f.HelpText,
			&f.IsRequired,
			&f.Unit,
			&f.MinValue,
			&f.MaxValue,
			&f.MinLength,
			&f.MaxLength,
			&f.DisplayOrder,
			&f.IsActive,
			&f.CreatedAt,
			&f.UpdatedAt,
		); err != nil {
			return nil, err
		}
		f.Options = []model.WorkTypeFieldOption{}
		fields = append(fields, f)
		fieldIDs = append(fieldIDs, f.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(fieldIDs) == 0 {
		return fields, nil
	}

	byID := make(map[string]*model.WorkTypeField, len(fields))
	for i := range fields {
		byID[fields[i].ID] = &fields[i]
	}

	optionRows, err := r.db.Query(ctx, `
		SELECT id, field_id, value, label, display_order, is_active, created_at, updated_at
		FROM work_type_field_options
		WHERE field_id = ANY($1) AND is_active = TRUE
		ORDER BY field_id, display_order ASC, label ASC
	`, fieldIDs)
	if err != nil {
		return nil, err
	}
	defer optionRows.Close()

	for optionRows.Next() {
		var opt model.WorkTypeFieldOption
		if err := optionRows.Scan(
			&opt.ID,
			&opt.FieldID,
			&opt.Value,
			&opt.Label,
			&opt.DisplayOrder,
			&opt.IsActive,
			&opt.CreatedAt,
			&opt.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if field, ok := byID[opt.FieldID]; ok {
			opt.FieldKey = field.FieldKey
			field.Options = append(field.Options, opt)
		}
	}

	return fields, optionRows.Err()
}
