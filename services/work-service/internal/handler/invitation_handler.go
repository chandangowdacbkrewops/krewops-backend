package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/middleware"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
)

type createInvitationsBody struct {
	WorkerIDs      []string              `json:"worker_ids"`
	OfferedPayment *model.OfferedPayment `json:"offered_payment"`
}

func (h *WorkHandler) ListEligibleRecruitmentWorks(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.workService.ListEligibleRecruitmentWorks(c.Request.Context(), userID, c.Query("worker_id"))
	if err != nil {
		writeApplicationError(c, err, "unable to list eligible work")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"works": result}})
}

func (h *WorkHandler) CreateWorkInvitations(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var body createInvitationsBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	result, err := h.workService.CreateWorkInvitations(
		c.Request.Context(),
		userID,
		c.Param("workId"),
		body.WorkerIDs,
		body.OfferedPayment,
	)
	if err != nil {
		writeApplicationError(c, err, "unable to create invitations")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"results": result}})
}

func (h *WorkHandler) ListWorkInvitations(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.workService.ListWorkInvitations(c.Request.Context(), userID, c.Param("workId"))
	if err != nil {
		writeApplicationError(c, err, "unable to list invitations")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"invitations": result}})
}

func (h *WorkHandler) ListMyInvitations(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var status *string
	if raw := strings.TrimSpace(c.Query("status")); raw != "" {
		status = &raw
	}
	result, err := h.workService.ListMyInvitations(c.Request.Context(), userID, status)
	if err != nil {
		writeApplicationError(c, err, "unable to list invitations")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"invitations": result}})
}

func (h *WorkHandler) AcceptWorkInvitation(c *gin.Context) {
	h.respondInvitation(c, true)
}

func (h *WorkHandler) DeclineWorkInvitation(c *gin.Context) {
	h.respondInvitation(c, false)
}

func (h *WorkHandler) respondInvitation(c *gin.Context, accept bool) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var (
		result *model.WorkInvitation
		err    error
	)
	if accept {
		result, err = h.workService.AcceptWorkInvitation(c.Request.Context(), userID, c.Param("invitationId"))
	} else {
		result, err = h.workService.DeclineWorkInvitation(c.Request.Context(), userID, c.Param("invitationId"))
	}
	if err != nil {
		writeApplicationError(c, err, "unable to update invitation")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) CancelWorkInvitation(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.workService.CancelWorkInvitation(
		c.Request.Context(),
		userID,
		c.Param("workId"),
		c.Param("invitationId"),
	)
	if err != nil {
		writeApplicationError(c, err, "unable to cancel invitation")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}
