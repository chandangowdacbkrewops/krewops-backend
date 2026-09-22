package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
)

type OwnerRepository struct {
	db *pgxpool.Pool
}

func NewOwnerRepository(db *pgxpool.Pool) *OwnerRepository {
	return &OwnerRepository{db: db}
}

func (r *OwnerRepository) CreateOwnerProfile(
	ctx context.Context,
	authUserID string,
	req model.CreateOwnerProfileRequest,
	dob time.Time,
) (*model.ProfileRecord, *model.OwnerProfileRecord, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	profile, err := r.upsertProfile(ctx, tx, authUserID, req, dob)
	if err != nil {
		return nil, nil, err
	}

	ownerProfile, err := r.upsertOwnerProfile(ctx, tx, profile.ID, req)
	if err != nil {
		return nil, nil, err
	}

	if err := r.replaceBusinessType(ctx, tx, ownerProfile.ID, req.BusinessTypeID); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	return profile, ownerProfile, nil
}

func (r *OwnerRepository) upsertProfile(
	ctx context.Context,
	tx pgx.Tx,
	authUserID string,
	req model.CreateOwnerProfileRequest,
	dob time.Time,
) (*model.ProfileRecord, error) {
	userType := model.UserTypeUser

	query := `
		INSERT INTO profiles (
			user_id,
			first_name,
			last_name,
			user_type,
			date_of_birth,
			email,
			country,
			state,
			city,
			postal_code,
			preferred_language,
			profile_completed,
			onboarding_completed
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, TRUE, TRUE)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			user_type = EXCLUDED.user_type,
			date_of_birth = EXCLUDED.date_of_birth,
			email = EXCLUDED.email,
			country = EXCLUDED.country,
			state = EXCLUDED.state,
			city = EXCLUDED.city,
			postal_code = EXCLUDED.postal_code,
			preferred_language = EXCLUDED.preferred_language,
			profile_completed = TRUE,
			onboarding_completed = TRUE,
			updated_at = NOW()
		RETURNING
			id,
			user_id,
			first_name,
			last_name,
			user_type,
			date_of_birth,
			email,
			country,
			state,
			city,
			postal_code,
			preferred_language,
			profile_completed,
			created_at,
			updated_at
	`

	profile := &model.ProfileRecord{}
	err := tx.QueryRow(ctx, query,
		authUserID,
		req.FirstName,
		req.LastName,
		userType,
		dob,
		req.Email,
		req.Country,
		req.State,
		req.City,
		req.PostalCode,
		req.PreferredLanguage,
	).Scan(
		&profile.ID,
		&profile.AuthUserID,
		&profile.FirstName,
		&profile.LastName,
		&profile.UserType,
		&profile.DateOfBirth,
		&profile.Email,
		&profile.Country,
		&profile.State,
		&profile.City,
		&profile.PostalCode,
		&profile.PreferredLanguage,
		&profile.ProfileCompleted,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (r *OwnerRepository) upsertOwnerProfile(
	ctx context.Context,
	tx pgx.Tx,
	profileID string,
	req model.CreateOwnerProfileRequest,
) (*model.OwnerProfileRecord, error) {
	query := `
		INSERT INTO owner_profiles (
			profile_id,
			business_name,
			business_description,
			registration_type,
			business_size,
			gst_registration_number
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (profile_id) DO UPDATE SET
			business_name = EXCLUDED.business_name,
			business_description = EXCLUDED.business_description,
			registration_type = EXCLUDED.registration_type,
			business_size = EXCLUDED.business_size,
			gst_registration_number = EXCLUDED.gst_registration_number,
			updated_at = NOW()
		RETURNING id, profile_id, registration_type, business_name, business_description, business_size, gst_registration_number
	`

	owner := &model.OwnerProfileRecord{}
	err := tx.QueryRow(ctx, query,
		profileID,
		req.BusinessName,
		req.BusinessDescription,
		req.RegistrationType,
		req.BusinessSize,
		req.GSTRegistrationNumber,
	).Scan(
		&owner.ID,
		&owner.ProfileID,
		&owner.RegistrationType,
		&owner.BusinessName,
		&owner.BusinessDescription,
		&owner.BusinessSize,
		&owner.GSTRegistrationNumber,
	)
	if err != nil {
		return nil, err
	}

	return owner, nil
}

func (r *OwnerRepository) replaceBusinessType(
	ctx context.Context,
	tx pgx.Tx,
	ownerProfileID string,
	businessTypeID string,
) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM owner_business_types
		WHERE owner_profile_id = $1
	`, ownerProfileID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO owner_business_types (owner_profile_id, business_type_id)
		VALUES ($1, $2)
	`, ownerProfileID, businessTypeID)

	return err
}

func (r *OwnerRepository) GetOwnerProfileByAuthUserID(
	ctx context.Context,
	authUserID string,
) (*model.ProfileRecord, *model.OwnerProfileRecord, error) {
	profileQuery := `
		SELECT
			id,
			user_id,
			first_name,
			last_name,
			user_type,
			date_of_birth,
			email,
			country,
			state,
			city,
			postal_code,
			preferred_language,
			profile_completed,
			created_at,
			updated_at
		FROM profiles
		WHERE user_id = $1
	`

	profile := &model.ProfileRecord{}
	err := r.db.QueryRow(ctx, profileQuery, authUserID).Scan(
		&profile.ID,
		&profile.AuthUserID,
		&profile.FirstName,
		&profile.LastName,
		&profile.UserType,
		&profile.DateOfBirth,
		&profile.Email,
		&profile.Country,
		&profile.State,
		&profile.City,
		&profile.PostalCode,
		&profile.PreferredLanguage,
		&profile.ProfileCompleted,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	if profile.UserType == nil || *profile.UserType != model.UserTypeUser {
		return profile, nil, nil
	}

	ownerQuery := `
		SELECT
			op.id,
			op.profile_id,
			op.registration_type,
			op.business_name,
			op.business_description,
			op.business_size,
			op.gst_registration_number,
			bt.id,
			bt.name
		FROM owner_profiles op
		LEFT JOIN owner_business_types obt ON obt.owner_profile_id = op.id
		LEFT JOIN business_types bt ON bt.id = obt.business_type_id
		WHERE op.profile_id = $1
	`

	owner := &model.OwnerProfileRecord{}
	var businessTypeID, businessTypeName *string
	err = r.db.QueryRow(ctx, ownerQuery, profile.ID).Scan(
		&owner.ID,
		&owner.ProfileID,
		&owner.RegistrationType,
		&owner.BusinessName,
		&owner.BusinessDescription,
		&owner.BusinessSize,
		&owner.GSTRegistrationNumber,
		&businessTypeID,
		&businessTypeName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return profile, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	if businessTypeID != nil && businessTypeName != nil {
		owner.BusinessType = &model.BusinessType{
			ID:   *businessTypeID,
			Name: *businessTypeName,
		}
	}

	return profile, owner, nil
}
