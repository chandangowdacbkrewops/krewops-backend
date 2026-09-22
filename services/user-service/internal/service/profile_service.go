package service

import (
	"context"
	"errors"
	"strings"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/repository"
)

type ProfileService struct {
	profileRepository  *repository.ProfileRepository
	ownerService       *OwnerService
	workTypeRepository *repository.WorkTypeRepository
}

var ErrInvalidProfile = errors.New("first_name, last_name, country, state, city, and postal_code are required")

func NewProfileService(
	profileRepository *repository.ProfileRepository,
	ownerService *OwnerService,
	workTypeRepository *repository.WorkTypeRepository,
) *ProfileService {
	return &ProfileService{
		profileRepository:  profileRepository,
		ownerService:       ownerService,
		workTypeRepository: workTypeRepository,
	}
}

func (s *ProfileService) GetProfile(
	ctx context.Context,
	authUserID string,
) (*model.ProfileResponse, error) {
	ownerResp, err := s.ownerService.GetOwnerProfileResponse(ctx, authUserID)
	if err != nil {
		return nil, err
	}
	if ownerResp != nil {
		return BuildProfileResponse(nil, ownerResp), nil
	}

	profile, err := s.profileRepository.FindByAuthUserID(ctx, authUserID)
	if err != nil {
		return nil, err
	}

	if profile == nil {
		return &model.ProfileResponse{
			ProfileCompleted:    false,
			OnboardingCompleted: false,
		}, nil
	}

	record := &model.ProfileRecord{
		ID:                  profile.ID,
		AuthUserID:          profile.AuthUserID,
		FirstName:           profile.FirstName,
		LastName:            profile.LastName,
		UserType:            profile.UserType,
		DateOfBirth:         profile.DateOfBirth,
		Email:               profile.Email,
		Country:             profile.Country,
		State:               profile.State,
		City:                profile.City,
		PostalCode:          profile.PostalCode,
		PreferredLanguage:   profile.PreferredLanguage,
		ProfileCompleted:    profile.ProfileCompleted,
		OnboardingCompleted: profile.OnboardingCompleted,
		CreatedAt:           profile.CreatedAt,
		UpdatedAt:           profile.UpdatedAt,
	}

	return BuildProfileResponse(record, nil), nil
}

func (s *ProfileService) CreateProfile(
	ctx context.Context,
	authUserID string,
	req model.CreateProfileRequest,
) (*model.ProfileResponse, error) {
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Country = strings.TrimSpace(req.Country)
	req.State = strings.TrimSpace(req.State)
	req.City = strings.TrimSpace(req.City)
	req.PostalCode = strings.TrimSpace(req.PostalCode)
	req.UserType = strings.TrimSpace(req.UserType)

	if req.FirstName == "" || req.LastName == "" || req.Country == "" ||
		req.State == "" || req.City == "" || req.PostalCode == "" || !model.ValidUserType(req.UserType) {
		return nil, ErrInvalidProfile
	}

	profile, err := s.profileRepository.CreateProfile(ctx, authUserID, req)
	if err != nil {
		return nil, err
	}

	record := &model.ProfileRecord{
		ID:                  profile.ID,
		AuthUserID:          profile.AuthUserID,
		FirstName:           profile.FirstName,
		LastName:            profile.LastName,
		UserType:            profile.UserType,
		DateOfBirth:         profile.DateOfBirth,
		Email:               profile.Email,
		Country:             profile.Country,
		State:               profile.State,
		City:                profile.City,
		PostalCode:          profile.PostalCode,
		PreferredLanguage:   profile.PreferredLanguage,
		ProfileCompleted:    profile.ProfileCompleted,
		OnboardingCompleted: profile.OnboardingCompleted,
	}

	return BuildProfileResponse(record, nil), nil
}

func (s *ProfileService) UpdateProfile(
	ctx context.Context,
	authUserID string,
	req model.UpdateProfileRequest,
) (*model.ProfileResponse, error) {
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Country = strings.TrimSpace(req.Country)
	req.State = strings.TrimSpace(req.State)
	req.City = strings.TrimSpace(req.City)
	req.PostalCode = strings.TrimSpace(req.PostalCode)

	if req.FirstName == "" || req.LastName == "" || req.Country == "" ||
		req.State == "" || req.City == "" || req.PostalCode == "" {
		return nil, ErrInvalidProfile
	}

	profile, err := s.profileRepository.UpdateProfile(ctx, authUserID, req)
	if err != nil {
		return nil, err
	}

	return BuildProfileResponse(&model.ProfileRecord{
		ID:                  profile.ID,
		AuthUserID:          profile.AuthUserID,
		FirstName:           profile.FirstName,
		LastName:            profile.LastName,
		UserType:            profile.UserType,
		DateOfBirth:         profile.DateOfBirth,
		Email:               profile.Email,
		Country:             profile.Country,
		State:               profile.State,
		City:                profile.City,
		PostalCode:          profile.PostalCode,
		PreferredLanguage:   profile.PreferredLanguage,
		ProfileCompleted:    profile.ProfileCompleted,
		OnboardingCompleted: profile.OnboardingCompleted,
	}, nil), nil
}

func (s *ProfileService) CreateWorkerProfile(
	ctx context.Context,
	authUserID string,
	req model.CreateWorkerProfileRequest,
) (*model.WorkerProfile, error) {
	req.WorkerType = strings.ToLower(strings.TrimSpace(req.WorkerType))
	if req.RateType != nil {
		value := strings.ToLower(strings.TrimSpace(*req.RateType))
		req.RateType = &value
	}
	if req.AvailabilityStatus != nil {
		value := strings.ToLower(strings.TrimSpace(*req.AvailabilityStatus))
		req.AvailabilityStatus = &value
	}
	return s.profileRepository.CreateWorkerProfile(ctx, authUserID, req)
}

func (s *ProfileService) CreateOwnerProfile(
	ctx context.Context,
	authUserID string,
	req model.CreateOwnerProfileRequest,
) (*model.OwnerProfileResponse, error) {
	return s.ownerService.CreateOwnerProfile(ctx, authUserID, req)
}

func (s *ProfileService) ListBusinessTypes(ctx context.Context) ([]model.BusinessType, error) {
	return s.ownerService.ListBusinessTypes(ctx)
}

func (s *ProfileService) ListWorkTypes(ctx context.Context) ([]model.WorkType, error) {
	return s.workTypeRepository.ListAll(ctx)
}
