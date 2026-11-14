package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"

	workv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/work/v1"
	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/response"
)

type CreateInvitationsRequest struct {
	WorkerIDs      []string             `json:"worker_ids"`
	OfferedPayment *InvitationPayment   `json:"offered_payment"`
}

type InvitationPayment struct {
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	RateBasis string  `json:"rate_basis"`
	RateUnit  *string `json:"rate_unit"`
	Notes     *string `json:"notes"`
}

func (h *WorkHandler) ListEligibleRecruitmentWorks(c *gin.Context) {
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := invitationContext(c)
	defer cancel()

	result, err := h.workClient.ListEligibleRecruitmentWorks(ctx, &workv1.ListEligibleRecruitmentWorksRequest{
		UserId:   userID,
		WorkerId: c.Query("worker_id"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to list eligible work")
		return
	}
	response.Success(c, http.StatusOK, result)
}

func (h *WorkHandler) CreateWorkInvitations(c *gin.Context) {
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	var request CreateInvitationsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid request", nil)
		return
	}
	ctx, cancel := invitationContext(c)
	defer cancel()

	result, err := h.workClient.CreateWorkInvitations(ctx, &workv1.CreateWorkInvitationsRequest{
		UserId:         userID,
		WorkId:         c.Param("workId"),
		WorkerIds:      request.WorkerIDs,
		OfferedPayment: toProtoOfferedPayment(request.OfferedPayment),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to create invitations")
		return
	}
	response.Success(c, http.StatusOK, result)
}

func (h *WorkHandler) ListWorkInvitations(c *gin.Context) {
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := invitationContext(c)
	defer cancel()

	result, err := h.workClient.ListWorkInvitations(ctx, &workv1.ListWorkInvitationsRequest{
		UserId: userID,
		WorkId: c.Param("workId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to list invitations")
		return
	}
	response.Success(c, http.StatusOK, result)
}

func (h *WorkHandler) ListMyInvitations(c *gin.Context) {
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := invitationContext(c)
	defer cancel()

	var status *string
	if raw := c.Query("status"); raw != "" {
		status = &raw
	}
	result, err := h.workClient.ListMyInvitations(ctx, &workv1.ListMyInvitationsRequest{
		UserId: userID,
		Status: status,
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to list invitations")
		return
	}
	response.Success(c, http.StatusOK, result)
}

func (h *WorkHandler) AcceptWorkInvitation(c *gin.Context) {
	h.respondToInvitation(c, true)
}

func (h *WorkHandler) DeclineWorkInvitation(c *gin.Context) {
	h.respondToInvitation(c, false)
}

func (h *WorkHandler) respondToInvitation(c *gin.Context, accept bool) {
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := invitationContext(c)
	defer cancel()

	request := &workv1.RespondWorkInvitationRequest{
		UserId:       userID,
		InvitationId: c.Param("invitationId"),
	}
	var (
		result *workv1.RespondWorkInvitationResponse
		err    error
	)
	if accept {
		result, err = h.workClient.AcceptWorkInvitation(ctx, request)
	} else {
		result, err = h.workClient.DeclineWorkInvitation(ctx, request)
	}
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to update invitation")
		return
	}
	response.Success(c, http.StatusOK, result)
}

func (h *WorkHandler) CancelWorkInvitation(c *gin.Context) {
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := invitationContext(c)
	defer cancel()

	result, err := h.workClient.CancelWorkInvitation(ctx, &workv1.CancelWorkInvitationRequest{
		UserId:       userID,
		WorkId:       c.Param("workId"),
		InvitationId: c.Param("invitationId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to cancel invitation")
		return
	}
	response.Success(c, http.StatusOK, result)
}

func invitationContext(c *gin.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", c.GetString(response.RequestIDContextKey))
	return ctx, cancel
}

func toProtoOfferedPayment(payment *InvitationPayment) *workv1.OfferedPayment {
	if payment == nil {
		return nil
	}
	var notes *string
	if payment.Notes != nil {
		notes = payment.Notes
	}
	return &workv1.OfferedPayment{
		Amount:    payment.Amount,
		Currency:  payment.Currency,
		RateBasis: payment.RateBasis,
		RateUnit:  payment.RateUnit,
		Notes:     notes,
	}
}
