package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/repository"
)

func (s *WorkService) ListEligibleRecruitmentWorks(
	ctx context.Context,
	ownerID string,
	workerID string,
) ([]model.EligibleRecruitmentWork, error) {
	if strings.TrimSpace(workerID) == "" {
		return nil, fmt.Errorf("worker_id is required")
	}
	open := model.StatusOpen
	works, err := s.workRepository.ListByUserID(ctx, ownerID, &open)
	if err != nil {
		return nil, err
	}
	worker, err := s.workOwnerRepository.FindRecruitmentWorker(ctx, workerID)
	if err != nil {
		return nil, err
	}

	results := make([]model.EligibleRecruitmentWork, 0, len(works))
	for i := range works {
		work := works[i]
		item, err := s.eligibilityForWork(ctx, &work, worker)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, nil
}

func (s *WorkService) CreateWorkInvitations(
	ctx context.Context,
	ownerID string,
	workID string,
	workerIDs []string,
	override *model.OfferedPayment,
) ([]model.InvitationCreateResult, error) {
	work, err := s.requireWorkOwner(ctx, ownerID, workID)
	if err != nil {
		return nil, err
	}
	payment, err := model.ResolveOfferedPayment(work, override)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	results := make([]model.InvitationCreateResult, 0, len(workerIDs))
	for _, rawID := range workerIDs {
		workerID := strings.TrimSpace(rawID)
		if workerID == "" {
			continue
		}
		if _, exists := seen[workerID]; exists {
			results = append(results, model.InvitationCreateResult{
				WorkerID: workerID,
				Outcome:  model.InvitationOutcomeRejected,
				Reason:   model.InvitationReasonDuplicateInRequest,
			})
			continue
		}
		seen[workerID] = struct{}{}
		result, err := s.createOneInvitation(ctx, work, workerID, payment)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("worker_ids is required")
	}
	return results, nil
}

func (s *WorkService) ListWorkInvitations(
	ctx context.Context,
	ownerID string,
	workID string,
) ([]model.WorkInvitation, error) {
	if _, err := s.requireWorkOwner(ctx, ownerID, workID); err != nil {
		return nil, err
	}
	invitations, err := s.invitationRepository.ListByWorkID(ctx, workID)
	if err != nil {
		return nil, err
	}
	invitations, err = s.expireInvitations(ctx, invitations)
	if err != nil {
		return nil, err
	}
	return s.decorateInvitations(ctx, invitations)
}

func (s *WorkService) ListMyInvitations(
	ctx context.Context,
	workerID string,
	status *string,
) ([]model.WorkInvitation, error) {
	if strings.TrimSpace(workerID) == "" {
		return nil, fmt.Errorf("worker_id is required")
	}
	filter := ""
	if status != nil {
		filter = strings.TrimSpace(*status)
	}
	invitations, err := s.invitationRepository.ListByWorkerID(ctx, workerID, filter)
	if err != nil {
		return nil, err
	}
	invitations, err = s.expireInvitations(ctx, invitations)
	if err != nil {
		return nil, err
	}
	if filter != "" && model.NormalizeInvitationStatus(filter) != model.InvitationStatusExpired {
		filtered := make([]model.WorkInvitation, 0, len(invitations))
		for _, invitation := range invitations {
			if invitation.Status == model.NormalizeInvitationStatus(filter) {
				filtered = append(filtered, invitation)
			}
		}
		invitations = filtered
	}
	return s.decorateInvitations(ctx, invitations)
}

func (s *WorkService) AcceptWorkInvitation(
	ctx context.Context,
	workerID string,
	invitationID string,
) (*model.WorkInvitation, error) {
	invitation, err := s.requireInvitation(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if invitation.WorkerID != workerID {
		return nil, ErrNotInvitationWorker
	}
	invitation, err = s.expireInvitation(ctx, invitation)
	if err != nil {
		return nil, err
	}
	if invitation.Status == model.InvitationStatusAccepted {
		decorated, err := s.decorateInvitations(ctx, []model.WorkInvitation{*invitation})
		if err != nil {
			return nil, err
		}
		return &decorated[0], nil
	}
	if invitation.Status == model.InvitationStatusExpired {
		return nil, ErrInvitationExpired
	}
	if !model.InvitationTransitionAllowed(invitation.Status, model.InvitationStatusAccepted) {
		return nil, ErrInvalidInvitationStatus
	}

	work, err := s.workRepository.FindByID(ctx, invitation.WorkID)
	if err != nil {
		return nil, err
	}
	if work == nil {
		return nil, ErrWorkNotFound
	}
	if model.NormalizeWorkStatus(work.Status) != model.StatusOpen {
		return nil, ErrInvalidWorkStatus
	}

	existing, err := s.assignmentRepository.FindAnyByWorkIDAndWorkerID(ctx, work.ID, workerID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrWorkerAlreadyAssigned
	}

	seatCount := invitation.SeatCount
	if seatCount < 1 {
		seatCount = 1
	}
	claimed, err := s.workRepository.ClaimSeats(ctx, work.ID, seatCount)
	if err != nil {
		return nil, err
	}
	if !model.ShouldAcceptAfterSeatClaim(claimed) {
		return nil, ErrOfferedWorkersExceedRemaining
	}

	accepted, err := s.invitationRepository.UpdateStatus(
		ctx,
		invitation.ID,
		[]string{model.InvitationStatusPending},
		model.InvitationStatusAccepted,
		bson.M{"responded_at": time.Now().UTC()},
	)
	if err != nil {
		_ = s.workRepository.ReleaseSeats(ctx, work.ID, seatCount)
		return nil, err
	}
	if accepted == nil {
		_ = s.workRepository.ReleaseSeats(ctx, work.ID, seatCount)
		return nil, ErrInvalidInvitationStatus
	}

	assignment := model.BuildAssignmentFromInvitation(claimed, accepted)
	saved, err := s.assignmentRepository.Insert(ctx, assignment)
	if err != nil {
		_ = s.workRepository.ReleaseSeats(ctx, work.ID, seatCount)
		_, _ = s.invitationRepository.UpdateStatus(
			ctx,
			invitation.ID,
			[]string{model.InvitationStatusAccepted},
			model.InvitationStatusPending,
			bson.M{"responded_at": nil, "assignment_id": ""},
		)
		if errors.Is(err, repository.ErrDuplicateAssignment) {
			return nil, ErrWorkerAlreadyAssigned
		}
		log.Printf("accept invitation failed: invitation_id=%s stage=assignment error=%v", invitation.ID, err)
		return nil, err
	}

	withAssignment, err := s.invitationRepository.UpdateStatus(
		ctx,
		invitation.ID,
		[]string{model.InvitationStatusAccepted},
		model.InvitationStatusAccepted,
		bson.M{"assignment_id": saved.ID},
	)
	if err != nil {
		log.Printf("accept invitation assignment link failed: invitation_id=%s assignment_id=%s error=%v", invitation.ID, saved.ID, err)
		accepted.AssignmentID = saved.ID
		withAssignment = accepted
	} else if withAssignment == nil {
		accepted.AssignmentID = saved.ID
		withAssignment = accepted
	}

	decorated, err := s.decorateInvitations(ctx, []model.WorkInvitation{*withAssignment})
	if err != nil {
		return withAssignment, nil
	}
	return &decorated[0], nil
}

func (s *WorkService) DeclineWorkInvitation(
	ctx context.Context,
	workerID string,
	invitationID string,
) (*model.WorkInvitation, error) {
	invitation, err := s.requireInvitation(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if invitation.WorkerID != workerID {
		return nil, ErrNotInvitationWorker
	}
	invitation, err = s.expireInvitation(ctx, invitation)
	if err != nil {
		return nil, err
	}
	if invitation.Status == model.InvitationStatusDeclined {
		return invitation, nil
	}
	if invitation.Status == model.InvitationStatusExpired {
		return nil, ErrInvitationExpired
	}
	if !model.InvitationTransitionAllowed(invitation.Status, model.InvitationStatusDeclined) {
		return nil, ErrInvalidInvitationStatus
	}
	updated, err := s.invitationRepository.UpdateStatus(
		ctx,
		invitation.ID,
		[]string{model.InvitationStatusPending},
		model.InvitationStatusDeclined,
		bson.M{"responded_at": time.Now().UTC()},
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrInvalidInvitationStatus
	}
	decorated, err := s.decorateInvitations(ctx, []model.WorkInvitation{*updated})
	if err != nil {
		return updated, nil
	}
	return &decorated[0], nil
}

func (s *WorkService) CancelWorkInvitation(
	ctx context.Context,
	ownerID string,
	workID string,
	invitationID string,
) (*model.WorkInvitation, error) {
	if _, err := s.requireWorkOwner(ctx, ownerID, workID); err != nil {
		return nil, err
	}
	invitation, err := s.requireInvitation(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if invitation.WorkID != workID {
		return nil, ErrInvitationNotFound
	}
	invitation, err = s.expireInvitation(ctx, invitation)
	if err != nil {
		return nil, err
	}
	if invitation.Status == model.InvitationStatusCancelled {
		return invitation, nil
	}
	if !model.InvitationTransitionAllowed(invitation.Status, model.InvitationStatusCancelled) {
		return nil, ErrInvalidInvitationStatus
	}
	updated, err := s.invitationRepository.UpdateStatus(
		ctx,
		invitation.ID,
		[]string{model.InvitationStatusPending},
		model.InvitationStatusCancelled,
		bson.M{"responded_at": time.Now().UTC()},
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrInvalidInvitationStatus
	}
	return updated, nil
}

func (s *WorkService) cancelPendingInvitations(ctx context.Context, workID string) error {
	if s.invitationRepository == nil {
		return nil
	}
	return s.invitationRepository.CancelPendingByWorkID(ctx, workID)
}

func (s *WorkService) createOneInvitation(
	ctx context.Context,
	work *model.WorkRecord,
	workerID string,
	payment model.OfferedPayment,
) (model.InvitationCreateResult, error) {
	worker, err := s.workOwnerRepository.FindRecruitmentWorker(ctx, workerID)
	if err != nil {
		return model.InvitationCreateResult{}, err
	}
	pending, err := s.invitationRepository.FindPending(ctx, work.ID, workerID)
	if err != nil {
		return model.InvitationCreateResult{}, err
	}
	if pending != nil {
		pending, err = s.expireInvitation(ctx, pending)
		if err != nil {
			return model.InvitationCreateResult{}, err
		}
		if pending.Status == model.InvitationStatusPending {
			return model.InvitationCreateResult{
				WorkerID:   workerID,
				Outcome:    model.InvitationOutcomeAlreadyPending,
				Invitation: pending,
			}, nil
		}
	}

	eligible, reason, seatCount, err := s.eligibilityForWorker(ctx, work, worker, workerID)
	if err != nil {
		return model.InvitationCreateResult{}, err
	}
	if !eligible {
		return model.InvitationCreateResult{
			WorkerID: workerID,
			Outcome:  model.InvitationOutcomeRejected,
			Reason:   reason,
		}, nil
	}

	created, err := s.invitationRepository.Insert(ctx, &model.WorkInvitation{
		WorkID:         work.ID,
		OwnerID:        work.UserID,
		WorkerID:       workerID,
		Status:         model.InvitationStatusPending,
		ApplicantType:  model.NormalizeApplicantType(worker.WorkerType),
		SeatCount:      seatCount,
		OfferedPayment: payment,
		ExpiresAt:      time.Now().UTC().Add(model.InvitationTTL),
	})
	if errors.Is(err, repository.ErrDuplicatePendingInvitation) {
		existing, findErr := s.invitationRepository.FindPending(ctx, work.ID, workerID)
		if findErr != nil {
			return model.InvitationCreateResult{}, findErr
		}
		return model.InvitationCreateResult{
			WorkerID:   workerID,
			Outcome:    model.InvitationOutcomeAlreadyPending,
			Invitation: existing,
		}, nil
	}
	if err != nil {
		return model.InvitationCreateResult{}, err
	}
	return model.InvitationCreateResult{
		WorkerID:   workerID,
		Outcome:    model.InvitationOutcomeCreated,
		Invitation: created,
	}, nil
}

func (s *WorkService) eligibilityForWork(
	ctx context.Context,
	work *model.WorkRecord,
	worker *model.RecruitmentWorker,
) (model.EligibleRecruitmentWork, error) {
	workerID := ""
	if worker != nil {
		workerID = worker.UserID
	}
	eligible, reason, seatCount, err := s.eligibilityForWorker(ctx, work, worker, workerID)
	if err != nil {
		return model.EligibleRecruitmentWork{}, err
	}
	return model.EligibleRecruitmentWork{
		Work:      model.BuildWorkResponse(work),
		Eligible:  eligible,
		Reason:    reason,
		SeatCount: seatCount,
	}, nil
}

func (s *WorkService) eligibilityForWorker(
	ctx context.Context,
	work *model.WorkRecord,
	worker *model.RecruitmentWorker,
	workerID string,
) (bool, string, int, error) {
	var hasAssignment bool
	var hasPending bool
	var hasContractor bool
	var hasPendingContractor bool
	if workerID != "" {
		assignment, err := s.assignmentRepository.FindAnyByWorkIDAndWorkerID(ctx, work.ID, workerID)
		if err != nil {
			return false, "", 0, err
		}
		hasAssignment = assignment != nil

		pending, err := s.invitationRepository.FindPending(ctx, work.ID, workerID)
		if err != nil {
			return false, "", 0, err
		}
		if pending != nil {
			pending, err = s.expireInvitation(ctx, pending)
			if err != nil {
				return false, "", 0, err
			}
			hasPending = pending.Status == model.InvitationStatusPending
		}

		contractor, err := s.assignmentRepository.FindActiveContractorByWorkID(ctx, work.ID)
		if err != nil {
			return false, "", 0, err
		}
		hasContractor = contractor != nil

		hasPendingContractor, err = s.invitationRepository.HasPendingContractor(ctx, work.ID, workerID)
		if err != nil {
			return false, "", 0, err
		}
	}

	eligible, reason, seatCount := model.RecruitmentEligibility(model.RecruitmentEligibilityInput{
		Work:                       work,
		Worker:                     worker,
		HasAssignment:              hasAssignment,
		HasPendingInvite:           hasPending,
		HasActiveContractor:        hasContractor,
		HasPendingContractorInvite: hasPendingContractor,
	})
	return eligible, reason, seatCount, nil
}

func (s *WorkService) requireInvitation(ctx context.Context, invitationID string) (*model.WorkInvitation, error) {
	if strings.TrimSpace(invitationID) == "" {
		return nil, fmt.Errorf("invitation_id is required")
	}
	invitation, err := s.invitationRepository.FindByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if invitation == nil {
		return nil, ErrInvitationNotFound
	}
	return invitation, nil
}

func (s *WorkService) expireInvitation(ctx context.Context, invitation *model.WorkInvitation) (*model.WorkInvitation, error) {
	if invitation == nil || !model.InvitationExpired(invitation.Status, invitation.ExpiresAt, time.Now().UTC()) {
		return invitation, nil
	}
	updated, err := s.invitationRepository.UpdateStatus(
		ctx,
		invitation.ID,
		[]string{model.InvitationStatusPending},
		model.InvitationStatusExpired,
		bson.M{"responded_at": time.Now().UTC()},
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		current, err := s.invitationRepository.FindByID(ctx, invitation.ID)
		if err != nil {
			return nil, err
		}
		if current != nil {
			return current, nil
		}
	}
	return updated, nil
}

func (s *WorkService) expireInvitations(
	ctx context.Context,
	invitations []model.WorkInvitation,
) ([]model.WorkInvitation, error) {
	for i := range invitations {
		updated, err := s.expireInvitation(ctx, &invitations[i])
		if err != nil {
			return nil, err
		}
		if updated != nil {
			invitations[i] = *updated
		}
	}
	return invitations, nil
}

func (s *WorkService) decorateInvitations(
	ctx context.Context,
	invitations []model.WorkInvitation,
) ([]model.WorkInvitation, error) {
	if len(invitations) == 0 {
		return invitations, nil
	}
	workIDs := make([]string, 0, len(invitations))
	ownerIDs := make([]string, 0, len(invitations))
	seenWork := map[string]struct{}{}
	seenOwner := map[string]struct{}{}
	for _, invitation := range invitations {
		if _, ok := seenWork[invitation.WorkID]; !ok {
			seenWork[invitation.WorkID] = struct{}{}
			workIDs = append(workIDs, invitation.WorkID)
		}
		if _, ok := seenOwner[invitation.OwnerID]; !ok {
			seenOwner[invitation.OwnerID] = struct{}{}
			ownerIDs = append(ownerIDs, invitation.OwnerID)
		}
		if _, ok := seenOwner[invitation.WorkerID]; !ok {
			seenOwner[invitation.WorkerID] = struct{}{}
			ownerIDs = append(ownerIDs, invitation.WorkerID)
		}
	}
	works, err := s.workRepository.FindByIDs(ctx, workIDs)
	if err != nil {
		return nil, err
	}
	names, err := s.workOwnerRepository.FindDisplayNames(ctx, ownerIDs)
	if err != nil {
		log.Printf("invitation display names failed: error=%v", err)
		names = map[string]string{}
	}
	for i := range invitations {
		if work, ok := works[invitations[i].WorkID]; ok {
			invitations[i].Work = model.BuildWorkResponse(&work)
		}
		invitations[i].OwnerName = names[invitations[i].OwnerID]
		invitations[i].WorkerName = names[invitations[i].WorkerID]
	}
	return invitations, nil
}
