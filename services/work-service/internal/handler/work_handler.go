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

func (h *WorkHandler) StartWork(c *gin.Context) {
	h.updateWorkStatus(c, h.workService.StartWork, "unable to start work")
}

func (h *WorkHandler) CompleteWork(c *gin.Context) {
	h.updateWorkStatus(c, h.workService.CompleteWork, "unable to complete work")
}

func (h *WorkHandler) ConfirmWorkCompletion(c *gin.Context) {
	h.updateWorkStatus(c, h.workService.ConfirmWorkCompletion, "unable to confirm work completion")
}

func (h *WorkHandler) CancelWork(c *gin.Context) {
	h.updateWorkStatus(c, h.workService.CancelWork, "unable to cancel work")
}

func (h *WorkHandler) CloseApplications(c *gin.Context) {
	h.updateWorkStatus(c, h.workService.CloseApplications, "unable to close applications")
}

func (h *WorkHandler) CompleteJob(c *gin.Context) {
	h.updateWorkStatus(c, h.workService.CompleteJob, "unable to complete job")
}

func (h *WorkHandler) CloseWork(c *gin.Context) {
	h.updateWorkStatus(c, h.workService.CloseWork, "unable to close work")
}

func (h *WorkHandler) StartAssignment(c *gin.Context) {
	h.updateAssignment(c, h.workService.StartAssignment, "unable to start assignment")
}

func (h *WorkHandler) CompleteAssignment(c *gin.Context) {
	h.updateAssignment(c, h.workService.CompleteAssignment, "unable to complete assignment")
}

