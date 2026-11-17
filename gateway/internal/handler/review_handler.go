package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	workv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/work/v1"
	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/response"
)

type CreateReviewRequest struct {
	Rating  int32   `json:"rating" binding:"required"`
	Comment *string `json:"comment"`
}

//	@Summary		Submit a review for a work assignment
//	@Description	Lets the work owner or the assigned worker/contractor leave a 1-5 rating (with optional comment) about the other party, once the work has been paid out or closed. One review per assignment per reviewer; reviews cannot be edited or deleted.
//	@Tags			reviews
//	@Accept			json
//	@Produce		json
//	@Param			assignmentId	path		string				true	"Assignment ID"
//	@Param			request			body		CreateReviewRequest	true	"Review payload"
//	@Success		201				{object}	response.SuccessResponse{data=workv1.CreateReviewResponse}
//	@Failure		400				{object}	response.ErrorResponse
//	@Failure		401				{object}	response.ErrorResponse
//	@Failure		403				{object}	response.ErrorResponse
//	@Failure		409				{object}	response.ErrorResponse
//	@Security		BearerAuth
//	@Router			/work-assignments/{assignmentId}/reviews [post]
func (h *WorkHandler) CreateReview(c *gin.Context) {
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	var request CreateReviewRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid request", nil)
		return
	}
	ctx, cancel := invitationContext(c)
	defer cancel()

	result, err := h.workClient.CreateReview(ctx, &workv1.CreateReviewRequest{
		UserId:       userID,
		AssignmentId: c.Param("assignmentId"),
		Rating:       request.Rating,
		Comment:      request.Comment,
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to submit review")
		return
	}
	response.Success(c, http.StatusCreated, result)
}

//	@Summary		List reviews for a work posting
//	@Description	Returns all reviews left on a work posting. Visible to the work owner or any party with an assignment on that work.
//	@Tags			reviews
//	@Produce		json
//	@Param			workId	path		string	true	"Work ID"
//	@Success		200		{object}	response.SuccessResponse{data=workv1.ListWorkReviewsResponse}
//	@Failure		401		{object}	response.ErrorResponse
//	@Failure		403		{object}	response.ErrorResponse
//	@Failure		404		{object}	response.ErrorResponse
//	@Security		BearerAuth
//	@Router			/work/{workId}/reviews [get]
func (h *WorkHandler) ListWorkReviews(c *gin.Context) {
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := invitationContext(c)
	defer cancel()

	result, err := h.workClient.ListWorkReviews(ctx, &workv1.ListWorkReviewsRequest{
		UserId: userID,
		WorkId: c.Param("workId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to list reviews")
		return
	}
	response.Success(c, http.StatusOK, result)
}

//	@Summary		List reviews received by a user
//	@Description	Returns the reviews a user has received across all of their work, meant for display on their profile.
//	@Tags			reviews
//	@Produce		json
//	@Param			userId	path		string	true	"User ID"
//	@Success		200		{object}	response.SuccessResponse{data=workv1.ListUserReviewsResponse}
//	@Failure		401		{object}	response.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/{userId}/reviews [get]
func (h *WorkHandler) ListUserReviews(c *gin.Context) {
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := invitationContext(c)
	defer cancel()

	result, err := h.workClient.ListUserReviews(ctx, &workv1.ListUserReviewsRequest{
		UserId:       userID,
		TargetUserId: c.Param("userId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to list reviews")
		return
	}
	response.Success(c, http.StatusOK, result)
}

//	@Summary		Get a user's aggregate rating summary
//	@Description	Returns the average rating and review count for a user, computed from all reviews they have received.
//	@Tags			reviews
//	@Produce		json
//	@Param			userId	path		string	true	"User ID"
//	@Success		200		{object}	response.SuccessResponse{data=workv1.GetUserRatingSummaryResponse}
//	@Failure		401		{object}	response.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/{userId}/rating-summary [get]
func (h *WorkHandler) GetUserRatingSummary(c *gin.Context) {
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := invitationContext(c)
	defer cancel()

	result, err := h.workClient.GetUserRatingSummary(ctx, &workv1.GetUserRatingSummaryRequest{
		UserId:       userID,
		TargetUserId: c.Param("userId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to get rating summary")
		return
	}
	response.Success(c, http.StatusOK, result)
}
