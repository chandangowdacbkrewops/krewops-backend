package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
)

type PaymentTypeRepository struct {
	db *pgxpool.Pool
}

func NewPaymentTypeRepository(db *pgxpool.Pool) *PaymentTypeRepository {
	return &PaymentTypeRepository{db: db}
}

func (r *PaymentTypeRepository) ListByCategoryID(ctx context.Context, categoryID string) ([]model.PaymentType, error) {
	rows, err := r.db.Query(ctx, `
		SELECT pt.id, pt.code, pt.name, pt.is_active, pt.created_at
		FROM work_category_payment_types wcpt
		JOIN payment_types pt ON pt.id = wcpt.payment_type_id
		WHERE wcpt.work_category_id = $1
			AND pt.is_active = TRUE
		ORDER BY pt.name ASC
	`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPaymentTypes(rows)
}

func (r *PaymentTypeRepository) ListByWorkTypeID(ctx context.Context, workTypeID string) ([]model.PaymentType, error) {
	rows, err := r.db.Query(ctx, `
		WITH type_payment_types AS (
			SELECT pt.id, pt.code, pt.name, pt.is_active, pt.created_at
			FROM work_type_payment_types wtpt
			JOIN payment_types pt ON pt.id = wtpt.payment_type_id
			WHERE wtpt.work_type_id = $1
				AND pt.is_active = TRUE
		)
		SELECT id, code, name, is_active, created_at
		FROM type_payment_types
		UNION ALL
		SELECT pt.id, pt.code, pt.name, pt.is_active, pt.created_at
		FROM work_types wt
		JOIN work_category_payment_types wcpt ON wcpt.work_category_id = wt.category_id
		JOIN payment_types pt ON pt.id = wcpt.payment_type_id
		WHERE wt.id = $1
			AND pt.is_active = TRUE
			AND NOT EXISTS (SELECT 1 FROM type_payment_types)
		ORDER BY name ASC
	`, workTypeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPaymentTypes(rows)
}

func scanPaymentTypes(rows pgx.Rows) ([]model.PaymentType, error) {
	types := []model.PaymentType{}
	for rows.Next() {
		var paymentType model.PaymentType
		if err := rows.Scan(
			&paymentType.ID,
			&paymentType.Code,
			&paymentType.Name,
			&paymentType.IsActive,
			&paymentType.CreatedAt,
		); err != nil {
			return nil, err
		}
		types = append(types, paymentType)
	}
	return types, rows.Err()
}
