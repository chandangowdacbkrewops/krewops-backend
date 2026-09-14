package repository

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
)

type WorkRepository struct {
	db *pgxpool.Pool
}

func NewWorkRepository(db *pgxpool.Pool) *WorkRepository {
	return &WorkRepository{db: db}
}

func (r *WorkRepository) InsertWork(
	ctx context.Context,
	record *model.WorkRecord,
) (*model.WorkRecord, error) {
	query := `
		INSERT INTO work_postings (
			user_id,
			title,
			description,
			work_type_id,
			address,
			city,
			state,
			latitude,
			longitude,
			workers_needed,
			experience_level,
			skills,
			tools_provided,
			materials_provided,
			start_date,
			duration_value,
			duration_unit,
			shift_timing,
			payment_type,
			budget_rate,
			payment_notes,
			accommodation_provided,
			meals_provided,
			status,
			published_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25
		)
		RETURNING
			id,
			user_id,
			title,
			description,
			work_type_id,
			address,
			city,
			state,
			latitude,
			longitude,
			workers_needed,
			experience_level,
			skills,
			tools_provided,
			materials_provided,
			start_date,
			duration_value,
			duration_unit,
			shift_timing,
			payment_type,
			budget_rate,
			payment_notes,
			accommodation_provided,
			meals_provided,
			status,
			published_at,
			created_at,
			updated_at
	`

	saved := &model.WorkRecord{}
	err := r.db.QueryRow(ctx, query,
		record.UserID,
		record.Title,
		record.Description,
		record.WorkTypeID,
		record.Address,
		record.City,
		record.State,
		record.Latitude,
		record.Longitude,
		record.WorkersNeeded,
		record.ExperienceLevel,
		record.Skills,
		record.ToolsProvided,
		record.MaterialsProvided,
		record.StartDate,
		record.DurationValue,
		record.DurationUnit,
		record.ShiftTiming,
		record.PaymentType,
		record.BudgetRate,
		record.PaymentNotes,
		record.AccommodationProvided,
		record.MealsProvided,
		record.Status,
		record.PublishedAt,
	).Scan(
		&saved.ID,
		&saved.UserID,
		&saved.Title,
		&saved.Description,
		&saved.WorkTypeID,
		&saved.Address,
		&saved.City,
		&saved.State,
		&saved.Latitude,
		&saved.Longitude,
		&saved.WorkersNeeded,
		&saved.ExperienceLevel,
		&saved.Skills,
		&saved.ToolsProvided,
		&saved.MaterialsProvided,
		&saved.StartDate,
		&saved.DurationValue,
		&saved.DurationUnit,
		&saved.ShiftTiming,
		&saved.PaymentType,
		&saved.BudgetRate,
		&saved.PaymentNotes,
		&saved.AccommodationProvided,
		&saved.MealsProvided,
		&saved.Status,
		&saved.PublishedAt,
		&saved.CreatedAt,
		&saved.UpdatedAt,
	)
	if err != nil {
		log.Printf("insert work failed: user_id=%s work_type_id=%s status=%s error=%v", record.UserID, stringValue(record.WorkTypeID), record.Status, err)
		return nil, err
	}

	return saved, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
