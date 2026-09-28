package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
)

type SkillRepository struct {
	db *pgxpool.Pool
}

func NewSkillRepository(db *pgxpool.Pool) *SkillRepository {
	return &SkillRepository{db: db}
}

func (r *SkillRepository) ListByWorkTypeID(ctx context.Context, workTypeID string) ([]model.Skill, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, work_type_id, code, name, is_active, created_at, updated_at
		FROM skills
		WHERE work_type_id = $1
			AND is_active = TRUE
		ORDER BY name ASC
	`, workTypeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	skills := []model.Skill{}
	for rows.Next() {
		var skill model.Skill
		if err := rows.Scan(
			&skill.ID,
			&skill.WorkTypeID,
			&skill.Code,
			&skill.Name,
			&skill.IsActive,
			&skill.CreatedAt,
			&skill.UpdatedAt,
		); err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	return skills, rows.Err()
}

func (r *SkillRepository) FindByIDs(ctx context.Context, ids []string) (map[string]model.Skill, error) {
	results := make(map[string]model.Skill, len(ids))
	if len(ids) == 0 {
		return results, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, work_type_id, code, name, is_active, created_at, updated_at
		FROM skills
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var skill model.Skill
		if err := rows.Scan(
			&skill.ID,
			&skill.WorkTypeID,
			&skill.Code,
			&skill.Name,
			&skill.IsActive,
			&skill.CreatedAt,
			&skill.UpdatedAt,
		); err != nil {
			return nil, err
		}
		results[skill.ID] = skill
	}
	return results, rows.Err()
}
