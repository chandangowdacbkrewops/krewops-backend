package service

import (
	"context"
	"errors"
	"strings"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/repository"
)

type ProfileService struct {
	profileRepository       *repository.ProfileRepository
	ownerService            *OwnerService
	workTypeRepository      *repository.WorkTypeRepository
	workCategoryRepository  *repository.WorkCategoryRepository
	workTypeFieldRepository *repository.WorkTypeFieldRepository
	paymentTypeRepository   *repository.PaymentTypeRepository
	skillRepository         *repository.SkillRepository
}

var (
	ErrInvalidProfile          = errors.New("first_name, last_name, country, state, city, and postal_code are required")
	ErrInvalidWorkerSelections = errors.New("invalid worker selections")
)

type workerSelectionError struct {
	msg string
}

func (e workerSelectionError) Error() string { return e.msg }
func (e workerSelectionError) Unwrap() error { return ErrInvalidWorkerSelections }

func invalidWorkerSelection(msg string) error {
	return workerSelectionError{msg: msg}
}

func NewProfileService(
	profileRepository *repository.ProfileRepository,
	ownerService *OwnerService,
	workTypeRepository *repository.WorkTypeRepository,
	workCategoryRepository *repository.WorkCategoryRepository,
	workTypeFieldRepository *repository.WorkTypeFieldRepository,
	paymentTypeRepository *repository.PaymentTypeRepository,
	skillRepository *repository.SkillRepository,
) *ProfileService {
	return &ProfileService{
		profileRepository:       profileRepository,
		ownerService:            ownerService,
		workTypeRepository:      workTypeRepository,
		workCategoryRepository:  workCategoryRepository,
		workTypeFieldRepository: workTypeFieldRepository,
		paymentTypeRepository:   paymentTypeRepository,
		skillRepository:         skillRepository,
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

	resp := BuildProfileResponse(record, nil)
	if err := s.attachWorkerCategories(ctx, authUserID, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *ProfileService) attachWorkerCategories(
	ctx context.Context,
	authUserID string,
	resp *model.ProfileResponse,
) error {
	if resp == nil {
		return nil
	}

	worker, err := s.profileRepository.FindWorkerProfileByUserID(ctx, authUserID)
	if errors.Is(err, repository.ErrWorkerProfileNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if worker == nil {
		return nil
	}

	resp.WorkCategory = worker.WorkCategory
	resp.WorkCategories = worker.WorkCategories
	return nil
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

	selections, err := s.validateOnboardingSelections(ctx, req.WorkerType, req.Selections)
	if err != nil {
		return nil, err
	}
	req.Selections = selections
	if req.WorkCategoryID == "" && len(selections) > 0 {
		req.WorkCategoryID = selections[0].WorkCategoryID
	}

	return s.profileRepository.CreateWorkerProfile(ctx, authUserID, req)
}

func (s *ProfileService) UpdateWorkerProfile(
	ctx context.Context,
	authUserID string,
	req model.UpdateWorkerProfileRequest,
) (*model.WorkerProfile, error) {
	profile, err := s.profileRepository.FindWorkerProfileByUserID(ctx, authUserID)
	if err != nil {
		return nil, err
	}

	allowedCategories := map[string]struct{}{}
	for _, category := range profile.WorkCategories {
		allowedCategories[category.WorkCategoryID] = struct{}{}
	}

	selections, err := s.validateProfileSelections(ctx, profile.WorkerType, req.Selections, allowedCategories)
	if err != nil {
		return nil, err
	}

	return s.profileRepository.UpdateWorkerSelections(ctx, authUserID, selections)
}

func normalizeOnboardingSelections(
	workerType string,
	selections []model.WorkerCategorySelection,
) ([]model.WorkerCategorySelection, error) {
	if workerType != "individual" && workerType != "contractor" {
		return nil, invalidWorkerSelection("worker_type must be individual or contractor")
	}
	if len(selections) == 0 {
		return nil, invalidWorkerSelection("selections are required")
	}

	normalized := make([]model.WorkerCategorySelection, 0, len(selections))
	seenCategories := map[string]struct{}{}

	for _, selection := range selections {
		categoryID := strings.TrimSpace(selection.WorkCategoryID)
		if categoryID == "" {
			return nil, invalidWorkerSelection("work_category_id is required")
		}
		if _, exists := seenCategories[categoryID]; exists {
			return nil, invalidWorkerSelection("duplicate work_category_id")
		}
		seenCategories[categoryID] = struct{}{}
		normalized = append(normalized, model.WorkerCategorySelection{
			WorkCategoryID: categoryID,
		})
	}

	if workerType == "individual" && len(normalized) != 1 {
		return nil, invalidWorkerSelection("individual workers can select only one work category")
	}

	return normalized, nil
}

func normalizeProfileSelections(
	workerType string,
	selections []model.WorkerCategorySelection,
) ([]model.WorkerCategorySelection, []string, []string, error) {
	if workerType != "individual" && workerType != "contractor" {
		return nil, nil, nil, invalidWorkerSelection("worker_type must be individual or contractor")
	}
	if len(selections) == 0 {
		return nil, nil, nil, invalidWorkerSelection("selections are required")
	}

	normalized := make([]model.WorkerCategorySelection, 0, len(selections))
	seenCategories := map[string]struct{}{}
	seenWorkTypes := map[string]struct{}{}
	seenSkills := map[string]struct{}{}
	workTypeIDs := []string{}
	skillIDs := []string{}

	for _, selection := range selections {
		categoryID := strings.TrimSpace(selection.WorkCategoryID)
		if categoryID == "" {
			return nil, nil, nil, invalidWorkerSelection("work_category_id is required")
		}
		if _, exists := seenCategories[categoryID]; exists {
			return nil, nil, nil, invalidWorkerSelection("duplicate work_category_id")
		}
		seenCategories[categoryID] = struct{}{}

		if len(selection.WorkTypes) == 0 {
			return nil, nil, nil, invalidWorkerSelection("at least one work type is required for each category")
		}

		normalizedTypes := make([]model.WorkerTypeSelection, 0, len(selection.WorkTypes))
		for _, workType := range selection.WorkTypes {
			workTypeID := strings.TrimSpace(workType.WorkTypeID)
			if workTypeID == "" {
				return nil, nil, nil, invalidWorkerSelection("work_type_id is required")
			}
			if _, exists := seenWorkTypes[workTypeID]; exists {
				return nil, nil, nil, invalidWorkerSelection("duplicate work_type_id")
			}
			seenWorkTypes[workTypeID] = struct{}{}
			workTypeIDs = append(workTypeIDs, workTypeID)

			normalizedSkills := make([]string, 0, len(workType.SkillIDs))
			for _, skillID := range workType.SkillIDs {
				skillID = strings.TrimSpace(skillID)
				if skillID == "" {
					continue
				}
				if _, exists := seenSkills[skillID]; exists {
					return nil, nil, nil, invalidWorkerSelection("duplicate skill_id")
				}
				seenSkills[skillID] = struct{}{}
				normalizedSkills = append(normalizedSkills, skillID)
				skillIDs = append(skillIDs, skillID)
			}
			if len(normalizedSkills) == 0 {
				return nil, nil, nil, invalidWorkerSelection("at least one skill is required for each work type")
			}
			normalizedTypes = append(normalizedTypes, model.WorkerTypeSelection{
				WorkTypeID: workTypeID,
				SkillIDs:   normalizedSkills,
			})
		}

		normalized = append(normalized, model.WorkerCategorySelection{
			WorkCategoryID: categoryID,
			WorkTypes:      normalizedTypes,
		})
	}

	if workerType == "individual" {
		if len(normalized) != 1 {
			return nil, nil, nil, invalidWorkerSelection("individual workers can select only one work category")
		}
		if len(normalized[0].WorkTypes) != 1 {
			return nil, nil, nil, invalidWorkerSelection("individual workers can select only one work type")
		}
		if len(normalized[0].WorkTypes[0].SkillIDs) > 2 {
			return nil, nil, nil, invalidWorkerSelection("individual workers can select up to 2 skills")
		}
	}

	return normalized, workTypeIDs, skillIDs, nil
}

func (s *ProfileService) validateOnboardingSelections(
	ctx context.Context,
	workerType string,
	selections []model.WorkerCategorySelection,
) ([]model.WorkerCategorySelection, error) {
	normalized, err := normalizeOnboardingSelections(workerType, selections)
	if err != nil {
		return nil, err
	}

	for _, selection := range normalized {
		category, err := s.workCategoryRepository.FindByID(ctx, selection.WorkCategoryID)
		if err != nil {
			return nil, err
		}
		if category == nil || !category.IsActive {
			return nil, invalidWorkerSelection("work_category_id is invalid")
		}
	}

	return normalized, nil
}

func (s *ProfileService) validateProfileSelections(
	ctx context.Context,
	workerType string,
	selections []model.WorkerCategorySelection,
	allowedCategories map[string]struct{},
) ([]model.WorkerCategorySelection, error) {
	normalized, workTypeIDs, skillIDs, err := normalizeProfileSelections(workerType, selections)
	if err != nil {
		return nil, err
	}

	for _, selection := range normalized {
		if _, ok := allowedCategories[selection.WorkCategoryID]; !ok {
			return nil, invalidWorkerSelection("work_category_id was not selected during onboarding")
		}
	}

	for _, selection := range normalized {
		category, err := s.workCategoryRepository.FindByID(ctx, selection.WorkCategoryID)
		if err != nil {
			return nil, err
		}
		if category == nil || !category.IsActive {
			return nil, invalidWorkerSelection("work_category_id is invalid")
		}
	}

	workTypes, err := s.workTypeRepository.FindByIDs(ctx, workTypeIDs)
	if err != nil {
		return nil, err
	}
	for _, selection := range normalized {
		for _, workType := range selection.WorkTypes {
			found, ok := workTypes[workType.WorkTypeID]
			if !ok || !found.IsActive {
				return nil, invalidWorkerSelection("work_type_id is invalid")
			}
			if found.CategoryID != selection.WorkCategoryID {
				return nil, invalidWorkerSelection("work_type_id does not belong to work_category_id")
			}
		}
	}

	skills, err := s.skillRepository.FindByIDs(ctx, skillIDs)
	if err != nil {
		return nil, err
	}
	for _, selection := range normalized {
		for _, workType := range selection.WorkTypes {
			for _, skillID := range workType.SkillIDs {
				found, ok := skills[skillID]
				if !ok || !found.IsActive {
					return nil, invalidWorkerSelection("skill_id is invalid")
				}
				if found.WorkTypeID != workType.WorkTypeID {
					return nil, invalidWorkerSelection("skill_id does not belong to work_type_id")
				}
			}
		}
	}

	return normalized, nil
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

func (s *ProfileService) ListWorkCategories(ctx context.Context) ([]model.WorkCategory, error) {
	return s.workCategoryRepository.ListAll(ctx)
}

func (s *ProfileService) ListWorkTypesByCategory(ctx context.Context, categoryID string) ([]model.WorkType, error) {
	return s.workTypeRepository.ListByCategoryID(ctx, categoryID)
}

func (s *ProfileService) ListWorkTypeFields(ctx context.Context, workTypeID string) ([]model.WorkTypeField, error) {
	return s.workTypeFieldRepository.ListByWorkTypeID(ctx, workTypeID)
}

func (s *ProfileService) ListWorkTypePaymentTypes(ctx context.Context, workTypeID string) ([]model.PaymentType, error) {
	return s.paymentTypeRepository.ListByWorkTypeID(ctx, workTypeID)
}

func (s *ProfileService) ListWorkCategoryPaymentTypes(ctx context.Context, categoryID string) ([]model.PaymentType, error) {
	return s.paymentTypeRepository.ListByCategoryID(ctx, categoryID)
}

func (s *ProfileService) ListWorkTypeSkills(ctx context.Context, workTypeID string) ([]model.Skill, error) {
	return s.skillRepository.ListByWorkTypeID(ctx, workTypeID)
}
