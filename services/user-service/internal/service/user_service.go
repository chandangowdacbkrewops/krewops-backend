package service

import (
	"context"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
)

type UserService struct {
	profileService *ProfileService
}

func NewUserService(profileService *ProfileService) *UserService {
	return &UserService{profileService: profileService}
}

func (s *UserService) GetProfile(
	ctx context.Context,
	authUserID string,
) (*model.ProfileResponse, error) {
	profile, err := s.profileService.GetProfile(ctx, authUserID)
	if err != nil {
		return nil, err
	}
	if profile != nil && profile.AuthUserID == "" {
		profile.AuthUserID = authUserID
	}
	return profile, nil
}

func (s *UserService) CreateProfile(
	ctx context.Context,
	authUserID string,
	req model.CreateProfileRequest,
) (*model.ProfileResponse, error) {
	profile, err := s.profileService.CreateProfile(ctx, authUserID, req)
	if err != nil {
		return nil, err
	}
	if profile != nil && profile.AuthUserID == "" {
		profile.AuthUserID = authUserID
	}
	return profile, nil
}

func (s *UserService) CreateWorkerProfile(
	ctx context.Context,
	authUserID string,
	req model.CreateWorkerProfileRequest,
) (*model.WorkerProfile, error) {
	return s.profileService.CreateWorkerProfile(ctx, authUserID, req)
}

func (s *UserService) CreateOwnerProfile(
	ctx context.Context,
	authUserID string,
	req model.CreateOwnerProfileRequest,
) (*model.OwnerProfileResponse, error) {
	return s.profileService.CreateOwnerProfile(ctx, authUserID, req)
}

func (s *UserService) ListBusinessTypes(ctx context.Context) ([]model.BusinessType, error) {
	return s.profileService.ListBusinessTypes(ctx)
}

func (s *UserService) ListWorkTypes(ctx context.Context) ([]model.WorkType, error) {
	return s.profileService.ListWorkTypes(ctx)
}
