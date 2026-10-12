package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
)

type RateTypeRepository struct {
	db *pgxpool.Pool
}

func NewRateTypeRepository(db *pgxpool.Pool) *RateTypeRepository {
	return &RateTypeRepository{db: db}
}

func (r *RateTypeRepository) Resolve(
	ctx context.Context,
	workTypeID string,
	segmentID string,
	categoryID string,
) ([]model.RateOption, error) {
	if workTypeID != "" {
		options, err := r.listByScope(ctx, "work_type_id", workTypeID)
		if err != nil {
			return nil, err
		}
		if len(options) > 0 {
			return options, nil
		}
		if segmentID == "" || categoryID == "" {
			var foundSegmentID, foundCategoryID string
			err := r.db.QueryRow(ctx, `
				SELECT COALESCE(segment_id::text, ''), COALESCE(category_id::text, '')
				FROM categories
				WHERE id = $1 AND node_type = 'WORK_TYPE'
			`, workTypeID).Scan(&foundSegmentID, &foundCategoryID)
			if err != nil && err != pgx.ErrNoRows {
				return nil, err
			}
			if segmentID == "" {
				segmentID = foundSegmentID
			}
			if categoryID == "" {
				categoryID = foundCategoryID
			}
		}
	}

	if segmentID != "" {
		options, err := r.listByScope(ctx, "work_segment_id", segmentID)
		if err != nil {
			return nil, err
		}
		if len(options) > 0 {
			return options, nil
		}
		if categoryID == "" {
			var foundCategoryID string
			err := r.db.QueryRow(ctx, `
				SELECT COALESCE(category_id::text, '')
				FROM categories
				WHERE id = $1 AND node_type = 'GROUP'
			`, segmentID).Scan(&foundCategoryID)
			if err != nil && err != pgx.ErrNoRows {
				return nil, err
			}
			categoryID = foundCategoryID
		}
	}

	if categoryID != "" {
		options, err := r.listByScope(ctx, "work_category_id", categoryID)
		if err != nil {
			return nil, err
		}
		if len(options) > 0 {
			return options, nil
		}
	}

	return r.listGlobal(ctx)
}

func (r *RateTypeRepository) ExistsPair(ctx context.Context, basis, unit string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM rate_bases rb
			WHERE rb.code = $1
				AND rb.is_active = TRUE
				AND (
					($2 = '' AND rb.code = 'negotiable')
					OR EXISTS (
						SELECT 1
						FROM rate_units ru
						WHERE ru.code = $2
							AND ru.is_active = TRUE
					)
				)
		)
	`, basis, unit).Scan(&exists)
	return exists, err
}

func (r *RateTypeRepository) listByScope(ctx context.Context, column, id string) ([]model.RateOption, error) {
	rows, err := r.db.Query(ctx, `
		SELECT rb.code, rb.name, ru.code, ru.name, m.is_default
		FROM work_rate_type_mappings m
		JOIN rate_bases rb ON rb.id = m.rate_basis_id
		LEFT JOIN rate_units ru ON ru.id = m.rate_unit_id
		WHERE m.`+column+` = $1
			AND rb.is_active = TRUE
			AND (m.rate_unit_id IS NULL OR ru.is_active = TRUE)
		ORDER BY m.is_default DESC, rb.name ASC, ru.name ASC NULLS LAST
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRateOptions(rows)
}

func (r *RateTypeRepository) listGlobal(ctx context.Context) ([]model.RateOption, error) {
	rows, err := r.db.Query(ctx, `
		SELECT rb.code, rb.name, ru.code, ru.name, m.is_default
		FROM work_rate_type_mappings m
		JOIN rate_bases rb ON rb.id = m.rate_basis_id
		LEFT JOIN rate_units ru ON ru.id = m.rate_unit_id
		WHERE m.work_type_id IS NULL
			AND m.work_segment_id IS NULL
			AND m.work_category_id IS NULL
			AND rb.is_active = TRUE
			AND (m.rate_unit_id IS NULL OR ru.is_active = TRUE)
		ORDER BY m.is_default DESC, rb.name ASC, ru.name ASC NULLS LAST
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRateOptions(rows)
}

func scanRateOptions(rows pgx.Rows) ([]model.RateOption, error) {
	options := []model.RateOption{}
	for rows.Next() {
		var option model.RateOption
		var unitCode, unitName *string
		if err := rows.Scan(
			&option.RateBasis,
			&option.RateBasisName,
			&unitCode,
			&unitName,
			&option.IsDefault,
		); err != nil {
			return nil, err
		}
		option.RateUnit = unitCode
		option.RateUnitName = unitName
		option.Label = model.RateOptionLabel(option.RateBasis, option.RateUnit)
		options = append(options, option)
	}
	return options, rows.Err()
}
