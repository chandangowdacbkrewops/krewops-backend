package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/repository"
)

var (
	ErrWorkerProfileNotCompleted  = errors.New("user has not completed worker profile")
	ErrWorkNotFound               = errors.New("work posting not found")
	ErrWorkNotPublished           = errors.New("work posting is not published")
	ErrCannotApplyToOwnWork       = errors.New("cannot apply to your own work posting")
	ErrDuplicateApplication       = repository.ErrDuplicateApplication
	ErrApplicationNotFound        = errors.New("application not found")
	ErrNotWorkOwner               = errors.New("only the work owner can perform this action")
	ErrNotApplicationWorker       = errors.New("only the applicant can perform this action")
	ErrInvalidApplicationStatus   = errors.New("application cannot be updated from its current status")
	ErrApplicationAlreadyAccepted = repository.ErrApplicationAlreadyAccepted
)

func (s *WorkService) ApplyToWork(
	ctx context.Context,
	userID string,
	workID string,
	req model.ApplyToWorkRequest,
) (*model.WorkApplication, error) {
	log.Printf("apply to work started: user_id=%s work_id=%s", userID, workID)

	if strings.TrimSpace(workID) == "" {
		return nil, fmt.Errorf("work_id is required")
	}

	completed, err := s.workOwnerRepository.HasCompletedWorkerProfile(ctx, userID)
	if err != nil {
		log.Printf("apply to work failed: user_id=%s work_id=%s stage=worker_profile_check error=%v", userID, workID, err)
		return nil, err
	}
	if !completed {
		log.Printf("apply to work rejected: user_id=%s work_id=%s reason=worker_profile_incomplete", userID, workID)
		return nil, ErrWorkerProfileNotCompleted
	}

	work, err := s.workRepository.FindByID(ctx, workID)
	if err != nil {
		log.Printf("apply to work failed: user_id=%s work_id=%s stage=find_work error=%v", userID, workID, err)
		return nil, err
	}
	if work == nil {
		return nil, ErrWorkNotFound
	}
	if work.UserID == userID {
		return nil, ErrCannotApplyToOwnWork
	}
	if work.Status != model.StatusPublished {
		return nil, ErrWorkNotPublished
	}

	quotation, err := normalizeQuotation(req.Quotation)
	if err != nil {
		log.Printf("apply to work rejected: user_id=%s work_id=%s stage=quotation_validation error=%v", userID, workID, err)
		return nil, err
	}
	allowed, err := s.paymentTypeRepository.ListAllowedCodes(
		ctx,
		stringValue(work.WorkTypeID),
		stringValue(work.WorkCategoryID),
	)
	if err != nil {
		log.Printf("apply to work failed: user_id=%s work_id=%s stage=payment_types error=%v", userID, workID, err)
		return nil, err
	}
	if !containsFold(allowed, quotation.PriceType) {
		log.Printf("apply to work rejected: user_id=%s work_id=%s stage=quotation_price_type price_type=%s", userID, workID, quotation.PriceType)
		return nil, fmt.Errorf("quotation.price_type is not allowed for this work")
	}

	saved, err := s.applicationRepository.Insert(ctx, &model.WorkApplication{
		WorkID:    workID,
		WorkerID:  userID,
		Status:    model.ApplicationStatusApplied,
		Quotation: quotation,
	})
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateApplication) {
			log.Printf("apply to work rejected: user_id=%s work_id=%s reason=duplicate", userID, workID)
			return nil, ErrDuplicateApplication
		}
		log.Printf("apply to work failed: user_id=%s work_id=%s stage=insert error=%v", userID, workID, err)
		return nil, err
	}
	log.Printf("apply to work succeeded: user_id=%s work_id=%s application_id=%s", userID, workID, saved.ID)

	return saved, nil
}

func (s *WorkService) ListWorkApplications(
	ctx context.Context,
	userID string,
	workID string,
	status *string,
) ([]model.WorkApplication, error) {
	log.Printf("list work applications started: user_id=%s work_id=%s", userID, workID)

	if strings.TrimSpace(workID) == "" {
		return nil, fmt.Errorf("work_id is required")
	}

	var statusFilter *string
	if status != nil && strings.TrimSpace(*status) != "" {
		normalized := strings.ToUpper(strings.TrimSpace(*status))
		if !model.ValidApplicationStatus(normalized) {
			return nil, fmt.Errorf("status is invalid")
		}
		statusFilter = &normalized
	}

	if _, err := s.requireWorkOwner(ctx, userID, workID); err != nil {
		return nil, err
	}

	records, err := s.applicationRepository.ListByWorkID(ctx, workID, statusFilter)
	if err != nil {
		log.Printf("list work applications failed: user_id=%s work_id=%s error=%v", userID, workID, err)
		return nil, err
	}

	return records, nil
}

