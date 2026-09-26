package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/repository"
)

var ErrOnboardingNotCompleted = errors.New("user has not completed onboarding")

type WorkService struct {
	workRepository          *repository.WorkRepository
	applicationRepository   *repository.ApplicationRepository
	workOwnerRepository     *repository.WorkOwnerRepository
	workTypeRepository      *repository.WorkTypeRepository
	workCategoryRepository  *repository.WorkCategoryRepository
	workTypeFieldRepository *repository.WorkTypeFieldRepository
	paymentTypeRepository   *repository.PaymentTypeRepository
}

func NewWorkService(
	workRepository *repository.WorkRepository,
	applicationRepository *repository.ApplicationRepository,
	workOwnerRepository *repository.WorkOwnerRepository,
	workTypeRepository *repository.WorkTypeRepository,
	workCategoryRepository *repository.WorkCategoryRepository,
	workTypeFieldRepository *repository.WorkTypeFieldRepository,
	paymentTypeRepository *repository.PaymentTypeRepository,
) *WorkService {
	return &WorkService{
		workRepository:          workRepository,
		applicationRepository:   applicationRepository,
		workOwnerRepository:     workOwnerRepository,
		workTypeRepository:      workTypeRepository,
		workCategoryRepository:  workCategoryRepository,
		workTypeFieldRepository: workTypeFieldRepository,
		paymentTypeRepository:   paymentTypeRepository,
	}
}