func (h *WorkHandler) UpsertDailyLog(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req model.UpsertDailyLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	result, err := h.workService.UpsertDailyLog(c.Request.Context(), userID, c.Param("assignmentId"), c.Param("date"), req)
	if err != nil {
		writeApplicationError(c, err, "unable to save daily log")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) ListAssignmentDailyLogs(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.workService.ListAssignmentDailyLogs(c.Request.Context(), userID, c.Param("assignmentId"))
	if err != nil {
		writeApplicationError(c, err, "unable to list daily logs")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) ListWorkDailyLogs(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.workService.ListWorkDailyLogs(c.Request.Context(), userID, c.Param("workId"))
	if err != nil {
		writeApplicationError(c, err, "unable to list daily logs")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) GetWorkSettlement(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.workService.GetWorkSettlement(c.Request.Context(), userID, c.Param("workId"))
	if err != nil {
		writeApplicationError(c, err, "unable to get settlement")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) RecordAssignmentPayment(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req model.RecordPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	result, err := h.workService.RecordAssignmentPayment(c.Request.Context(), userID, c.Param("workId"), c.Param("assignmentId"), req)
	if err != nil {
		writeApplicationError(c, err, "unable to record payment")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) ListAssignmentPayments(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.workService.ListAssignmentPayments(c.Request.Context(), userID, c.Param("assignmentId"))
	if err != nil {
		writeApplicationError(c, err, "unable to list payments")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) ConfirmAssignmentPayment(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.workService.ConfirmAssignmentPayment(c.Request.Context(), userID, c.Param("assignmentId"), c.Param("paymentId"))
	if err != nil {
		writeApplicationError(c, err, "unable to confirm payment")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) DisputeAssignmentPayment(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req model.DisputePaymentRequest
	_ = c.ShouldBindJSON(&req)
	result, err := h.workService.DisputeAssignmentPayment(c.Request.Context(), userID, c.Param("assignmentId"), c.Param("paymentId"), req.Message)
	if err != nil {
		writeApplicationError(c, err, "unable to dispute payment")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) updateAssignment(
	c *gin.Context,
	fn func(ctx context.Context, userID, assignmentID string) (*model.WorkAssignment, error),
	fallback string,
) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := fn(c.Request.Context(), userID, c.Param("assignmentId"))
	if err != nil {
		writeApplicationError(c, err, fallback)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) updateWorkStatus(
	c *gin.Context,
	fn func(ctx context.Context, userID, workID string) (*model.WorkResponse, error),
	fallback string,
) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	result, err := fn(c.Request.Context(), userID, c.Param("workId"))
	if err != nil {
		writeApplicationError(c, err, fallback)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
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
		errors.Is(err, service.ErrNotApplicationWorker),
		errors.Is(err, service.ErrNotAssignedWorker),
		errors.Is(err, service.ErrContractorRequired):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrWorkNotFound),
		errors.Is(err, service.ErrApplicationNotFound),
		errors.Is(err, service.ErrAssignmentNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrDuplicateApplication),
		errors.Is(err, service.ErrApplicationAlreadyAccepted),
		errors.Is(err, service.ErrNoWorkerSeats):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrWorkNotPublished),
		errors.Is(err, service.ErrInvalidApplicationStatus),
		errors.Is(err, service.ErrInvalidWorkStatus),
		errors.Is(err, service.ErrWorkNotReadyToStart),
		errors.Is(err, service.ErrApplicationsClosed),
		errors.Is(err, service.ErrInvalidAssignmentStatus),
		errors.Is(err, service.ErrLogsFrozen),
		errors.Is(err, service.ErrAssignmentsNotCompleted),
		errors.Is(err, service.ErrPaymentsIncomplete),
		errors.Is(err, service.ErrInvalidPaymentMethod),
		errors.Is(err, service.ErrNoActiveAssignment),
		errors.Is(err, service.ErrContractorAssignmentRequired),
		errors.Is(err, service.ErrContractorAlreadyAssigned),
		errors.Is(err, service.ErrHiringModeMismatch),
		errors.Is(err, service.ErrPaymentAdjustmentRequired),
		errors.Is(err, service.ErrOverpayment),
		errors.Is(err, service.ErrIndividualWorkersNotAllowed),
		isValidationError(err):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": fallback})
	}
}

func (h *WorkHandler) GetWorkDayRecords(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.workService.GetWorkDayRecords(c.Request.Context(), userID, c.Param("workId"), c.Query("date"))
	if err != nil {
		writeApplicationError(c, err, "unable to get work records")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) SaveWorkDayRecords(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req model.SaveWorkDayRecordsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	result, err := h.workService.SaveWorkDayRecords(c.Request.Context(), userID, c.Param("workId"), req)
	if err != nil {
		writeApplicationError(c, err, "unable to save work records")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) GetWorkerWorkSummary(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.workService.GetWorkerWorkSummary(c.Request.Context(), userID, c.Param("workId"))
	if err != nil {
		writeApplicationError(c, err, "unable to get worker work summary")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) GetContractorWorkRecords(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.workService.GetContractorWorkRecords(c.Request.Context(), userID, c.Param("workId"), c.Query("date"))
	if err != nil {
		writeApplicationError(c, err, "unable to get contractor work records")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) SaveContractorWorkRecord(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req model.SaveContractorWorkRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	result, err := h.workService.SaveContractorWorkRecord(c.Request.Context(), userID, c.Param("workId"), req)
	if err != nil {
		writeApplicationError(c, err, "unable to save contractor work record")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) GetContractorWorkSummary(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.workService.GetContractorWorkSummary(c.Request.Context(), userID, c.Param("workId"))
	if err != nil {
		writeApplicationError(c, err, "unable to get contractor work summary")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) GetWorkPayments(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.workService.GetWorkPayments(c.Request.Context(), userID, c.Param("workId"))
	if err != nil {
		writeApplicationError(c, err, "unable to get work payments")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *WorkHandler) RecordObligationPayment(c *gin.Context) {
	userID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req model.RecordObligationPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	result, err := h.workService.RecordObligationPayment(c.Request.Context(), userID, c.Param("paymentId"), req)
	if err != nil {
		writeApplicationError(c, err, "unable to record payment")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
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
		"work_segment_id",
		"address",
		"city",
		"state",
		"workers_needed",
		"tools_provided",
		"materials_provided",
		"start_date",
		"duration_value",
		"duration_unit",
		"rate_basis",
		"rate_unit",
		"budget_rate",
		"payment_notes",
		"attributes",
		"quotation",
		"work_id",
		"application_id",
		"day_portion",
		"day_type",
		"work_date",
		"amount",
		"amount_minor",
		"assignment_id",
		"deployed_worker_count",
		"paid_at",
		"hiring_mode",
		"payment_id",
	}
	for _, prefix := range validationPrefixes {
		if strings.HasPrefix(msg, prefix) {
			return true
		}
	}
	return false
}
