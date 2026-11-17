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

func (s *UserService) UpdateProfile(
	ctx context.Context,
	authUserID string,
	req model.UpdateProfileRequest,
) (*model.ProfileResponse, error) {
	profile, err := s.profileService.UpdateProfile(ctx, authUserID, req)
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

func (s *UserService) UpdateWorkerProfile(
	ctx context.Context,
	authUserID string,
	req model.UpdateWorkerProfileRequest,
) (*model.WorkerProfile, error) {
	return s.profileService.UpdateWorkerProfile(ctx, authUserID, req)
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

func (s *UserService) ListWorkCategories(ctx context.Context) ([]model.WorkCategory, error) {
	return s.profileService.ListWorkCategories(ctx)
}

func (s *UserService) ListWorkTypesByCategory(ctx context.Context, categoryID string) ([]model.WorkType, error) {
	return s.profileService.ListWorkTypesByCategory(ctx, categoryID)
}

func (s *UserService) ListWorkSegmentsByCategory(ctx context.Context, categoryID string) ([]model.WorkSegment, error) {
	return s.profileService.ListWorkSegmentsByCategory(ctx, categoryID)
}

func (s *UserService) ListWorkTypesBySegment(ctx context.Context, segmentID string) ([]model.WorkType, error) {
	return s.profileService.ListWorkTypesBySegment(ctx, segmentID)
}

func (s *UserService) ListWorkTypeFields(ctx context.Context, workTypeID string) ([]model.WorkTypeField, error) {
	return s.profileService.ListWorkTypeFields(ctx, workTypeID)
}

func (s *UserService) ListWorkTypeRateTypes(ctx context.Context, workTypeID string) ([]model.RateOption, error) {
	return s.profileService.ListWorkTypeRateTypes(ctx, workTypeID)
}

func (s *UserService) ListWorkSegmentRateTypes(ctx context.Context, segmentID string) ([]model.RateOption, error) {
	return s.profileService.ListWorkSegmentRateTypes(ctx, segmentID)
}

func (s *UserService) ListWorkCategoryRateTypes(ctx context.Context, categoryID string) ([]model.RateOption, error) {
	return s.profileService.ListWorkCategoryRateTypes(ctx, categoryID)
}

func (s *UserService) ListWorkTypeSkills(ctx context.Context, workTypeID string) ([]model.Skill, error) {
	return s.profileService.ListWorkTypeSkills(ctx, workTypeID)
}

func (s *UserService) ListWorkTypeCommonRoles(ctx context.Context, workTypeID string) ([]model.CommonRole, error) {
	return s.profileService.ListWorkTypeCommonRoles(ctx, workTypeID)
}

func (s *UserService) GetWorkTypeConfiguration(ctx context.Context, workTypeID string) (*WorkTypeConfiguration, error) {
	return s.profileService.GetWorkTypeConfiguration(ctx, workTypeID)
}
