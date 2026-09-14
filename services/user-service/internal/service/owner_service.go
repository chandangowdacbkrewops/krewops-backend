package service

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/repository"
)

type OwnerService struct {
	ownerRepository        *repository.OwnerRepository
	businessTypeRepository *repository.BusinessTypeRepository
}

func NewOwnerService(
	ownerRepository *repository.OwnerRepository,
	businessTypeRepository *repository.BusinessTypeRepository,
) *OwnerService {
	return &OwnerService{
		ownerRepository:        ownerRepository,
		businessTypeRepository: businessTypeRepository,
	}
}

func (s *OwnerService) ListBusinessTypes(ctx context.Context) ([]model.BusinessType, error) {
	return s.businessTypeRepository.ListAll(ctx)
}

func (s *OwnerService) CreateOwnerProfile(
	ctx context.Context,
	authUserID string,
	req model.CreateOwnerProfileRequest,
) (*model.OwnerProfileResponse, error) {
	if err := s.validateRequest(ctx, req); err != nil {
		return nil, err
	}

	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return nil, fmt.Errorf("date_of_birth must be in YYYY-MM-DD format")
	}

	if age := ageYears(dob, time.Now()); age < 18 {
		return nil, fmt.Errorf("user must be at least 18 years old")
	}

	profile, owner, err := s.ownerRepository.CreateOwnerProfile(ctx, authUserID, req, dob)
	if err != nil {
		return nil, err
	}

	businessType, err := s.businessTypeRepository.FindByID(ctx, req.BusinessTypeID)
	if err != nil {
		return nil, err
	}

	owner.BusinessType = businessType
	return buildOwnerProfileResponse(profile, owner), nil
}

func (s *OwnerService) validateRequest(ctx context.Context, req model.CreateOwnerProfileRequest) error {
	registrationType := strings.ToLower(strings.TrimSpace(req.RegistrationType))
	if registrationType != model.RegistrationBusiness && registrationType != model.RegistrationIndividual {
		return fmt.Errorf("registration_type must be business or individual")
	}
	req.RegistrationType = registrationType

	if registrationType == model.RegistrationBusiness {
		if req.BusinessName == nil || strings.TrimSpace(*req.BusinessName) == "" {
			return fmt.Errorf("business_name is required when registration_type is business")
		}
	}

	if req.Email != nil && strings.TrimSpace(*req.Email) != "" {
		if _, err := mail.ParseAddress(*req.Email); err != nil {
			return fmt.Errorf("email must be a valid email address")
		}
	}

	businessType, err := s.businessTypeRepository.FindByID(ctx, req.BusinessTypeID)
	if err != nil {
		return err
	}
	if businessType == nil {
		return fmt.Errorf("business_type_id is invalid")
	}

	return nil
}

func (s *OwnerService) GetOwnerProfileResponse(
	ctx context.Context,
	authUserID string,
) (*model.OwnerProfileResponse, error) {
	profile, owner, err := s.ownerRepository.GetOwnerProfileByAuthUserID(ctx, authUserID)
	if err != nil {
		return nil, err
	}
	if profile == nil || owner == nil {
		return nil, nil
	}

	return buildOwnerProfileResponse(profile, owner), nil
}

func buildOwnerProfileResponse(
	profile *model.ProfileRecord,
	owner *model.OwnerProfileRecord,
) *model.OwnerProfileResponse {
	resp := &model.OwnerProfileResponse{
		ProfileCompleted:  profile.ProfileCompleted,
		UserType:          model.UserTypeWorkOwner,
		FirstName:         derefString(profile.FirstName),
		LastName:          profile.LastName,
		Email:             profile.Email,
		Country:           derefString(profile.Country),
		State:             derefString(profile.State),
		City:              derefString(profile.City),
		PostalCode:        derefString(profile.PostalCode),
		PreferredLanguage: derefString(profile.PreferredLanguage),
	}

	if profile.DateOfBirth != nil {
		resp.DateOfBirth = profile.DateOfBirth.Format("2006-01-02")
	}

	if owner != nil && owner.BusinessType != nil {
		resp.OwnerProfile = &model.OwnerProfileDetails{
			RegistrationType:      derefString(owner.RegistrationType),
			BusinessName:          owner.BusinessName,
			BusinessDescription:   owner.BusinessDescription,
			BusinessType:          *owner.BusinessType,
			BusinessSize:          owner.BusinessSize,
			GSTRegistrationNumber: owner.GSTRegistrationNumber,
		}
	}

	return resp
}

func BuildProfileResponse(
	profile *model.ProfileRecord,
	ownerResp *model.OwnerProfileResponse,
) *model.ProfileResponse {
	if ownerResp != nil {
		return &model.ProfileResponse{
			ProfileCompleted:    ownerResp.ProfileCompleted,
			OnboardingCompleted: true,
			FirstName:           &ownerResp.FirstName,
			LastName:            ownerResp.LastName,
			UserType:            &ownerResp.UserType,
			DateOfBirth:         &ownerResp.DateOfBirth,
			Email:               ownerResp.Email,
			Country:             &ownerResp.Country,
			State:               &ownerResp.State,
			City:                &ownerResp.City,
			PostalCode:          &ownerResp.PostalCode,
			PreferredLanguage:   &ownerResp.PreferredLanguage,
			OwnerProfile:        ownerResp.OwnerProfile,
		}
	}

	if profile == nil {
		return &model.ProfileResponse{ProfileCompleted: false}
	}

	resp := &model.ProfileResponse{
		ID:                  profile.ID,
		AuthUserID:          profile.AuthUserID,
		ProfileCompleted:    profile.ProfileCompleted,
		OnboardingCompleted: profile.OnboardingCompleted,
		FirstName:           profile.FirstName,
		LastName:            profile.LastName,
		UserType:            profile.UserType,
		Email:               profile.Email,
		Country:             profile.Country,
		State:               profile.State,
		City:                profile.City,
		PostalCode:          profile.PostalCode,
		PreferredLanguage:   profile.PreferredLanguage,
	}

	if profile.DateOfBirth != nil {
		formatted := profile.DateOfBirth.Format("2006-01-02")
		resp.DateOfBirth = &formatted
	}

	return resp
}

func ageYears(dob time.Time, now time.Time) int {
	years := now.Year() - dob.Year()
	if now.YearDay() < dob.YearDay() {
		years--
	}
	return years
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
