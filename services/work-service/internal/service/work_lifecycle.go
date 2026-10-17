package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func (s *WorkService) StartWork(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkResponse, error) {
	work, err := s.requireWorkOwner(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if work.Status == model.StatusInProgress {
		return model.BuildWorkResponse(work), nil
	}
	if work.Status != model.StatusOpen {
		return nil, ErrInvalidWorkStatus
	}
	if !startAllowed(work.StartDate, time.Now().UTC()) {
		return nil, ErrWorkNotReadyToStart
	}

	assignments, err := s.assignmentRepository.ListActiveByWorkID(ctx, workID)
	if err != nil {
		return nil, err
	}
	if err := validateStartAssignments(work, assignments); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	updated, err := s.transitionWorkStatus(ctx, workID, work.Status, model.StatusInProgress, bson.M{"started_at": now}, work.Status)
	if err != nil {
		return nil, err
	}
	return model.BuildWorkResponse(updated), nil
}

func validateStartAssignments(work *model.WorkRecord, assignments []model.WorkAssignment) error {
	hiringMode := model.NormalizeHiringMode(work.HiringMode)
	active := 0
	contractors := 0
	for _, assignment := range assignments {
		if !model.IsActiveAssignment(assignment.Status) {
			continue
		}
		active++
		if model.NormalizeAssignmentType(assignment.AssignmentType) == model.AssignmentTypeContractor {
			contractors++
		}
	}
	if hiringMode == model.HiringModeContractor {
		if contractors != 1 || active != 1 {
			return ErrContractorAssignmentRequired
		}
		return nil
	}
	if active < 1 {
		return ErrNoActiveAssignment
	}
	return nil
}

func (s *WorkService) CompleteWork(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkResponse, error) {
	work, err := s.requireWorkOwner(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if work.Status == model.StatusPaymentPending || work.Status == model.StatusCompleted {
		if _, err := s.ensurePaymentObligations(ctx, work); err != nil {
			return nil, err
		}
		return model.BuildWorkResponse(work), nil
	}
	if work.Status == model.StatusClosed {
		return model.BuildWorkResponse(work), nil
	}
	if work.Status != model.StatusInProgress {
		return nil, ErrInvalidWorkStatus
	}

	if _, err := s.ensurePaymentObligations(ctx, work); err != nil {
		return nil, err
	}

	updated, err := s.transitionWorkStatus(ctx, workID, work.Status, model.StatusPaymentPending, nil, work.Status)
	if err != nil {
		return nil, err
	}
	return model.BuildWorkResponse(updated), nil
}

func (s *WorkService) CloseApplications(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkResponse, error) {
	work, err := s.requireWorkOwner(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if !work.ApplicationsOpen &&
		(work.Status == model.StatusOpen || work.Status == model.StatusInProgress) {
		return model.BuildWorkResponse(work), nil
	}
	if work.Status != model.StatusOpen && work.Status != model.StatusInProgress {
		return nil, ErrInvalidWorkStatus
	}
	if work.WorkersSelected < 1 {
		return nil, fmt.Errorf("workers_selected must be at least 1 before closing applications")
	}

	updated, err := s.workRepository.CloseApplications(ctx, workID)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrInvalidWorkStatus
	}
	return model.BuildWorkResponse(updated), nil
}

func (s *WorkService) ConfirmWorkCompletion(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkResponse, error) {
	return s.CompleteWork(ctx, userID, workID)
}

func (s *WorkService) CompleteJob(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkResponse, error) {
	return s.CompleteWork(ctx, userID, workID)
}

func (s *WorkService) CloseWork(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkResponse, error) {
	work, err := s.requireWorkOwner(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if work.Status == model.StatusClosed {
		return model.BuildWorkResponse(work), nil
	}
	if work.Status != model.StatusPaymentPending && work.Status != model.StatusCompleted {
		return nil, ErrInvalidWorkStatus
	}

	obligations, err := s.ensurePaymentObligations(ctx, work)
	if err != nil {
		return nil, err
	}
	if !allObligationsResolved(obligations) {
		return nil, ErrPaymentsIncomplete
	}

	updated, err := s.transitionWorkStatus(ctx, workID, work.Status, model.StatusClosed, nil, work.Status)
	if err != nil {
		return nil, err
	}
	return model.BuildWorkResponse(updated), nil
}

func (s *WorkService) CancelWork(
	ctx context.Context,
	userID string,
	workID string,
) (*model.WorkResponse, error) {
	work, err := s.requireWorkOwner(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	if work.Status == model.StatusCancelled {
		return model.BuildWorkResponse(work), nil
	}
	if !model.CanTransition(work.Status, model.StatusCancelled) {
		return nil, ErrInvalidWorkStatus
	}

	updated, err := s.transitionWorkStatus(ctx, workID, work.Status, model.StatusCancelled, nil, work.Status)
	if err != nil {
		log.Printf("cancel work failed: user_id=%s work_id=%s error=%v", userID, workID, err)
		return nil, err
	}

	return model.BuildWorkResponse(updated), nil
}

func (s *WorkService) StartAssignment(
	ctx context.Context,
	userID string,
	assignmentID string,
) (*model.WorkAssignment, error) {
	assignment, work, err := s.requireWorkerAssignment(ctx, userID, "", assignmentID)
	if err != nil {
		return nil, err
	}
	return s.startAssignment(ctx, work, assignment)
}

func (s *WorkService) CompleteAssignment(
	ctx context.Context,
	userID string,
	assignmentID string,
) (*model.WorkAssignment, error) {
	assignment, work, err := s.requireWorkerAssignment(ctx, userID, "", assignmentID)
	if err != nil {
		return nil, err
	}
	return s.completeAssignment(ctx, work, assignment)
}

func (s *WorkService) startAssignment(
	ctx context.Context,
	work *model.WorkRecord,
	assignment *model.WorkAssignment,
) (*model.WorkAssignment, error) {
	if assignment.Status == model.AssignmentStatusInProgress ||
		assignment.Status == model.AssignmentStatusSubmitted ||
		assignment.Status == model.AssignmentStatusApproved {
		return assignment, nil
	}
	if assignment.Status != model.AssignmentStatusAssigned {
		return nil, ErrInvalidAssignmentStatus
	}
	if model.WorkRecordsLocked(work.Status) {
		return nil, ErrInvalidWorkStatus
	}

	now := time.Now().UTC()
	updated, err := s.assignmentRepository.UpdateStatus(
		ctx,
		assignment.ID,
		[]string{model.AssignmentStatusAssigned},
		model.AssignmentStatusInProgress,
		bson.M{"started_at": now},
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrInvalidAssignmentStatus
	}
	return updated, nil
}

func (s *WorkService) completeAssignment(
	ctx context.Context,
	work *model.WorkRecord,
	assignment *model.WorkAssignment,
) (*model.WorkAssignment, error) {
	if assignment.Status == model.AssignmentStatusSubmitted || assignment.Status == model.AssignmentStatusApproved {
		return assignment, nil
	}
	if assignment.Status != model.AssignmentStatusInProgress && assignment.Status != model.AssignmentStatusAssigned {
		return nil, ErrInvalidAssignmentStatus
	}
	// Completing an assignment does not change the work posting. Workers can
	// still submit after the owner has moved the job to payment_pending.
	if model.WorkRecordsLocked(work.Status) {
		return nil, ErrInvalidWorkStatus
	}

	now := time.Now().UTC()
	updated, err := s.assignmentRepository.UpdateStatus(
		ctx,
		assignment.ID,
		[]string{model.AssignmentStatusInProgress, model.AssignmentStatusAssigned, model.AssignmentStatusStarted},
		model.AssignmentStatusSubmitted,
		bson.M{"completed_at": now},
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrInvalidAssignmentStatus
	}
	return updated, nil
}

func (s *WorkService) cancelAssignmentForApplication(
	ctx context.Context,
	work *model.WorkRecord,
	application *model.WorkApplication,
) error {
	assignment, err := s.assignmentRepository.FindByWorkIDAndWorkerID(ctx, work.ID, application.WorkerID)
	if err != nil {
		return err
	}
	if assignment != nil && assignment.Status != model.AssignmentStatusCancelled {
		if assignment.Status != model.AssignmentStatusAssigned {
			return ErrInvalidAssignmentStatus
		}
		if _, err := s.assignmentRepository.UpdateStatus(
			ctx,
			assignment.ID,
			[]string{model.AssignmentStatusAssigned},
			model.AssignmentStatusCancelled,
			nil,
		); err != nil {
			return err
		}
		if err := s.workRepository.ReleaseSeat(ctx, work.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *WorkService) transitionWorkStatus(
	ctx context.Context,
	workID string,
	current string,
	to string,
	extra bson.M,
	from ...string,
) (*model.WorkRecord, error) {
	if !model.CanTransition(current, to) {
		return nil, ErrInvalidWorkStatus
	}

	matchFrom := from
	if len(matchFrom) == 0 {
		matchFrom = []string{current}
	}

	updated, err := s.workRepository.UpdateStatus(ctx, workID, matchFrom, to, extra)
	if err != nil {
		return nil, err
	}
	if updated != nil {
		return updated, nil
	}

	record, err := s.workRepository.FindByID(ctx, workID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrWorkNotFound
	}
	if model.NormalizeWorkStatus(record.Status) == model.NormalizeWorkStatus(to) {
		return record, nil
	}

	return nil, ErrInvalidWorkStatus
}

func (s *WorkService) requireWorkerAssignment(
	ctx context.Context,
	userID string,
	workID string,
	assignmentID string,
) (*model.WorkAssignment, *model.WorkRecord, error) {
	var assignment *model.WorkAssignment
	var err error
	if strings.TrimSpace(assignmentID) != "" {
		assignment, err = s.assignmentRepository.FindByID(ctx, assignmentID)
	} else {
		if strings.TrimSpace(workID) == "" {
			return nil, nil, fmt.Errorf("work_id is required")
		}
		assignment, err = s.assignmentRepository.FindByWorkIDAndWorkerID(ctx, workID, userID)
	}
	if err != nil {
		return nil, nil, err
	}
	if assignment == nil || assignment.Status == model.AssignmentStatusCancelled {
		return nil, nil, ErrNotAssignedWorker
	}
	if assignment.WorkerID != userID && assignment.PayeeID() != userID {
		return nil, nil, ErrNotAssignedWorker
	}

	work, err := s.workRepository.FindByID(ctx, assignment.WorkID)
	if err != nil {
		return nil, nil, err
	}
	if work == nil {
		return nil, nil, ErrWorkNotFound
	}
	work.Status = model.NormalizeWorkStatus(work.Status)
	return assignment, work, nil
}

func allObligationsResolved(obligations []model.PaymentObligation) bool {
	if len(obligations) == 0 {
		return false
	}
	for _, obligation := range obligations {
		if !model.PaymentObligationResolved(obligation) {
			return false
		}
	}
	return true
}

func allPaymentReceiptsConfirmed(obligations []model.PaymentObligation, transactions []model.PaymentTransaction) bool {
	if len(obligations) == 0 {
		return false
	}
	byObligation := make(map[string][]model.PaymentTransaction, len(obligations))
	for _, txn := range transactions {
		byObligation[txn.PaymentObligationID] = append(byObligation[txn.PaymentObligationID], txn)
	}
	for _, obligation := range obligations {
		if obligation.PaidAmountMinor <= 0 {
			continue
		}
		items := byObligation[obligation.ID]
		if len(items) == 0 {
			return false
		}
		for _, txn := range items {
			if txn.ReceiptStatus != model.PaymentReceiptConfirmed {
				return false
			}
		}
	}
	return true
}

func (s *WorkService) maybeCloseWorkIfSettled(ctx context.Context, workID string) (*model.WorkRecord, error) {
	if s.obligationRepository == nil || strings.TrimSpace(workID) == "" {
		return nil, nil
	}
	work, err := s.workRepository.FindByID(ctx, workID)
	if err != nil || work == nil {
		return work, err
	}
	work.Status = model.NormalizeWorkStatus(work.Status)
	if work.Status != model.StatusPaymentPending && work.Status != model.StatusCompleted {
		return work, nil
	}
	obligations, err := s.obligationRepository.ListByWorkID(ctx, workID)
	if err != nil {
		return work, err
	}
	if !allObligationsResolved(obligations) {
		return work, nil
	}
	transactions, err := s.obligationRepository.ListTransactionsByWorkID(ctx, workID)
	if err != nil {
		return work, err
	}
	if !allPaymentReceiptsConfirmed(obligations, transactions) {
		return work, nil
	}
	closed, err := s.transitionWorkStatus(ctx, workID, work.Status, model.StatusClosed, nil, work.Status)
	if err != nil {
		return work, err
	}
	return closed, nil
}

func startAllowed(startDate *time.Time, now time.Time) bool {
	if startDate == nil {
		return true
	}

	scheduled := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return !scheduled.After(today)
}

func allActiveAssignmentsCompleted(assignments []model.WorkAssignment) bool {
	if len(assignments) == 0 {
		return false
	}
	for _, assignment := range assignments {
		status := model.NormalizeAssignmentStatus(assignment.Status)
		if status != model.AssignmentStatusSubmitted && status != model.AssignmentStatusApproved {
			return false
		}
	}
	return true
}

func allAssignmentsHavePaidRecord(assignments []model.WorkAssignment, paidIDs map[string]bool) bool {
	if len(assignments) == 0 {
		return false
	}
	for _, assignment := range assignments {
		if !paidIDs[assignment.ID] {
			return false
		}
	}
	return true
}