func (s *WorkService) CreateWork(
	ctx context.Context,
	userID string,
	req model.CreateWorkRequest,
) (*model.WorkResponse, error) {
	log.Printf("create work service started: user_id=%s", userID)

	isOnboardingComplete, err := s.workOwnerRepository.HasCompletedOnboarding(ctx, userID)
	if err != nil {
		log.Printf("create work service failed: user_id=%s stage=onboarding_check error=%v", userID, err)
		return nil, err
	}
	if !isOnboardingComplete {
		log.Printf("create work rejected: user_id=%s reason=onboarding_incomplete", userID)
		return nil, ErrOnboardingNotCompleted
	}
	log.Printf("create work onboarding check passed: user_id=%s", userID)

	status := model.StatusPublished
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		status = strings.ToLower(strings.TrimSpace(*req.Status))
	}
	if !model.ValidStatus(status) {
		log.Printf("create work rejected: user_id=%s stage=status_validation status=%s", userID, status)
		return nil, fmt.Errorf("status must be draft or published")
	}

	startDate, err := parseOptionalDate(req.StartDate)
	if err != nil {
		log.Printf("create work rejected: user_id=%s stage=start_date_validation error=%v", userID, err)
		return nil, err
	}

	if status == model.StatusPublished {
		if err := validateForPublish(req, startDate); err != nil {
			log.Printf("create work rejected: user_id=%s stage=publish_validation error=%v", userID, err)
			return nil, err
		}
	} else if err := validateOptionalFields(req, startDate); err != nil {
		log.Printf("create work rejected: user_id=%s stage=draft_validation error=%v", userID, err)
		return nil, err
	}

	if req.Title != nil && len(strings.TrimSpace(*req.Title)) > 80 {
		return nil, fmt.Errorf("title must be 80 characters or fewer")
	}
	if req.Description != nil && len(strings.TrimSpace(*req.Description)) > 500 {
		return nil, fmt.Errorf("description must be 500 characters or fewer")
	}
	if req.PaymentNotes != nil && len(strings.TrimSpace(*req.PaymentNotes)) > 300 {
		return nil, fmt.Errorf("payment_notes must be 300 characters or fewer")
	}

	var workTypeName *string
	var workTypeCategoryID string
	fields := []model.WorkTypeField{}
	if req.WorkTypeID != nil && strings.TrimSpace(*req.WorkTypeID) != "" {
		log.Printf("create work work type lookup: user_id=%s work_type_id=%s", userID, strings.TrimSpace(*req.WorkTypeID))
		workType, err := s.workTypeRepository.FindByID(ctx, strings.TrimSpace(*req.WorkTypeID))
		if err != nil {
			log.Printf("create work failed: user_id=%s stage=work_type_lookup error=%v", userID, err)
			return nil, err
		}
		if workType == nil {
			log.Printf("create work rejected: user_id=%s reason=invalid_work_type work_type_id=%s", userID, strings.TrimSpace(*req.WorkTypeID))
			return nil, fmt.Errorf("work_type_id is invalid")
		}
		workTypeName = &workType.Name
		workTypeCategoryID = workType.CategoryID

		fields, err = s.workTypeFieldRepository.ListActiveByWorkTypeID(ctx, strings.TrimSpace(*req.WorkTypeID))
		if err != nil {
			log.Printf("create work failed: user_id=%s stage=work_type_fields error=%v", userID, err)
			return nil, err
		}
	}

	var workCategoryName *string
	if req.WorkCategoryID != nil && strings.TrimSpace(*req.WorkCategoryID) != "" {
		categoryID := strings.TrimSpace(*req.WorkCategoryID)
		log.Printf("create work work category lookup: user_id=%s work_category_id=%s", userID, categoryID)
		category, err := s.workCategoryRepository.FindByID(ctx, categoryID)
		if err != nil {
			log.Printf("create work failed: user_id=%s stage=work_category_lookup error=%v", userID, err)
			return nil, err
		}
		if category == nil {
			log.Printf("create work rejected: user_id=%s reason=invalid_work_category work_category_id=%s", userID, categoryID)
			return nil, fmt.Errorf("work_category_id is invalid")
		}
		if workTypeCategoryID != "" && workTypeCategoryID != category.ID {
			log.Printf("create work rejected: user_id=%s reason=work_type_category_mismatch work_type_id=%s work_category_id=%s", userID, strings.TrimSpace(*req.WorkTypeID), categoryID)
			return nil, fmt.Errorf("work_type_id does not belong to work_category_id")
		}
		workCategoryName = &category.Name
	}

	if err := s.validatePaymentType(
		ctx,
		req.PaymentType,
		stringValue(trimmedOrNil(req.WorkTypeID)),
		firstNonEmpty(stringValue(trimmedOrNil(req.WorkCategoryID)), workTypeCategoryID),
	); err != nil {
		log.Printf("create work rejected: user_id=%s stage=payment_type_validation error=%v", userID, err)
		return nil, err
	}

	attributes := normalizeAttributes(req.Attributes)
	if err := validateAttributes(status, fields, attributes); err != nil {
		log.Printf("create work rejected: user_id=%s stage=attributes_validation error=%v", userID, err)
		return nil, err
	}

	var publishedAt *time.Time
	if status == model.StatusPublished {
		now := time.Now().UTC()
		publishedAt = &now
	}

	record := &model.WorkRecord{
		UserID:           userID,
		Title:            trimmedOrNil(req.Title),
		WorkTypeID:       trimmedOrNil(req.WorkTypeID),
		WorkTypeName:     workTypeName,
		WorkCategoryID:   trimmedOrNil(req.WorkCategoryID),
		WorkCategoryName: workCategoryName,
		Description:      trimmedOrNil(req.Description),
		Attributes:       attributes,

		Address:   trimmedOrNil(req.Address),
		City:      trimmedOrNil(req.City),
		State:     trimmedOrNil(req.State),
		Latitude:  req.Latitude,
		Longitude: req.Longitude,

		WorkersNeeded:     req.WorkersNeeded,
		ExperienceLevel:   trimmedOrNil(req.ExperienceLevel),
		Skills:            normalizeSkills(req.Skills),
		ToolsProvided:     req.ToolsProvided,
		MaterialsProvided: req.MaterialsProvided,

		StartDate:     startDate,
		DurationValue: req.DurationValue,
		DurationUnit:  lowerOrNil(req.DurationUnit),
		ShiftTiming:   trimmedOrNil(req.ShiftTiming),

		PaymentType:           lowerOrNil(req.PaymentType),
		BudgetRate:            req.BudgetRate,
		PaymentNotes:          trimmedOrNil(req.PaymentNotes),
		AccommodationProvided: req.AccommodationProvided,
		MealsProvided:         req.MealsProvided,

		Status:      status,
		PublishedAt: publishedAt,
	}

	saved, err := s.workRepository.InsertWork(ctx, record)
	if err != nil {
		log.Printf("create work failed: user_id=%s stage=insert error=%v", userID, err)
		return nil, err
	}
	log.Printf("create work inserted: user_id=%s work_id=%s status=%s", userID, saved.ID, saved.Status)

	return model.BuildWorkResponse(saved), nil
}

func (s *WorkService) ListMyWorks(
	ctx context.Context,
	userID string,
	status *string,
) ([]*model.WorkResponse, error) {
	log.Printf("list my works started: user_id=%s", userID)

	var statusFilter *string
	if status != nil && strings.TrimSpace(*status) != "" {
		normalized := strings.ToLower(strings.TrimSpace(*status))
		if !model.ValidStatus(normalized) {
			return nil, fmt.Errorf("status must be draft or published")
		}
		statusFilter = &normalized
	}

	records, err := s.workRepository.ListByUserID(ctx, userID, statusFilter)
	if err != nil {
		log.Printf("list my works failed: user_id=%s error=%v", userID, err)
		return nil, err
	}

	results := make([]*model.WorkResponse, 0, len(records))
	for i := range records {
		results = append(results, model.BuildWorkResponse(&records[i]))
	}

	return results, nil
}

