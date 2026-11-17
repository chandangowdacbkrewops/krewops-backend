package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RateOption struct {
	RateBasis string
	RateUnit  string
}

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
) ([]RateOption, error) {
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

func RateOptionAllowed(options []RateOption, basis, unit string) bool {
	for _, option := range options {
		if option.RateBasis == basis && option.RateUnit == unit {
			return true
		}
	}
	return false
}

func (r *RateTypeRepository) listByScope(ctx context.Context, column, id string) ([]RateOption, error) {
	rows, err := r.db.Query(ctx, `
		SELECT rb.code, COALESCE(ru.code, '')
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

func (r *RateTypeRepository) listGlobal(ctx context.Context) ([]RateOption, error) {
	rows, err := r.db.Query(ctx, `
		SELECT rb.code, COALESCE(ru.code, '')
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

func scanRateOptions(rows pgx.Rows) ([]RateOption, error) {
	options := []RateOption{}
	for rows.Next() {
		var option RateOption
		if err := rows.Scan(&option.RateBasis, &option.RateUnit); err != nil {
			return nil, err
		}
		options = append(options, option)
	}
	return options, rows.Err()
}
