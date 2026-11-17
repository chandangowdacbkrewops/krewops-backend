package grpcserver

import (
	"context"
	"strings"
	"time"

	workv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/work/v1"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
)

func (s *WorkServer) ListEligibleRecruitmentWorks(
	ctx context.Context,
	req *workv1.ListEligibleRecruitmentWorksRequest,
) (*workv1.ListEligibleRecruitmentWorksResponse, error) {
	results, err := s.workService.ListEligibleRecruitmentWorks(ctx, req.GetUserId(), req.GetWorkerId())
	if err != nil {
		return nil, mapApplicationError(err, "unable to list eligible work")
	}
	works := make([]*workv1.EligibleRecruitmentWork, 0, len(results))
	for _, result := range results {
		works = append(works, &workv1.EligibleRecruitmentWork{
			Work:      toProtoWorkPosting(result.Work),
			Eligible:  result.Eligible,
			Reason:    result.Reason,
			SeatCount: int32(result.SeatCount),
		})
	}
	return &workv1.ListEligibleRecruitmentWorksResponse{Works: works}, nil
}

func (s *WorkServer) CreateWorkInvitations(
	ctx context.Context,
	req *workv1.CreateWorkInvitationsRequest,
) (*workv1.CreateWorkInvitationsResponse, error) {
	results, err := s.workService.CreateWorkInvitations(
		ctx,
		req.GetUserId(),
		req.GetWorkId(),
		req.GetWorkerIds(),
		offeredPaymentFromProto(req.GetOfferedPayment()),
	)
	if err != nil {
		return nil, mapApplicationError(err, "unable to create invitations")
	}
	mapped := make([]*workv1.InvitationCreateResult, 0, len(results))
	for _, result := range results {
		mapped = append(mapped, &workv1.InvitationCreateResult{
			WorkerId:   result.WorkerID,
			Outcome:    result.Outcome,
			Reason:     result.Reason,
			Invitation: toProtoInvitation(result.Invitation),
		})
	}
	return &workv1.CreateWorkInvitationsResponse{Results: mapped}, nil
}

func (s *WorkServer) ListWorkInvitations(
	ctx context.Context,
	req *workv1.ListWorkInvitationsRequest,
) (*workv1.ListWorkInvitationsResponse, error) {
	invitations, err := s.workService.ListWorkInvitations(ctx, req.GetUserId(), req.GetWorkId())
	if err != nil {
		return nil, mapApplicationError(err, "unable to list invitations")
	}
	return &workv1.ListWorkInvitationsResponse{Invitations: toProtoInvitations(invitations)}, nil
}

func (s *WorkServer) ListMyInvitations(
	ctx context.Context,
	req *workv1.ListMyInvitationsRequest,
) (*workv1.ListMyInvitationsResponse, error) {
	invitations, err := s.workService.ListMyInvitations(ctx, req.GetUserId(), req.Status)
	if err != nil {
		return nil, mapApplicationError(err, "unable to list invitations")
	}
	return &workv1.ListMyInvitationsResponse{Invitations: toProtoInvitations(invitations)}, nil
}

func (s *WorkServer) AcceptWorkInvitation(
	ctx context.Context,
	req *workv1.RespondWorkInvitationRequest,
) (*workv1.RespondWorkInvitationResponse, error) {
	invitation, err := s.workService.AcceptWorkInvitation(ctx, req.GetUserId(), req.GetInvitationId())
	if err != nil {
		return nil, mapApplicationError(err, "unable to accept invitation")
	}
	return &workv1.RespondWorkInvitationResponse{Invitation: toProtoInvitation(invitation)}, nil
}

func (s *WorkServer) DeclineWorkInvitation(
	ctx context.Context,
	req *workv1.RespondWorkInvitationRequest,
) (*workv1.RespondWorkInvitationResponse, error) {
	invitation, err := s.workService.DeclineWorkInvitation(ctx, req.GetUserId(), req.GetInvitationId())
	if err != nil {
		return nil, mapApplicationError(err, "unable to decline invitation")
	}
	return &workv1.RespondWorkInvitationResponse{Invitation: toProtoInvitation(invitation)}, nil
}

func (s *WorkServer) CancelWorkInvitation(
	ctx context.Context,
	req *workv1.CancelWorkInvitationRequest,
) (*workv1.RespondWorkInvitationResponse, error) {
	invitation, err := s.workService.CancelWorkInvitation(
		ctx,
		req.GetUserId(),
		req.GetWorkId(),
		req.GetInvitationId(),
	)
	if err != nil {
		return nil, mapApplicationError(err, "unable to cancel invitation")
	}
	return &workv1.RespondWorkInvitationResponse{Invitation: toProtoInvitation(invitation)}, nil
}

func offeredPaymentFromProto(payment *workv1.OfferedPayment) *model.OfferedPayment {
	if payment == nil {
		return nil
	}
	if payment.GetAmount() == 0 && strings.TrimSpace(payment.GetRateBasis()) == "" && strings.TrimSpace(payment.GetCurrency()) == "" {
		return nil
	}
	var rateUnit *string
	if strings.TrimSpace(payment.GetRateUnit()) != "" {
		value := payment.GetRateUnit()
		rateUnit = &value
	}
	var notes *string
	if strings.TrimSpace(payment.GetNotes()) != "" {
		value := payment.GetNotes()
		notes = &value
	}
	return &model.OfferedPayment{
		Amount:    payment.GetAmount(),
		Currency:  payment.GetCurrency(),
		RateBasis: payment.GetRateBasis(),
		RateUnit:  rateUnit,
		Notes:     notes,
	}
}

func toProtoInvitations(invitations []model.WorkInvitation) []*workv1.WorkInvitation {
	mapped := make([]*workv1.WorkInvitation, 0, len(invitations))
	for i := range invitations {
		mapped = append(mapped, toProtoInvitation(&invitations[i]))
	}
	return mapped
}

func toProtoInvitation(invitation *model.WorkInvitation) *workv1.WorkInvitation {
	if invitation == nil {
		return nil
	}
	protoInvitation := &workv1.WorkInvitation{
		Id:            invitation.ID,
		WorkId:        invitation.WorkID,
		OwnerId:       invitation.OwnerID,
		WorkerId:      invitation.WorkerID,
		Status:        invitation.Status,
		ApplicantType: invitation.ApplicantType,
		SeatCount:     int32(invitation.SeatCount),
		OfferedPayment: &workv1.OfferedPayment{
			Amount:    invitation.OfferedPayment.Amount,
			Currency:  invitation.OfferedPayment.Currency,
			RateBasis: invitation.OfferedPayment.RateBasis,
			RateUnit:  invitation.OfferedPayment.RateUnit,
			Notes:     invitation.OfferedPayment.Notes,
		},
		CreatedAt:    invitation.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    invitation.UpdatedAt.UTC().Format(time.RFC3339),
		ExpiresAt:    invitation.ExpiresAt.UTC().Format(time.RFC3339),
		RespondedAt:  timePtrString(invitation.RespondedAt),
		AssignmentId: stringPtr(invitation.AssignmentID),
		OwnerName:    stringPtr(invitation.OwnerName),
		WorkerName:   stringPtr(invitation.WorkerName),
		Work:         toProtoWorkPosting(invitation.Work),
	}
	return protoInvitation
}