func (s *WorkService) ListMyApplications(
	ctx context.Context,
	userID string,
	status *string,
) ([]model.MyWorkApplication, error) {
	log.Printf("list my applications started: user_id=%s", userID)

	var statusFilter *string
	if status != nil && strings.TrimSpace(*status) != "" {
		normalized := strings.ToUpper(strings.TrimSpace(*status))
		if !model.ValidApplicationStatus(normalized) {
			return nil, fmt.Errorf("status is invalid")
		}
		statusFilter = &normalized
	}

	applications, err := s.applicationRepository.ListByWorkerID(ctx, userID, statusFilter)
	if err != nil {
		log.Printf("list my applications failed: user_id=%s stage=list error=%v", userID, err)
		return nil, err
	}

	workIDs := make([]string, 0, len(applications))
	seen := make(map[string]struct{}, len(applications))
	for _, application := range applications {
		if _, ok := seen[application.WorkID]; ok {
			continue
		}
		seen[application.WorkID] = struct{}{}
		workIDs = append(workIDs, application.WorkID)
	}

	works, err := s.workRepository.FindByIDs(ctx, workIDs)
	if err != nil {
		log.Printf("list my applications failed: user_id=%s stage=works error=%v", userID, err)
		return nil, err
	}

	results := make([]model.MyWorkApplication, 0, len(applications))
	for _, application := range applications {
		item := model.MyWorkApplication{Application: application}
		if work, ok := works[application.WorkID]; ok {
			item.Work = model.BuildWorkResponse(&work)
		}
		results = append(results, item)
	}

	return results, nil
}

func (s *WorkService) ShortlistWorkApplication(
	ctx context.Context,
	userID string,
	workID string,
	applicationID string,
) (*model.WorkApplication, error) {
	return s.updateOwnerApplication(
		ctx,
		userID,
		workID,
		applicationID,
		[]string{model.ApplicationStatusApplied},
		model.ApplicationStatusShortlisted,
		"shortlist",
	)
}

func (s *WorkService) RejectWorkApplication(
	ctx context.Context,
	userID string,
	workID string,
	applicationID string,
) (*model.WorkApplication, error) {
	return s.updateOwnerApplication(
		ctx,
		userID,
		workID,
		applicationID,
		[]string{model.ApplicationStatusApplied, model.ApplicationStatusShortlisted},
		model.ApplicationStatusRejected,
		"reject",
	)
}

func (s *WorkService) CancelWorkApplication(
	ctx context.Context,
	userID string,
	workID string,
	applicationID string,
) (*model.WorkApplication, error) {
	return s.updateOwnerApplication(
		ctx,
		userID,
		workID,
		applicationID,
		[]string{
			model.ApplicationStatusApplied,
			model.ApplicationStatusShortlisted,
			model.ApplicationStatusAccepted,
		},
		model.ApplicationStatusCancelled,
		"cancel",
	)
}

func (s *WorkService) WithdrawWorkApplication(
	ctx context.Context,
	userID string,
	workID string,
	applicationID string,
) (*model.WorkApplication, error) {
	log.Printf("withdraw work application started: user_id=%s work_id=%s application_id=%s", userID, workID, applicationID)

	application, err := s.requireApplicationOnWork(ctx, workID, applicationID)
	if err != nil {
		return nil, err
	}
	if application.WorkerID != userID {
		return nil, ErrNotApplicationWorker
	}
	if application.Status != model.ApplicationStatusApplied &&
		application.Status != model.ApplicationStatusShortlisted {
		return nil, ErrInvalidApplicationStatus
	}

	updated, err := s.applicationRepository.UpdateStatus(
		ctx,
		workID,
		applicationID,
		[]string{model.ApplicationStatusApplied, model.ApplicationStatusShortlisted},
		model.ApplicationStatusWithdrawn,
	)
	if err != nil {
		log.Printf("withdraw work application failed: user_id=%s work_id=%s application_id=%s error=%v", userID, workID, applicationID, err)
		return nil, err
	}
	if updated == nil {
		return nil, ErrInvalidApplicationStatus
	}

	return updated, nil
}

