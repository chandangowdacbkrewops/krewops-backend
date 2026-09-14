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
	workRepository      *repository.WorkRepository
	workOwnerRepository *repository.WorkOwnerRepository
	workTypeRepository  *repository.WorkTypeRepository
}

func NewWorkService(
	workRepository *repository.WorkRepository,
	workOwnerRepository *repository.WorkOwnerRepository,
	workTypeRepository *repository.WorkTypeRepository,
) *WorkService {
	return &WorkService{
		workRepository:      workRepository,
		workOwnerRepository: workOwnerRepository,
		workTypeRepository:  workTypeRepository,
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
	}

	var publishedAt *time.Time
	if status == model.StatusPublished {
		now := time.Now().UTC()
		publishedAt = &now
	}

	record := &model.WorkRecord{
		UserID:      userID,
		Title:       trimmedOrNil(req.Title),
		WorkTypeID:  trimmedOrNil(req.WorkTypeID),
		Description: trimmedOrNil(req.Description),

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
	if isBlank(req.PaymentType) || !model.ValidPaymentType(strings.ToLower(strings.TrimSpace(*req.PaymentType))) {
		return fmt.Errorf("payment_type must be one of per_day, hourly, fixed")
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
	if req.PaymentType != nil && !isBlank(req.PaymentType) &&
		!model.ValidPaymentType(strings.ToLower(strings.TrimSpace(*req.PaymentType))) {
		return fmt.Errorf("payment_type must be one of per_day, hourly, fixed")
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
