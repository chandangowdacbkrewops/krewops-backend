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

// ListByWorkTypeID returns the active skills mapped to a work type via the
// work_type_skills join table, ordered by sort_order then name.
func (r *SkillRepository) ListByWorkTypeID(ctx context.Context, workTypeID string) ([]model.Skill, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.id, s.code, s.name, s.is_active, s.created_at, s.updated_at, wts.is_required
		FROM skills s
		JOIN work_type_skills wts ON wts.skill_id = s.id
		WHERE wts.work_type_id = $1
			AND s.is_active = TRUE
		ORDER BY wts.sort_order ASC, s.name ASC
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
			&skill.Code,
			&skill.Name,
			&skill.IsActive,
			&skill.CreatedAt,
			&skill.UpdatedAt,
			&skill.IsRequired,
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
		SELECT id, code, name, is_active, created_at, updated_at
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

// AllowedForWorkType returns the subset of skillIDs that are validly mapped
// to the given work type via work_type_skills (used to validate worker
// profile / work posting skill selections against the reusable catalog).
func (r *SkillRepository) AllowedForWorkType(ctx context.Context, workTypeID string, skillIDs []string) (map[string]bool, error) {
	allowed := make(map[string]bool, len(skillIDs))
	if len(skillIDs) == 0 {
		return allowed, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT skill_id
		FROM work_type_skills
		WHERE work_type_id = $1 AND skill_id = ANY($2)
	`, workTypeID, skillIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		allowed[id] = true
	}
	return allowed, rows.Err()
}
