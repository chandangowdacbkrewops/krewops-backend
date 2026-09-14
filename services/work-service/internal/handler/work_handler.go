package handler

import (
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
	}
	for _, prefix := range validationPrefixes {
		if strings.HasPrefix(msg, prefix) {
			return true
		}
	}
	return false
}