func (s *WorkService) AcceptWorkApplication(
	ctx context.Context,
	userID string,
	workID string,
	applicationID string,
) (*model.WorkApplication, error) {
	log.Printf("accept work application started: user_id=%s work_id=%s application_id=%s", userID, workID, applicationID)

	if _, err := s.requireWorkOwner(ctx, userID, workID); err != nil {
		return nil, err
	}

	application, err := s.requireApplicationOnWork(ctx, workID, applicationID)
	if err != nil {
		return nil, err
	}
	if application.Status != model.ApplicationStatusApplied &&
		application.Status != model.ApplicationStatusShortlisted {
		return nil, ErrInvalidApplicationStatus
	}

	updated, err := s.applicationRepository.AcceptAndRejectOthers(ctx, workID, applicationID)
	if err != nil {
		if errors.Is(err, repository.ErrApplicationAlreadyAccepted) {
			return nil, ErrApplicationAlreadyAccepted
		}
		log.Printf("accept work application failed: user_id=%s work_id=%s application_id=%s error=%v", userID, workID, applicationID, err)
		return nil, err
	}
	if updated == nil {
		return nil, ErrInvalidApplicationStatus
	}

	return updated, nil
}

func (s *WorkService) updateOwnerApplication(
	ctx context.Context,
	userID string,
	workID string,
	applicationID string,
	from []string,
	to string,
	action string,
) (*model.WorkApplication, error) {
	log.Printf("%s work application started: user_id=%s work_id=%s application_id=%s", action, userID, workID, applicationID)

	if _, err := s.requireWorkOwner(ctx, userID, workID); err != nil {
		return nil, err
	}

	application, err := s.requireApplicationOnWork(ctx, workID, applicationID)
	if err != nil {
		return nil, err
	}

	allowed := false
	for _, status := range from {
		if application.Status == status {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, ErrInvalidApplicationStatus
	}

	updated, err := s.applicationRepository.UpdateStatus(ctx, workID, applicationID, from, to)
	if err != nil {
		log.Printf("%s work application failed: user_id=%s work_id=%s application_id=%s error=%v", action, userID, workID, applicationID, err)
		return nil, err
	}
	if updated == nil {
		return nil, ErrInvalidApplicationStatus
	}

	return updated, nil
}

func (s *WorkService) requireWorkOwner(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkRecord, error) {
	if strings.TrimSpace(workID) == "" {
		return nil, fmt.Errorf("work_id is required")
	}

	work, err := s.workRepository.FindByID(ctx, workID)
	if err != nil {
		return nil, err
	}
	if work == nil {
		return nil, ErrWorkNotFound
	}
	if work.UserID != userID {
		return nil, ErrNotWorkOwner
	}

	return work, nil
}

func (s *WorkService) requireApplicationOnWork(
	ctx context.Context,
	workID string,
	applicationID string,
) (*model.WorkApplication, error) {
	if strings.TrimSpace(applicationID) == "" {
		return nil, fmt.Errorf("application_id is required")
	}

	application, err := s.applicationRepository.FindByID(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	if application == nil || application.WorkID != workID {
		return nil, ErrApplicationNotFound
	}

	return application, nil
}

func normalizeQuotation(quotation model.Quotation) (model.Quotation, error) {
	if quotation.Amount <= 0 {
		return model.Quotation{}, fmt.Errorf("quotation.amount must be greater than 0")
	}

	currency := strings.ToUpper(strings.TrimSpace(quotation.Currency))
	if currency == "" {
		currency = "INR"
	}

	priceType := strings.ToLower(strings.TrimSpace(quotation.PriceType))
	if priceType == "" {
		return model.Quotation{}, fmt.Errorf("quotation.price_type is required")
	}

	if quotation.EstimatedDurationHours != nil && *quotation.EstimatedDurationHours <= 0 {
		return model.Quotation{}, fmt.Errorf("quotation.estimated_duration_hours must be greater than 0")
	}

	var message *string
	if quotation.Message != nil {
		trimmed := strings.TrimSpace(*quotation.Message)
		if len(trimmed) > 500 {
			return model.Quotation{}, fmt.Errorf("quotation.message must be 500 characters or fewer")
		}
		if trimmed != "" {
			message = &trimmed
		}
	}

	return model.Quotation{
		Amount:                 quotation.Amount,
		Currency:               currency,
		PriceType:              priceType,
		EstimatedDurationHours: quotation.EstimatedDurationHours,
		Message:                message,
	}, nil
}