func validateForPublish(req model.CreateWorkRequest, startDate *time.Time) error {
	if isBlank(req.Title) {
		return fmt.Errorf("title is required")
	}
	if isBlank(req.Description) {
		return fmt.Errorf("description is required")
	}
	if isBlank(req.WorkTypeID) {
		return fmt.Errorf("work_type_id is required")
	}
	if isBlank(req.WorkCategoryID) {
		return fmt.Errorf("work_category_id is required")
	}
	if isBlank(req.Address) {
		return fmt.Errorf("address is required")
	}
	if isBlank(req.City) {
		return fmt.Errorf("city is required")
	}
	if isBlank(req.State) {
		return fmt.Errorf("state is required")
	}
	if req.WorkersNeeded == nil || *req.WorkersNeeded < 1 {
		return fmt.Errorf("workers_needed must be at least 1")
	}
	if req.ToolsProvided == nil {
		return fmt.Errorf("tools_provided is required")
	}
	if req.MaterialsProvided == nil {
		return fmt.Errorf("materials_provided is required")
	}
	if startDate == nil {
		return fmt.Errorf("start_date is required")
	}
	if startDate.Before(today()) {
		return fmt.Errorf("start_date must not be in the past")
	}
	if req.DurationValue == nil || *req.DurationValue < 1 {
		return fmt.Errorf("duration_value must be at least 1")
	}
	if isBlank(req.DurationUnit) || !model.ValidDurationUnit(strings.ToLower(strings.TrimSpace(*req.DurationUnit))) {
		return fmt.Errorf("duration_unit must be one of days, weeks, months")
	}
	if isBlank(req.PaymentType) {
		return fmt.Errorf("payment_type is required")
	}
	if req.BudgetRate == nil || *req.BudgetRate <= 0 {
		return fmt.Errorf("budget_rate must be greater than 0")
	}

	return nil
}

func validateOptionalFields(req model.CreateWorkRequest, startDate *time.Time) error {
	if req.WorkersNeeded != nil && *req.WorkersNeeded < 1 {
		return fmt.Errorf("workers_needed must be at least 1")
	}
	if startDate != nil && startDate.Before(today()) {
		return fmt.Errorf("start_date must not be in the past")
	}
	if req.DurationValue != nil && *req.DurationValue < 1 {
		return fmt.Errorf("duration_value must be at least 1")
	}
	if req.DurationUnit != nil && !isBlank(req.DurationUnit) &&
		!model.ValidDurationUnit(strings.ToLower(strings.TrimSpace(*req.DurationUnit))) {
		return fmt.Errorf("duration_unit must be one of days, weeks, months")
	}
	if req.BudgetRate != nil && *req.BudgetRate <= 0 {
		return fmt.Errorf("budget_rate must be greater than 0")
	}

	return nil
}

func parseOptionalDate(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*value))
	if err != nil {
		return nil, fmt.Errorf("start_date must be in YYYY-MM-DD format")
	}

	return &parsed, nil
}

func today() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func isBlank(value *string) bool {
	return value == nil || strings.TrimSpace(*value) == ""
}

func trimmedOrNil(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func lowerOrNil(value *string) *string {
	trimmed := trimmedOrNil(value)
	if trimmed == nil {
		return nil
	}
	lowered := strings.ToLower(*trimmed)
	return &lowered
}

func (s *WorkService) validatePaymentType(
	ctx context.Context,
	paymentType *string,
	workTypeID string,
	categoryID string,
) error {
	if isBlank(paymentType) {
		return nil
	}
	if workTypeID == "" && categoryID == "" {
		return nil
	}

	return s.requireAllowedPaymentType(ctx, strings.ToLower(strings.TrimSpace(*paymentType)), workTypeID, categoryID)
}

func (s *WorkService) requireAllowedPaymentType(
	ctx context.Context,
	code string,
	workTypeID string,
	categoryID string,
) error {
	allowed, err := s.paymentTypeRepository.ListAllowedCodes(ctx, workTypeID, categoryID)
	if err != nil {
		return err
	}
	if !containsFold(allowed, code) {
		return fmt.Errorf("payment_type is not allowed for the selected work type or category")
	}
	return nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func validateAttributes(status string, fields []model.WorkTypeField, attributes map[string]string) error {
	allowed := make(map[string]model.WorkTypeField, len(fields))
	for _, field := range fields {
		allowed[field.FieldKey] = field
	}

	for key := range attributes {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("attributes contains unknown field %s", key)
		}
	}

	if status != model.StatusPublished {
		return nil
	}

	for _, field := range fields {
		if !field.IsRequired {
			continue
		}
		value, ok := attributes[field.FieldKey]
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("attributes.%s is required", field.FieldKey)
		}
	}

	return nil
}

func normalizeAttributes(attributes map[string]string) map[string]string {
	normalized := make(map[string]string, len(attributes))
	for key, value := range attributes {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		normalized[trimmedKey] = strings.TrimSpace(value)
	}
	return normalized
}

func normalizeSkills(skills []string) []string {
	if skills == nil {
		return []string{}
	}

	normalized := make([]string, 0, len(skills))
	for _, skill := range skills {
		trimmed := strings.TrimSpace(skill)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}

	return normalized
}
