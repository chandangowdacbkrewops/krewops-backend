package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
)

type BusinessTypeRepository struct {
	db *pgxpool.Pool
}

func NewBusinessTypeRepository(db *pgxpool.Pool) *BusinessTypeRepository {
	return &BusinessTypeRepository{db: db}
}

func (r *BusinessTypeRepository) ListAll(ctx context.Context) ([]model.BusinessType, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name
		FROM business_types
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	types := []model.BusinessType{}
	for rows.Next() {
		var bt model.BusinessType
		if err := rows.Scan(&bt.ID, &bt.Name); err != nil {
			return nil, err
		}
		types = append(types, bt)
	}

	return types, rows.Err()
}

func (r *BusinessTypeRepository) FindByID(ctx context.Context, id string) (*model.BusinessType, error) {
	bt := &model.BusinessType{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name
		FROM business_types
		WHERE id = $1
	`, id).Scan(&bt.ID, &bt.Name)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return bt, nil
}
