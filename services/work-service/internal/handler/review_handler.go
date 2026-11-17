package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/middleware"
)

type createReviewBody struct {
	Rating  int32   `json:"rating"`
	Comment *string `json:"comment"`
}

func (h *WorkHandler) CreateReview(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var body createReviewBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.workService.CreateReview(
		c.Request.Context(),
		userID,
		c.Param("assignmentId"),
		body.Rating,
		body.Comment,
	)
	if err != nil {
		writeApplicationError(c, err, "unable to submit review")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": result})
}

func (h *WorkHandler) ListWorkReviews(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	result, err := h.workService.ListWorkReviews(c.Request.Context(), userID, c.Param("workId"))
	if err != nil {
		writeApplicationError(c, err, "unable to list reviews")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"reviews": result}})
}

func (h *WorkHandler) ListUserReviews(c *gin.Context) {
	_, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	result, err := h.workService.ListUserReviews(c.Request.Context(), c.Param("userId"))
	if err != nil {
		writeApplicationError(c, err, "unable to list reviews")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"reviews": result}})
}

func (h *WorkHandler) GetUserRatingSummary(c *gin.Context) {
	_, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	result, err := h.workService.GetUserRatingSummary(c.Request.Context(), c.Param("userId"))
	if err != nil {
		writeApplicationError(c, err, "unable to get rating summary")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}
