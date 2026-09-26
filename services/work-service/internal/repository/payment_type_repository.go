package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentTypeRepository struct {
	db *pgxpool.Pool
}

func NewPaymentTypeRepository(db *pgxpool.Pool) *PaymentTypeRepository {
	return &PaymentTypeRepository{db: db}
}

func (r *PaymentTypeRepository) ListAllowedCodes(
	ctx context.Context,
	workTypeID string,
	categoryID string,
) ([]string, error) {
	if workTypeID != "" {
		codes, err := r.listCodes(ctx, `
			SELECT pt.code
			FROM work_type_payment_types wtpt
			JOIN payment_types pt ON pt.id = wtpt.payment_type_id
			WHERE wtpt.work_type_id = $1
				AND pt.is_active = TRUE
			ORDER BY pt.name ASC
		`, workTypeID)
		if err != nil {
			return nil, err
		}
		if len(codes) > 0 {
			return codes, nil
		}

		codes, err = r.listCodes(ctx, `
			SELECT pt.code
			FROM work_types wt
			JOIN work_category_payment_types wcpt ON wcpt.work_category_id = wt.category_id
			JOIN payment_types pt ON pt.id = wcpt.payment_type_id
			WHERE wt.id = $1
				AND pt.is_active = TRUE
			ORDER BY pt.name ASC
		`, workTypeID)
		if err != nil {
			return nil, err
		}
		if len(codes) > 0 {
			return codes, nil
		}
	}

	if categoryID != "" {
		return r.listCodes(ctx, `
			SELECT pt.code
			FROM work_category_payment_types wcpt
			JOIN payment_types pt ON pt.id = wcpt.payment_type_id
			WHERE wcpt.work_category_id = $1
				AND pt.is_active = TRUE
			ORDER BY pt.name ASC
		`, categoryID)
	}

	return []string{}, nil
}

func (r *PaymentTypeRepository) listCodes(ctx context.Context, query string, arg string) ([]string, error) {
	rows, err := r.db.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	codes := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}
