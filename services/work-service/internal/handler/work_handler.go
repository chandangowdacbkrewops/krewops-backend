package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/middleware"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/service"
)

type WorkHandler struct {
	workService *service.WorkService
}

func NewWorkHandler(workService *service.WorkService) *WorkHandler {
	return &WorkHandler{workService: workService}
}

func (h *WorkHandler) CreateWork(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req model.CreateWorkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.workService.CreateWork(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrOnboardingNotCompleted) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to create work posting"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result})
}

func (h *WorkHandler) ListMyWorks(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var status *string
	if raw := strings.TrimSpace(c.Query("status")); raw != "" {
		status = &raw
	}

	result, err := h.workService.ListMyWorks(c.Request.Context(), userID, status)
	if err != nil {
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to list work postings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"works": result}})
}

func (h *WorkHandler) ApplyToWork(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req model.ApplyToWorkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.workService.ApplyToWork(c.Request.Context(), userID, c.Param("workId"), req)
	if err != nil {
		writeApplicationError(c, err, "unable to apply to work")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result})
}

func (h *WorkHandler) ListWorkApplications(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var status *string
	if raw := strings.TrimSpace(c.Query("status")); raw != "" {
		status = &raw
	}

	result, err := h.workService.ListWorkApplications(c.Request.Context(), userID, c.Param("workId"), status)
	if err != nil {
		writeApplicationError(c, err, "unable to list work applications")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) ListMyApplications(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var status *string
	if raw := strings.TrimSpace(c.Query("status")); raw != "" {
		status = &raw
	}

	result, err := h.workService.ListMyApplications(c.Request.Context(), userID, status)
	if err != nil {
		writeApplicationError(c, err, "unable to list applications")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"applications": result}})
}

func (h *WorkHandler) ShortlistWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workService.ShortlistWorkApplication, "unable to shortlist application")
}

func (h *WorkHandler) AcceptWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workService.AcceptWorkApplication, "unable to accept application")
}

func (h *WorkHandler) RejectWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workService.RejectWorkApplication, "unable to reject application")
}

func (h *WorkHandler) WithdrawWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workService.WithdrawWorkApplication, "unable to withdraw application")
}

func (h *WorkHandler) CancelWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workService.CancelWorkApplication, "unable to cancel application")
}

func (h *WorkHandler) updateApplication(
	c *gin.Context,
	fn func(ctx context.Context, userID, workID, applicationID string) (*model.WorkApplication, error),
	fallback string,
) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	result, err := fn(c.Request.Context(), userID, c.Param("workId"), c.Param("applicationId"))
	if err != nil {
		writeApplicationError(c, err, fallback)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func writeApplicationError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrWorkerProfileNotCompleted),
		errors.Is(err, service.ErrCannotApplyToOwnWork),
		errors.Is(err, service.ErrNotWorkOwner),
		errors.Is(err, service.ErrNotApplicationWorker):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrWorkNotFound),
		errors.Is(err, service.ErrApplicationNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrDuplicateApplication),
		errors.Is(err, service.ErrApplicationAlreadyAccepted):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrWorkNotPublished),
		errors.Is(err, service.ErrInvalidApplicationStatus),
		isValidationError(err):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": fallback})
	}
}

func (h *WorkHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "work-service",
		"status":  "ok",
	})
}

func isValidationError(err error) bool {
	msg := err.Error()
	validationPrefixes := []string{
		"status",
		"title",
		"description",
		"work_type_id",
		"work_category_id",
		"address",
		"city",
		"state",
		"workers_needed",
		"tools_provided",
		"materials_provided",
		"start_date",
		"duration_value",
		"duration_unit",
		"payment_type",
		"budget_rate",
		"payment_notes",
		"attributes",
		"quotation",
		"work_id",
		"application_id",
	}
	for _, prefix := range validationPrefixes {
		if strings.HasPrefix(msg, prefix) {
			return true
		}
	}
	return false
}
