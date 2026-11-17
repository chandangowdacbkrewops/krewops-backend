package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
)

type CommonRoleRepository struct {
	db *pgxpool.Pool
}

func NewCommonRoleRepository(db *pgxpool.Pool) *CommonRoleRepository {
	return &CommonRoleRepository{db: db}
}

// ListByWorkTypeID returns the active common roles mapped to a work type via
// the work_type_common_roles join table, ordered by sort_order then name.
func (r *CommonRoleRepository) ListByWorkTypeID(ctx context.Context, workTypeID string) ([]model.CommonRole, error) {
	rows, err := r.db.Query(ctx, `
		SELECT cr.id, cr.code, cr.name, cr.is_active, cr.created_at, cr.updated_at, wtcr.is_primary
		FROM common_roles cr
		JOIN work_type_common_roles wtcr ON wtcr.common_role_id = cr.id
		WHERE wtcr.work_type_id = $1
			AND cr.is_active = TRUE
		ORDER BY wtcr.sort_order ASC, cr.name ASC
	`, workTypeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := []model.CommonRole{}
	for rows.Next() {
		var role model.CommonRole
		if err := rows.Scan(
			&role.ID,
			&role.Code,
			&role.Name,
			&role.IsActive,
			&role.CreatedAt,
			&role.UpdatedAt,
			&role.IsPrimary,
		); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}
