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

var (
	ErrWorkerProfileNotCompleted    = errors.New("user has not completed worker profile")
	ErrWorkNotFound                 = errors.New("work posting not found")
	ErrWorkNotPublished             = errors.New("work posting is not published")
	ErrCannotApplyToOwnWork         = errors.New("cannot apply to your own work posting")
	ErrDuplicateApplication         = repository.ErrDuplicateApplication
	ErrApplicationNotFound          = errors.New("application not found")
	ErrNotWorkOwner                 = errors.New("only the work owner can perform this action")
	ErrNotApplicationWorker         = errors.New("only the applicant can perform this action")
	ErrInvalidApplicationStatus     = errors.New("application cannot be updated from its current status")
	ErrApplicationAlreadyAccepted   = repository.ErrApplicationAlreadyAccepted
	ErrContractorRequired           = errors.New("only contractors can apply when more than one worker is required")
	ErrInvalidWorkStatus            = errors.New("work cannot change to the requested status")
	ErrNotAssignedWorker            = errors.New("only the assigned worker can perform this action")
	ErrWorkNotReadyToStart          = errors.New("work cannot be started before the scheduled start date")
	ErrApplicationsClosed           = errors.New("applications are closed for this work")
	ErrNoWorkerSeats                = repository.ErrApplicationAlreadyAccepted
	ErrAssignmentNotFound           = errors.New("assignment not found")
	ErrInvalidAssignmentStatus      = errors.New("assignment cannot be updated from its current status")
	ErrLogsFrozen                   = errors.New("work records cannot be changed after payment is settled or the work is closed")
	ErrAssignmentsNotCompleted      = errors.New("all selected workers must complete their assignments first")
	ErrPaymentsIncomplete           = errors.New("all payment obligations must be paid or waived before the work can be closed")
	ErrInvalidPaymentMethod         = errors.New("payment method must be CASH, UPI, BANK_TRANSFER, or OTHER")
	ErrNoActiveAssignment           = errors.New("at least one active assignment is required to start work")
	ErrContractorAssignmentRequired = errors.New("exactly one active contractor assignment is required to start work")
	ErrContractorAlreadyAssigned    = errors.New("only one active contractor can be assigned to this work")
	ErrHiringModeMismatch           = errors.New("work hiring mode does not allow this assignment type")
	ErrPaymentAdjustmentRequired    = errors.New("changing this record would alter a settled payment; an adjustment workflow is required")
	ErrOverpayment                  = errors.New("recorded payment would exceed the remaining payable amount")
	ErrIndividualWorkersNotAllowed  = errors.New("individual workers cannot be assigned when hiring mode is CONTRACTOR")
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

	completedType, err := s.workOwnerRepository.FindCompletedWorkerType(ctx, userID)
	if err != nil {
		log.Printf("apply to work failed: user_id=%s work_id=%s stage=worker_profile_check error=%v", userID, workID, err)
		return nil, err
	}
	if completedType == "" {
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
	if (work.Status != model.StatusOpen && work.Status != model.StatusInProgress) || !work.ApplicationsOpen {
		return nil, ErrWorkNotPublished
	}

	quotation, err := normalizeQuotation(req.Quotation)
	if err != nil {
		log.Printf("apply to work rejected: user_id=%s work_id=%s stage=quotation_validation error=%v", userID, workID, err)
		return nil, err
	}
	allowed, err := s.rateTypeRepository.Resolve(
		ctx,
		stringValue(work.WorkTypeID),
		stringValue(work.WorkSegmentID),
		stringValue(work.WorkCategoryID),
	)
	if err != nil {
		log.Printf("apply to work failed: user_id=%s work_id=%s stage=rate_types error=%v", userID, workID, err)
		return nil, err
	}
	unit := ""
	if quotation.RateUnit != nil {
		unit = *quotation.RateUnit
	}
	if !repository.RateOptionAllowed(allowed, quotation.RateBasis, unit) {
		log.Printf("apply to work rejected: user_id=%s work_id=%s stage=quotation_rate_option rate_basis=%s rate_unit=%s", userID, workID, quotation.RateBasis, unit)
		return nil, fmt.Errorf("quotation.rate_basis and rate_unit are not allowed for this work")
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

func workersNeeded(work *model.WorkRecord) int {
	if work == nil || work.WorkersNeeded == nil || *work.WorkersNeeded < 1 {
		return 1
	}
	return *work.WorkersNeeded
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

	assignments, err := s.assignmentRepository.ListByWorkerID(ctx, userID)
	if err != nil {
		return nil, err
	}
	assignmentByWork := make(map[string]model.WorkAssignment, len(assignments))
	for _, assignment := range assignments {
		assignmentByWork[assignment.WorkID] = assignment
	}

	results := make([]model.MyWorkApplication, 0, len(applications))
	for _, application := range applications {
		item := model.MyWorkApplication{Application: application}
		if work, ok := works[application.WorkID]; ok {
			status := model.NormalizeWorkStatus(work.Status)
			if status == model.StatusPaymentPending || status == model.StatusCompleted {
				if updated, err := s.maybeCloseWorkIfSettled(ctx, work.ID); err != nil {
					log.Printf("close settled work failed: user_id=%s work_id=%s error=%v", userID, work.ID, err)
				} else if updated != nil {
					work = *updated
				}
			}
			item.Work = model.BuildWorkResponse(&work)
		}
		if assignment, ok := assignmentByWork[application.WorkID]; ok && assignment.ApplicationID == application.ID {
			copied := assignment
			if err := s.hydrateAssignmentForRead(ctx, &copied); err != nil {
				return nil, err
			}
			item.Assignment = &copied
			payments, err := s.legacyPaymentsForAssignment(ctx, assignment.ID)
			if err != nil {
				return nil, err
			}
			item.Payments = payments
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
	work, err := s.requireWorkOwner(ctx, userID, workID)
	if err != nil {
		return nil, err
	}

	application, err := s.requireApplicationOnWork(ctx, workID, applicationID)
	if err != nil {
		return nil, err
	}
	if application.Status == model.ApplicationStatusAccepted &&
		work.Status != model.StatusOpen &&
		work.Status != model.StatusInProgress {
		return nil, ErrInvalidWorkStatus
	}

	updated, err := s.updateOwnerApplication(
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
	if err != nil {
		return nil, err
	}

	if application.Status == model.ApplicationStatusAccepted {
		if err := s.cancelAssignmentForApplication(ctx, work, application); err != nil {
			return nil, err
		}
	}

	return updated, nil
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

	work, err := s.requireWorkOwner(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if work.Status != model.StatusOpen && work.Status != model.StatusInProgress {
		return nil, ErrInvalidWorkStatus
	}
	hiringMode := model.NormalizeHiringMode(work.HiringMode)
	if hiringMode == model.HiringModeContractor {
		existing, err := s.assignmentRepository.FindActiveContractorByWorkID(ctx, workID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, ErrContractorAlreadyAssigned
		}
	}

	application, err := s.requireApplicationOnWork(ctx, workID, applicationID)
	if err != nil {
		return nil, err
	}
	if application.Status == model.ApplicationStatusAccepted {
		return application, nil
	}
	if !work.ApplicationsOpen {
		return nil, ErrApplicationsClosed
	}
	if application.Status != model.ApplicationStatusApplied &&
		application.Status != model.ApplicationStatusShortlisted {
		return nil, ErrInvalidApplicationStatus
	}

	claimed, err := s.workRepository.ClaimSeat(ctx, workID)
	if err != nil {
		return nil, err
	}
	if claimed == nil {
		return nil, ErrNoWorkerSeats
	}

	updated, err := s.applicationRepository.AcceptOne(ctx, workID, applicationID)
	if err != nil {
		_ = s.workRepository.ReleaseSeat(ctx, workID)
		log.Printf("accept work application failed: user_id=%s work_id=%s application_id=%s error=%v", userID, workID, applicationID, err)
		return nil, err
	}
	if updated == nil {
		_ = s.workRepository.ReleaseSeat(ctx, workID)
		return nil, ErrInvalidApplicationStatus
	}

	assignment := buildAssignmentFromAcceptance(claimed, updated)
	if _, err := s.assignmentRepository.Insert(ctx, assignment); err != nil {
		if errors.Is(err, repository.ErrDuplicateAssignment) {
			return updated, nil
		}
		log.Printf("accept work application failed: user_id=%s work_id=%s application_id=%s stage=assignment error=%v", userID, workID, applicationID, err)
		return nil, err
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

	work.Status = model.NormalizeWorkStatus(work.Status)
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

	rateBasis := strings.ToLower(strings.TrimSpace(quotation.RateBasis))
	if rateBasis == "" {
		return model.Quotation{}, fmt.Errorf("quotation.rate_basis is required")
	}
	var rateUnit *string
	if quotation.RateUnit != nil {
		trimmed := strings.ToLower(strings.TrimSpace(*quotation.RateUnit))
		if trimmed != "" {
			rateUnit = &trimmed
		}
	}
	if rateBasis == "negotiable" && rateUnit != nil {
		return model.Quotation{}, fmt.Errorf("quotation.rate_unit must be empty when rate_basis is negotiable")
	}
	if rateBasis != "negotiable" && rateUnit == nil {
		return model.Quotation{}, fmt.Errorf("quotation.rate_unit is required")
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
		RateBasis:              rateBasis,
		RateUnit:               rateUnit,
		EstimatedDurationHours: quotation.EstimatedDurationHours,
		Message:                message,
	}, nil
}

func buildAssignmentFromAcceptance(work *model.WorkRecord, application *model.WorkApplication) *model.WorkAssignment {
	hiringMode := model.NormalizeHiringMode(work.HiringMode)
	assignmentType := model.AssignmentTypeIndividual
	if hiringMode == model.HiringModeContractor {
		assignmentType = model.AssignmentTypeContractor
	}
	joinedAt := time.Now().UTC().Format("2006-01-02")
	assignment := &model.WorkAssignment{
		WorkID:         work.ID,
		ApplicationID:  application.ID,
		AssignmentType: assignmentType,
		WorkerID:       application.WorkerID,
		Status:         model.AssignmentStatusAssigned,
		Quotation:      application.Quotation,
		AgreedPayment:  model.AgreedPaymentFromQuotation(application.Quotation, hiringMode),
		JoinedAt:       joinedAt,
		PaymentStatus:  model.PaymentStatusDue,
	}
	if assignmentType == model.AssignmentTypeContractor {
		assignment.ContractorProfileID = application.WorkerID
		assignment.CommittedWorkerCount = workersNeeded(work)
		if qty := model.CompletedQuantityFromAttributes(work.Attributes); qty > 0 {
			assignment.CompletedQuantity = qty
		}
	} else {
		assignment.WorkerProfileID = application.WorkerID
	}
	return assignment
}
