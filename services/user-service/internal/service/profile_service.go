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
	workSegmentRepository   *repository.WorkSegmentRepository
	workTypeFieldRepository *repository.WorkTypeFieldRepository
	rateTypeRepository      *repository.RateTypeRepository
	skillRepository         *repository.SkillRepository
	commonRoleRepository    *repository.CommonRoleRepository
}

var (
	ErrInvalidProfile          = errors.New("first_name, last_name, country, state, city, and postal_code are required")
	ErrInvalidGender           = errors.New("gender must be male, female, or other")
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

func normalizeGender(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	gender := strings.ToLower(strings.TrimSpace(*value))
	if gender == "" {
		return nil, nil
	}
	if !model.ValidGender(gender) {
		return nil, ErrInvalidGender
	}
	return &gender, nil
}

func normalizeExpectedRate(req *model.CreateWorkerProfileRequest) error {
	min := req.ExpectedRateMin
	max := req.ExpectedRateMax
	if min == nil && max == nil && req.ExpectedRate != nil {
		min = req.ExpectedRate
		max = req.ExpectedRate
	}
	if min == nil && max == nil {
		return nil
	}
	if min == nil || max == nil {
		return invalidWorkerSelection("expected_rate_min and expected_rate_max are required together")
	}
	if *min <= 0 || *max <= 0 {
		return invalidWorkerSelection("expected_rate_min and expected_rate_max must be greater than 0")
	}
	if *max < *min {
		return invalidWorkerSelection("expected_rate_max must be greater than or equal to expected_rate_min")
	}

	req.ExpectedRateMin = min
	req.ExpectedRateMax = max
	req.ExpectedRate = min
	return nil
}

func NewProfileService(
	profileRepository *repository.ProfileRepository,
	ownerService *OwnerService,
	workTypeRepository *repository.WorkTypeRepository,
	workCategoryRepository *repository.WorkCategoryRepository,
	workSegmentRepository *repository.WorkSegmentRepository,
	workTypeFieldRepository *repository.WorkTypeFieldRepository,
	rateTypeRepository *repository.RateTypeRepository,
	skillRepository *repository.SkillRepository,
	commonRoleRepository *repository.CommonRoleRepository,
) *ProfileService {
	return &ProfileService{
		profileRepository:       profileRepository,
		ownerService:            ownerService,
		workTypeRepository:      workTypeRepository,
		workCategoryRepository:  workCategoryRepository,
		workSegmentRepository:   workSegmentRepository,
		workTypeFieldRepository: workTypeFieldRepository,
		rateTypeRepository:      rateTypeRepository,
		skillRepository:         skillRepository,
		commonRoleRepository:    commonRoleRepository,
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
		Gender:              profile.Gender,
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

	resp.WorkCategories = worker.WorkCategories
	resp.ExpectedRateMin = worker.ExpectedRateMin
	resp.ExpectedRateMax = worker.ExpectedRateMax
	resp.RateBasis = worker.RateBasis
	resp.RateUnit = worker.RateUnit
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
	gender, err := normalizeGender(req.Gender)
	if err != nil {
		return nil, err
	}
	req.Gender = gender

	profile, err := s.profileRepository.CreateProfile(ctx, authUserID, req)
	if err != nil {
		return nil, err
	}

	record := &model.ProfileRecord{
		ID:                  profile.ID,
		AuthUserID:          profile.AuthUserID,
		FirstName:           profile.FirstName,
		LastName:            profile.LastName,
		Gender:              profile.Gender,
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
	gender, err := normalizeGender(req.Gender)
	if err != nil {
		return nil, err
	}
	req.Gender = gender

	profile, err := s.profileRepository.UpdateProfile(ctx, authUserID, req)
	if err != nil {
		return nil, err
	}

	return BuildProfileResponse(&model.ProfileRecord{
		ID:                  profile.ID,
		AuthUserID:          profile.AuthUserID,
		FirstName:           profile.FirstName,
		LastName:            profile.LastName,
		Gender:              profile.Gender,
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
	if err := normalizeRatePair(&req.RateBasis, &req.RateUnit); err != nil {
		return nil, err
	}
	if req.AvailabilityStatus != nil {
		value := strings.ToLower(strings.TrimSpace(*req.AvailabilityStatus))
		req.AvailabilityStatus = &value
	}
	if err := normalizeExpectedRate(&req); err != nil {
		return nil, err
	}

	selections, err := s.validateOnboardingSelections(ctx, req.WorkerType, req.Selections)
	if err != nil {
		return nil, err
	}
	req.Selections = selections
	if err := s.validateWorkerRate(ctx, req.RateBasis, req.RateUnit, selections); err != nil {
		return nil, err
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

	selections, err := s.validateProfileSelections(ctx, profile.WorkerType, req.Selections)
	if err != nil {
		return nil, err
	}
	req.Selections = selections
	if err := normalizeRatePair(&req.RateBasis, &req.RateUnit); err != nil {
		return nil, err
	}
	if err := s.validateWorkerRate(ctx, req.RateBasis, req.RateUnit, selections); err != nil {
		return nil, err
	}

	return s.profileRepository.UpdateWorkerSelections(ctx, authUserID, req)
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
	seenSegments := map[string]struct{}{}
	segmentCount := 0

	for _, selection := range selections {
		categoryID := strings.TrimSpace(selection.WorkCategoryID)
		if categoryID == "" {
			return nil, invalidWorkerSelection("work_category_id is required")
		}
		if _, exists := seenCategories[categoryID]; exists {
			return nil, invalidWorkerSelection("duplicate work_category_id")
		}
		seenCategories[categoryID] = struct{}{}

		if len(selection.Segments) == 0 {
			return nil, invalidWorkerSelection("at least one work segment is required for each category")
		}

		normalizedSegments := make([]model.WorkerSegmentSelection, 0, len(selection.Segments))
		for _, segment := range selection.Segments {
			segmentID := strings.TrimSpace(segment.WorkSegmentID)
			if segmentID == "" {
				return nil, invalidWorkerSelection("work_segment_id is required")
			}
			if _, exists := seenSegments[segmentID]; exists {
				return nil, invalidWorkerSelection("duplicate work_segment_id")
			}
			seenSegments[segmentID] = struct{}{}
			segmentCount++
			normalizedSegments = append(normalizedSegments, model.WorkerSegmentSelection{
				WorkSegmentID: segmentID,
			})
		}

		normalized = append(normalized, model.WorkerCategorySelection{
			WorkCategoryID: categoryID,
			Segments:       normalizedSegments,
		})
	}

	if workerType == "individual" {
		if len(normalized) != 1 {
			return nil, invalidWorkerSelection("individual workers can select only one work category")
		}
		if segmentCount != 1 {
			return nil, invalidWorkerSelection("individual workers can select only one work segment")
		}
	}

	return normalized, nil
}

func normalizeProfileSelections(
	workerType string,
	selections []model.WorkerCategorySelection,
) ([]model.WorkerCategorySelection, []string, []string, []string, error) {
	if workerType != "individual" && workerType != "contractor" {
		return nil, nil, nil, nil, invalidWorkerSelection("worker_type must be individual or contractor")
	}
	if len(selections) == 0 {
		return nil, nil, nil, nil, invalidWorkerSelection("selections are required")
	}

	normalized := make([]model.WorkerCategorySelection, 0, len(selections))
	seenCategories := map[string]struct{}{}
	seenSegments := map[string]struct{}{}
	seenWorkTypes := map[string]struct{}{}
	seenSkills := map[string]struct{}{}
	segmentIDs := []string{}
	workTypeIDs := []string{}
	skillIDs := []string{}

	for _, selection := range selections {
		categoryID := strings.TrimSpace(selection.WorkCategoryID)
		if categoryID == "" {
			return nil, nil, nil, nil, invalidWorkerSelection("work_category_id is required")
		}
		if _, exists := seenCategories[categoryID]; exists {
			return nil, nil, nil, nil, invalidWorkerSelection("duplicate work_category_id")
		}
		seenCategories[categoryID] = struct{}{}

		if len(selection.Segments) == 0 {
			return nil, nil, nil, nil, invalidWorkerSelection("at least one work segment is required for each category")
		}

		normalizedSegments := make([]model.WorkerSegmentSelection, 0, len(selection.Segments))
		for _, segment := range selection.Segments {
			segmentID := strings.TrimSpace(segment.WorkSegmentID)
			if segmentID == "" {
				return nil, nil, nil, nil, invalidWorkerSelection("work_segment_id is required")
			}
			if _, exists := seenSegments[segmentID]; exists {
				return nil, nil, nil, nil, invalidWorkerSelection("duplicate work_segment_id")
			}
			seenSegments[segmentID] = struct{}{}
			segmentIDs = append(segmentIDs, segmentID)

			if len(segment.WorkTypes) == 0 {
				return nil, nil, nil, nil, invalidWorkerSelection("at least one work type is required for each segment")
			}

			normalizedTypes := make([]model.WorkerTypeSelection, 0, len(segment.WorkTypes))
			for _, workType := range segment.WorkTypes {
				workTypeID := strings.TrimSpace(workType.WorkTypeID)
				if workTypeID == "" {
					return nil, nil, nil, nil, invalidWorkerSelection("work_type_id is required")
				}
				if _, exists := seenWorkTypes[workTypeID]; exists {
					return nil, nil, nil, nil, invalidWorkerSelection("duplicate work_type_id")
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
						return nil, nil, nil, nil, invalidWorkerSelection("duplicate skill_id")
					}
					seenSkills[skillID] = struct{}{}
					normalizedSkills = append(normalizedSkills, skillID)
					skillIDs = append(skillIDs, skillID)
				}
				if len(normalizedSkills) == 0 {
					return nil, nil, nil, nil, invalidWorkerSelection("at least one skill is required for each work type")
				}
				normalizedTypes = append(normalizedTypes, model.WorkerTypeSelection{
					WorkTypeID: workTypeID,
					SkillIDs:   normalizedSkills,
				})
			}

			normalizedSegments = append(normalizedSegments, model.WorkerSegmentSelection{
				WorkSegmentID: segmentID,
				WorkTypes:     normalizedTypes,
			})
		}

		normalized = append(normalized, model.WorkerCategorySelection{
			WorkCategoryID: categoryID,
			Segments:       normalizedSegments,
		})
	}

	if workerType == "individual" {
		if len(normalized) != 1 {
			return nil, nil, nil, nil, invalidWorkerSelection("individual workers can select only one work category")
		}
		if len(normalized[0].Segments) != 1 {
			return nil, nil, nil, nil, invalidWorkerSelection("individual workers can select only one work segment")
		}
		if len(normalized[0].Segments[0].WorkTypes) != 1 {
			return nil, nil, nil, nil, invalidWorkerSelection("individual workers can select only one work type")
		}
		if len(normalized[0].Segments[0].WorkTypes[0].SkillIDs) > 2 {
			return nil, nil, nil, nil, invalidWorkerSelection("individual workers can select up to 2 skills")
		}
	}

	return normalized, segmentIDs, workTypeIDs, skillIDs, nil
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

		for _, segment := range selection.Segments {
			found, err := s.workSegmentRepository.FindByID(ctx, segment.WorkSegmentID)
			if err != nil {
				return nil, err
			}
			if found == nil || !found.IsActive {
				return nil, invalidWorkerSelection("work_segment_id is invalid")
			}
			if found.CategoryID != selection.WorkCategoryID {
				return nil, invalidWorkerSelection("work_segment_id does not belong to work_category_id")
			}
		}
	}

	return normalized, nil
}

func (s *ProfileService) validateProfileSelections(
	ctx context.Context,
	workerType string,
	selections []model.WorkerCategorySelection,
) ([]model.WorkerCategorySelection, error) {
	normalized, segmentIDs, workTypeIDs, skillIDs, err := normalizeProfileSelections(workerType, selections)
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

	segments, err := s.workSegmentRepository.FindByIDs(ctx, segmentIDs)
	if err != nil {
		return nil, err
	}
	for _, selection := range normalized {
		for _, segment := range selection.Segments {
			found, ok := segments[segment.WorkSegmentID]
			if !ok || !found.IsActive {
				return nil, invalidWorkerSelection("work_segment_id is invalid")
			}
			if found.CategoryID != selection.WorkCategoryID {
				return nil, invalidWorkerSelection("work_segment_id does not belong to work_category_id")
			}
		}
	}

	workTypes, err := s.workTypeRepository.FindByIDs(ctx, workTypeIDs)
	if err != nil {
		return nil, err
	}
	for _, selection := range normalized {
		for _, segment := range selection.Segments {
			for _, workType := range segment.WorkTypes {
				found, ok := workTypes[workType.WorkTypeID]
				if !ok || !found.IsActive {
					return nil, invalidWorkerSelection("work_type_id is invalid")
				}
				if found.SegmentID != segment.WorkSegmentID {
					return nil, invalidWorkerSelection("work_type_id does not belong to work_segment_id")
				}
				if found.CategoryID != selection.WorkCategoryID {
					return nil, invalidWorkerSelection("work_type_id does not belong to work_category_id")
				}
			}
		}
	}

	skills, err := s.skillRepository.FindByIDs(ctx, skillIDs)
	if err != nil {
		return nil, err
	}
	for _, selection := range normalized {
		for _, segment := range selection.Segments {
			for _, workType := range segment.WorkTypes {
				if len(workType.SkillIDs) == 0 {
					continue
				}
				for _, skillID := range workType.SkillIDs {
					if found, ok := skills[skillID]; !ok || !found.IsActive {
						return nil, invalidWorkerSelection("skill_id is invalid")
					}
				}
				// Skills are now global/reusable master data; validate
				// membership against the work_type_skills join table
				// instead of a 1:1 Skill.WorkTypeID column.
				allowed, err := s.skillRepository.AllowedForWorkType(ctx, workType.WorkTypeID, workType.SkillIDs)
				if err != nil {
					return nil, err
				}
				for _, skillID := range workType.SkillIDs {
					if !allowed[skillID] {
						return nil, invalidWorkerSelection("skill_id does not belong to work_type_id")
					}
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

func (s *ProfileService) ListWorkSegmentsByCategory(ctx context.Context, categoryID string) ([]model.WorkSegment, error) {
	return s.workSegmentRepository.ListByCategoryID(ctx, categoryID)
}

func (s *ProfileService) ListWorkTypesBySegment(ctx context.Context, segmentID string) ([]model.WorkType, error) {
	return s.workTypeRepository.ListBySegmentID(ctx, segmentID)
}

func (s *ProfileService) ListWorkTypeFields(ctx context.Context, workTypeID string) ([]model.WorkTypeField, error) {
	return s.workTypeFieldRepository.ListByWorkTypeID(ctx, workTypeID)
}

func (s *ProfileService) ListWorkTypeRateTypes(ctx context.Context, workTypeID string) ([]model.RateOption, error) {
	return s.rateTypeRepository.Resolve(ctx, workTypeID, "", "")
}

func (s *ProfileService) ListWorkSegmentRateTypes(ctx context.Context, segmentID string) ([]model.RateOption, error) {
	return s.rateTypeRepository.Resolve(ctx, "", segmentID, "")
}

func (s *ProfileService) ListWorkCategoryRateTypes(ctx context.Context, categoryID string) ([]model.RateOption, error) {
	return s.rateTypeRepository.Resolve(ctx, "", "", categoryID)
}

func normalizeRatePair(basis **string, unit **string) error {
	normalizedBasis := lowerPointer(*basis)
	normalizedUnit := lowerPointer(*unit)
	if normalizedBasis == nil && normalizedUnit == nil {
		*basis = nil
		*unit = nil
		return nil
	}
	if normalizedBasis == nil {
		return invalidWorkerSelection("rate_basis is required when rate_unit is set")
	}
	if *normalizedBasis == "negotiable" && normalizedUnit != nil {
		return invalidWorkerSelection("rate_unit must be empty when rate_basis is negotiable")
	}
	if *normalizedBasis != "negotiable" && normalizedUnit == nil {
		return invalidWorkerSelection("rate_unit is required")
	}
	*basis = normalizedBasis
	*unit = normalizedUnit
	return nil
}

func lowerPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.ToLower(strings.TrimSpace(*value))
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (s *ProfileService) validateWorkerRate(
	ctx context.Context,
	basis *string,
	unit *string,
	selections []model.WorkerCategorySelection,
) error {
	if basis == nil && unit == nil {
		return nil
	}
	if basis == nil {
		return invalidWorkerSelection("rate_basis is required when rate_unit is set")
	}

	basisCode := *basis
	unitCode := ""
	if unit != nil {
		unitCode = *unit
	}

	typeIDs, segmentIDs, categoryIDs := collectSelectionIDs(selections)
	allowed := []model.RateOption{}
	switch {
	case len(typeIDs) > 0:
		for _, id := range typeIDs {
			options, err := s.rateTypeRepository.Resolve(ctx, id, "", "")
			if err != nil {
				return err
			}
			allowed = append(allowed, options...)
		}
	case len(segmentIDs) > 0:
		for _, id := range segmentIDs {
			options, err := s.rateTypeRepository.Resolve(ctx, "", id, "")
			if err != nil {
				return err
			}
			allowed = append(allowed, options...)
		}
	case len(categoryIDs) > 0:
		for _, id := range categoryIDs {
			options, err := s.rateTypeRepository.Resolve(ctx, "", "", id)
			if err != nil {
				return err
			}
			allowed = append(allowed, options...)
		}
	default:
		exists, err := s.rateTypeRepository.ExistsPair(ctx, basisCode, unitCode)
		if err != nil {
			return err
		}
		if !exists {
			return invalidWorkerSelection("rate_basis and rate_unit are not a valid catalog pair")
		}
		return nil
	}

	if !model.RateOptionMatches(allowed, basisCode, unitCode) {
		return invalidWorkerSelection("rate_basis and rate_unit are not allowed for the selected work")
	}
	return nil
}

func collectSelectionIDs(selections []model.WorkerCategorySelection) (typeIDs, segmentIDs, categoryIDs []string) {
	seenTypes := map[string]struct{}{}
	seenSegments := map[string]struct{}{}
	seenCategories := map[string]struct{}{}
	for _, selection := range selections {
		if selection.WorkCategoryID != "" {
			if _, ok := seenCategories[selection.WorkCategoryID]; !ok {
				seenCategories[selection.WorkCategoryID] = struct{}{}
				categoryIDs = append(categoryIDs, selection.WorkCategoryID)
			}
		}
		for _, segment := range selection.Segments {
			if segment.WorkSegmentID != "" {
				if _, ok := seenSegments[segment.WorkSegmentID]; !ok {
					seenSegments[segment.WorkSegmentID] = struct{}{}
					segmentIDs = append(segmentIDs, segment.WorkSegmentID)
				}
			}
			for _, workType := range segment.WorkTypes {
				if workType.WorkTypeID == "" {
					continue
				}
				if _, ok := seenTypes[workType.WorkTypeID]; !ok {
					seenTypes[workType.WorkTypeID] = struct{}{}
					typeIDs = append(typeIDs, workType.WorkTypeID)
				}
			}
		}
	}
	return typeIDs, segmentIDs, categoryIDs
}

func (s *ProfileService) ListWorkTypeSkills(ctx context.Context, workTypeID string) ([]model.Skill, error) {
	return s.skillRepository.ListByWorkTypeID(ctx, workTypeID)
}

func (s *ProfileService) ListWorkTypeCommonRoles(ctx context.Context, workTypeID string) ([]model.CommonRole, error) {
	return s.commonRoleRepository.ListByWorkTypeID(ctx, workTypeID)
}

// GetWorkTypeConfiguration bundles everything the client needs to render a
// work type's dynamic configuration (skills, rate types, common roles, and
// dynamic attribute fields) into a single call.
type WorkTypeConfiguration struct {
	Skills      []model.Skill
	CommonRoles []model.CommonRole
	RateTypes   []model.RateOption
	Fields      []model.WorkTypeField
}

func (s *ProfileService) GetWorkTypeConfiguration(ctx context.Context, workTypeID string) (*WorkTypeConfiguration, error) {
	skills, err := s.skillRepository.ListByWorkTypeID(ctx, workTypeID)
	if err != nil {
		return nil, err
	}
	roles, err := s.commonRoleRepository.ListByWorkTypeID(ctx, workTypeID)
	if err != nil {
		return nil, err
	}
	rateTypes, err := s.rateTypeRepository.Resolve(ctx, workTypeID, "", "")
	if err != nil {
		return nil, err
	}
	fields, err := s.workTypeFieldRepository.ListByWorkTypeID(ctx, workTypeID)
	if err != nil {
		return nil, err
	}
	return &WorkTypeConfiguration{
		Skills:      skills,
		CommonRoles: roles,
		RateTypes:   rateTypes,
		Fields:      fields,
	}, nil
}
