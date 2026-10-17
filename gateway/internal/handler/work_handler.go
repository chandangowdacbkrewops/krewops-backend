package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/response"
	workv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/work/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type WorkHandler struct {
	workClient workv1.WorkServiceClient
}

type CreateWorkRequest struct {
	Status *string `json:"status"`

	Title          *string `json:"title"`
	WorkTypeID     *string `json:"work_type_id"`
	WorkCategoryID *string `json:"work_category_id"`
	WorkSegmentID  *string `json:"work_segment_id"`
	Description    *string `json:"description"`

	Address   *string  `json:"address"`
	City      *string  `json:"city"`
	State     *string  `json:"state"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`

	WorkersNeeded     *int32   `json:"workers_needed"`
	ExperienceLevel   *string  `json:"experience_level"`
	Skills            []string `json:"skills"`
	ToolsProvided     *bool    `json:"tools_provided"`
	MaterialsProvided *bool    `json:"materials_provided"`

	StartDate     *string `json:"start_date"`
	DurationValue *int32  `json:"duration_value"`
	DurationUnit  *string `json:"duration_unit"`
	ShiftTiming   *string `json:"shift_timing"`

	RateBasis             *string  `json:"rate_basis"`
	RateUnit              *string  `json:"rate_unit"`
	BudgetRate            *float64 `json:"budget_rate"`
	PaymentNotes          *string  `json:"payment_notes"`
	AccommodationProvided *bool    `json:"accommodation_provided"`
	MealsProvided         *bool    `json:"meals_provided"`
	HiringMode            *string  `json:"hiring_mode"`

	Attributes map[string]string `json:"attributes"`
}

func NewWorkHandler(
	workClient workv1.WorkServiceClient,
) *WorkHandler {

	return &WorkHandler{
		workClient: workClient,
	}
}

// CreateWork godoc
//
//	@Summary	Create a work posting
//	@Tags		Work
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		request	body		CreateWorkRequest	true	"Work posting details"
//	@Success	201		{object}	response.SuccessResponse{data=workv1.CreateWorkResponse}
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	401		{object}	response.ErrorResponse
//	@Failure	403		{object}	response.ErrorResponse	"User has not completed onboarding"
//	@Failure	500		{object}	response.ErrorResponse
//	@Router		/work [post]
func (h *WorkHandler) CreateWork(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	log.Printf("create work started: request_id=%s", requestID)

	var request CreateWorkRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("create work rejected: request_id=%s stage=bind error=%v", requestID, err)
		response.BadRequest(c, "invalid request", nil)
		return
	}

	userID, exists := c.Get("user_id")
	userIDString, valid := userID.(string)

	if !exists || !valid || userIDString == "" {
		log.Printf("create work rejected: request_id=%s stage=auth reason=missing_user_id", requestID)
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	log.Printf("create work request accepted: request_id=%s user_id=%s status=%s work_type_id=%s", requestID, userIDString, stringValue(request.Status), stringValue(request.WorkTypeID))

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		3*time.Second,
	)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

	createResp, err := h.workClient.CreateWork(
		ctx,
		&workv1.CreateWorkRequest{
			UserId:                userIDString,
			Status:                request.Status,
			Title:                 request.Title,
			WorkTypeId:            request.WorkTypeID,
			WorkCategoryId:        request.WorkCategoryID,
			WorkSegmentId:         request.WorkSegmentID,
			Description:           request.Description,
			Address:               request.Address,
			City:                  request.City,
			State:                 request.State,
			Latitude:              request.Latitude,
			Longitude:             request.Longitude,
			WorkersNeeded:         request.WorkersNeeded,
			ExperienceLevel:       request.ExperienceLevel,
			Skills:                request.Skills,
			ToolsProvided:         request.ToolsProvided,
			MaterialsProvided:     request.MaterialsProvided,
			StartDate:             request.StartDate,
			DurationValue:         request.DurationValue,
			DurationUnit:          request.DurationUnit,
			ShiftTiming:           request.ShiftTiming,
			RateBasis:             request.RateBasis,
			RateUnit:              request.RateUnit,
			BudgetRate:            request.BudgetRate,
			PaymentNotes:          request.PaymentNotes,
			AccommodationProvided: request.AccommodationProvided,
			MealsProvided:         request.MealsProvided,
			Attributes:            request.Attributes,
			HiringMode:            request.HiringMode,
		},
	)

	if err != nil {
		log.Printf("create work failed: request_id=%s user_id=%s grpc_code=%s error=%v", requestID, userIDString, status.Code(err), err)
		switch status.Code(err) {
		case codes.PermissionDenied:
			response.Forbidden(c, "user has not completed onboarding", nil)
		case codes.InvalidArgument:
			response.BadRequest(c, err.Error(), nil)
		default:
			response.Internal(c, "failed to create work posting", nil)
		}
		return
	}
	log.Printf("create work completed: request_id=%s user_id=%s work_id=%s", requestID, userIDString, createResp.GetWork().GetId())

	response.Success(
		c,
		http.StatusCreated,
		createResp,
	)
}

// ListMyWorks godoc
//
//	@Summary	List the current user's work postings
//	@Tags		Work
//	@Produce	json
//	@Security	BearerAuth
//	@Param		status	query		string	false	"Filter by work posting status"
//	@Success	200		{object}	response.SuccessResponse{data=workv1.ListMyWorksResponse}
//	@Failure	401		{object}	response.ErrorResponse
//	@Failure	500		{object}	response.ErrorResponse
//	@Router		/work [get]
func (h *WorkHandler) ListMyWorks(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

	var workStatus *string
	if raw := c.Query("status"); raw != "" {
		workStatus = &raw
	}

	listResp, err := h.workClient.ListMyWorks(ctx, &workv1.ListMyWorksRequest{
		UserId: userID,
		Status: workStatus,
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to list work postings")
		return
	}

	response.Success(c, http.StatusOK, listResp)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

type ApplyToWorkRequest struct {
	Quotation Quotation `json:"quotation"`
}

type Quotation struct {
	Amount                 float64  `json:"amount"`
	Currency               string   `json:"currency"`
	RateBasis              string   `json:"rate_basis"`
	RateUnit               *string  `json:"rate_unit"`
	EstimatedDurationHours *float64 `json:"estimated_duration_hours"`
	Message                *string  `json:"message"`
}

// ApplyToWork godoc
//
//	@Summary	Apply to a work posting
//	@Tags		Applications
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workId	path		string				true	"Work posting ID"
//	@Param		request	body		ApplyToWorkRequest	true	"Quotation details"
//	@Success	201		{object}	response.SuccessResponse{data=workv1.ApplyToWorkResponse}
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	401		{object}	response.ErrorResponse
//	@Failure	404		{object}	response.ErrorResponse	"Work posting not found"
//	@Failure	409		{object}	response.ErrorResponse	"Already applied"
//	@Failure	500		{object}	response.ErrorResponse
//	@Router		/work/{workId}/applications [post]
func (h *WorkHandler) ApplyToWork(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}

	var request ApplyToWorkRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid request", nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

	applyResp, err := h.workClient.ApplyToWork(ctx, &workv1.ApplyToWorkRequest{
		UserId: userID,
		WorkId: c.Param("workId"),
		Quotation: &workv1.Quotation{
			Amount:                 request.Quotation.Amount,
			Currency:               request.Quotation.Currency,
			RateBasis:              request.Quotation.RateBasis,
			RateUnit:               request.Quotation.RateUnit,
			EstimatedDurationHours: request.Quotation.EstimatedDurationHours,
			Message:                request.Quotation.Message,
		},
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to apply to work")
		return
	}

	response.Success(c, http.StatusCreated, applyResp)
}

// ListWorkApplications godoc
//
//	@Summary	List applications received for a work posting
//	@Tags		Applications
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workId	path		string	true	"Work posting ID"
//	@Param		status	query		string	false	"Filter by application status"
//	@Success	200		{object}	response.SuccessResponse{data=workv1.ListWorkApplicationsResponse}
//	@Failure	401		{object}	response.ErrorResponse
//	@Failure	403		{object}	response.ErrorResponse
//	@Failure	404		{object}	response.ErrorResponse	"Work posting not found"
//	@Failure	500		{object}	response.ErrorResponse
//	@Router		/work/{workId}/applications [get]
func (h *WorkHandler) ListWorkApplications(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

	var status *string
	if raw := c.Query("status"); raw != "" {
		status = &raw
	}

	listResp, err := h.workClient.ListWorkApplications(ctx, &workv1.ListWorkApplicationsRequest{
		UserId: userID,
		WorkId: c.Param("workId"),
		Status: status,
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to list work applications")
		return
	}

	response.Success(c, http.StatusOK, listResp)
}

// ListMyApplications godoc
//
//	@Summary	List the current user's applications
//	@Tags		Applications
//	@Produce	json
//	@Security	BearerAuth
//	@Param		status	query		string	false	"Filter by application status"
//	@Success	200		{object}	response.SuccessResponse{data=workv1.ListMyApplicationsResponse}
//	@Failure	401		{object}	response.ErrorResponse
//	@Failure	500		{object}	response.ErrorResponse
//	@Router		/work/applications [get]
func (h *WorkHandler) ListMyApplications(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

	var applicationStatus *string
	if raw := c.Query("status"); raw != "" {
		applicationStatus = &raw
	}

	listResp, err := h.workClient.ListMyApplications(ctx, &workv1.ListMyApplicationsRequest{
		UserId: userID,
		Status: applicationStatus,
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to list applications")
		return
	}

	response.Success(c, http.StatusOK, listResp)
}

// ShortlistWorkApplication godoc
//
//	@Summary	Shortlist an application (owner action)
//	@Tags		Applications
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workId			path		string	true	"Work posting ID"
//	@Param		applicationId	path		string	true	"Application ID"
//	@Success	200				{object}	response.SuccessResponse{data=workv1.UpdateWorkApplicationResponse}
//	@Failure	401				{object}	response.ErrorResponse
//	@Failure	403				{object}	response.ErrorResponse
//	@Failure	404				{object}	response.ErrorResponse	"Application not found"
//	@Failure	409				{object}	response.ErrorResponse	"Invalid state transition"
//	@Failure	500				{object}	response.ErrorResponse
//	@Router		/work/{workId}/applications/{applicationId}/shortlist [post]
func (h *WorkHandler) ShortlistWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workClient.ShortlistWorkApplication, "failed to shortlist application")
}

// AcceptWorkApplication godoc
//
//	@Summary	Accept an application (owner action)
//	@Tags		Applications
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workId			path		string	true	"Work posting ID"
//	@Param		applicationId	path		string	true	"Application ID"
//	@Success	200				{object}	response.SuccessResponse{data=workv1.UpdateWorkApplicationResponse}
//	@Failure	401				{object}	response.ErrorResponse
//	@Failure	403				{object}	response.ErrorResponse
//	@Failure	404				{object}	response.ErrorResponse	"Application not found"
//	@Failure	409				{object}	response.ErrorResponse	"Invalid state transition"
//	@Failure	500				{object}	response.ErrorResponse
//	@Router		/work/{workId}/applications/{applicationId}/accept [post]
func (h *WorkHandler) AcceptWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workClient.AcceptWorkApplication, "failed to accept application")
}

// RejectWorkApplication godoc
//
//	@Summary	Reject an application (owner action)
//	@Tags		Applications
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workId			path		string	true	"Work posting ID"
//	@Param		applicationId	path		string	true	"Application ID"
//	@Success	200				{object}	response.SuccessResponse{data=workv1.UpdateWorkApplicationResponse}
//	@Failure	401				{object}	response.ErrorResponse
//	@Failure	403				{object}	response.ErrorResponse
//	@Failure	404				{object}	response.ErrorResponse	"Application not found"
//	@Failure	409				{object}	response.ErrorResponse	"Invalid state transition"
//	@Failure	500				{object}	response.ErrorResponse
//	@Router		/work/{workId}/applications/{applicationId}/reject [post]
func (h *WorkHandler) RejectWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workClient.RejectWorkApplication, "failed to reject application")
}

// WithdrawWorkApplication godoc
//
//	@Summary	Withdraw an application (applicant action)
//	@Tags		Applications
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workId			path		string	true	"Work posting ID"
//	@Param		applicationId	path		string	true	"Application ID"
//	@Success	200				{object}	response.SuccessResponse{data=workv1.UpdateWorkApplicationResponse}
//	@Failure	401				{object}	response.ErrorResponse
//	@Failure	403				{object}	response.ErrorResponse
//	@Failure	404				{object}	response.ErrorResponse	"Application not found"
//	@Failure	409				{object}	response.ErrorResponse	"Invalid state transition"
//	@Failure	500				{object}	response.ErrorResponse
//	@Router		/work/{workId}/applications/{applicationId}/withdraw [post]
func (h *WorkHandler) WithdrawWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workClient.WithdrawWorkApplication, "failed to withdraw application")
}

// CancelWorkApplication godoc
//
//	@Summary	Cancel an application (owner or applicant action)
//	@Tags		Applications
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workId			path		string	true	"Work posting ID"
//	@Param		applicationId	path		string	true	"Application ID"
//	@Success	200				{object}	response.SuccessResponse{data=workv1.UpdateWorkApplicationResponse}
//	@Failure	401				{object}	response.ErrorResponse
//	@Failure	403				{object}	response.ErrorResponse
//	@Failure	404				{object}	response.ErrorResponse	"Application not found"
//	@Failure	409				{object}	response.ErrorResponse	"Invalid state transition"
//	@Failure	500				{object}	response.ErrorResponse
//	@Router		/work/{workId}/applications/{applicationId}/cancel [post]
func (h *WorkHandler) CancelWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workClient.CancelWorkApplication, "failed to cancel application")
}

// StartWork godoc
//
//	@Summary	Start assigned work
//	@Tags		Work
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workId	path		string	true	"Work posting ID"
//	@Success	200		{object}	response.SuccessResponse{data=workv1.WorkLifecycleResponse}
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	401		{object}	response.ErrorResponse
//	@Failure	403		{object}	response.ErrorResponse
//	@Failure	404		{object}	response.ErrorResponse
//	@Router		/work/{workId}/start [post]
func (h *WorkHandler) StartWork(c *gin.Context) {
	h.updateWorkStatus(c, h.workClient.StartWork, "failed to start work")
}

// CompleteWork godoc
//
//	@Summary	Mark assigned work completed
//	@Tags		Work
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workId	path		string	true	"Work posting ID"
//	@Success	200		{object}	response.SuccessResponse{data=workv1.WorkLifecycleResponse}
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	401		{object}	response.ErrorResponse
//	@Failure	403		{object}	response.ErrorResponse
//	@Failure	404		{object}	response.ErrorResponse
//	@Router		/work/{workId}/complete [post]
func (h *WorkHandler) CompleteWork(c *gin.Context) {
	h.updateWorkStatus(c, h.workClient.CompleteWork, "failed to complete work")
}

// ConfirmWorkCompletion godoc
//
//	@Summary	Confirm completed work and close it
//	@Tags		Work
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workId	path		string	true	"Work posting ID"
//	@Success	200		{object}	response.SuccessResponse{data=workv1.WorkLifecycleResponse}
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	401		{object}	response.ErrorResponse
//	@Failure	403		{object}	response.ErrorResponse
//	@Failure	404		{object}	response.ErrorResponse
//	@Router		/work/{workId}/confirm-completion [post]
func (h *WorkHandler) ConfirmWorkCompletion(c *gin.Context) {
	h.updateWorkStatus(c, h.workClient.ConfirmWorkCompletion, "failed to confirm work completion")
}

// CancelWork godoc
//
//	@Summary	Cancel a work posting
//	@Tags		Work
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workId	path		string	true	"Work posting ID"
//	@Success	200		{object}	response.SuccessResponse{data=workv1.WorkLifecycleResponse}
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	401		{object}	response.ErrorResponse
//	@Failure	403		{object}	response.ErrorResponse
//	@Failure	404		{object}	response.ErrorResponse
//	@Router		/work/{workId}/cancel [post]
func (h *WorkHandler) CancelWork(c *gin.Context) {
	h.updateWorkStatus(c, h.workClient.CancelWork, "failed to cancel work")
}

func (h *WorkHandler) CloseApplications(c *gin.Context) {
	h.updateWorkStatus(c, h.workClient.CloseApplications, "failed to close applications")
}

func (h *WorkHandler) CompleteJob(c *gin.Context) {
	h.updateWorkStatus(c, h.workClient.CompleteJob, "failed to complete job")
}

func (h *WorkHandler) CloseWork(c *gin.Context) {
	h.updateWorkStatus(c, h.workClient.CloseWork, "failed to close work")
}

func (h *WorkHandler) StartAssignment(c *gin.Context) {
	h.updateAssignment(c, h.workClient.StartAssignment, "failed to start assignment")
}

func (h *WorkHandler) CompleteAssignment(c *gin.Context) {
	h.updateAssignment(c, h.workClient.CompleteAssignment, "failed to complete assignment")
}

func (h *WorkHandler) UpsertDailyLog(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	var req modelDailyLog
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := h.workClient.UpsertDailyLog(ctx, &workv1.UpsertDailyLogRequest{
		UserId:       userID,
		AssignmentId: c.Param("assignmentId"),
		WorkDate:     c.Param("date"),
		DayPortion:   req.DayPortion,
		Note:         req.Note,
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to save daily log")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

type modelDailyLog struct {
	DayPortion string  `json:"day_portion"`
	Note       *string `json:"note"`
}

func (h *WorkHandler) ListAssignmentDailyLogs(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := h.workClient.ListAssignmentDailyLogs(ctx, &workv1.ListAssignmentDailyLogsRequest{
		UserId:       userID,
		AssignmentId: c.Param("assignmentId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to list daily logs")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *WorkHandler) ListWorkDailyLogs(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := h.workClient.ListWorkDailyLogs(ctx, &workv1.ListWorkDailyLogsRequest{
		UserId: userID,
		WorkId: c.Param("workId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to list daily logs")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *WorkHandler) GetWorkSettlement(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := h.workClient.GetWorkSettlement(ctx, &workv1.GetWorkSettlementRequest{
		UserId: userID,
		WorkId: c.Param("workId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to get settlement")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

type recordPaymentBody struct {
	Method    string  `json:"method"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Reference *string `json:"reference"`
}

func (h *WorkHandler) RecordAssignmentPayment(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	var req recordPaymentBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	var amount *float64
	if req.Amount > 0 {
		amount = &req.Amount
	}
	var currency *string
	if req.Currency != "" {
		currency = &req.Currency
	}
	resp, err := h.workClient.RecordAssignmentPayment(ctx, &workv1.RecordAssignmentPaymentRequest{
		UserId:       userID,
		WorkId:       c.Param("workId"),
		AssignmentId: c.Param("assignmentId"),
		Method:       req.Method,
		Amount:       amount,
		Currency:     currency,
		Reference:    req.Reference,
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to record payment")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *WorkHandler) ListAssignmentPayments(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := h.workClient.ListAssignmentPayments(ctx, &workv1.ListAssignmentPaymentsRequest{
		UserId:       userID,
		AssignmentId: c.Param("assignmentId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to list payments")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *WorkHandler) ConfirmAssignmentPayment(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := h.workClient.ConfirmAssignmentPayment(ctx, &workv1.UpdateAssignmentPaymentRequest{
		UserId:       userID,
		AssignmentId: c.Param("assignmentId"),
		PaymentId:    c.Param("paymentId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to confirm payment")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

type disputeBody struct {
	Message *string `json:"message"`
}

func (h *WorkHandler) DisputeAssignmentPayment(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	var req disputeBody
	_ = c.ShouldBindJSON(&req)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := h.workClient.DisputeAssignmentPayment(ctx, &workv1.DisputeAssignmentPaymentRequest{
		UserId:       userID,
		AssignmentId: c.Param("assignmentId"),
		PaymentId:    c.Param("paymentId"),
		Message:      req.Message,
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to dispute payment")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *WorkHandler) updateAssignment(
	c *gin.Context,
	fn func(ctx context.Context, in *workv1.AssignmentLifecycleRequest, opts ...grpc.CallOption) (*workv1.AssignmentLifecycleResponse, error),
	fallback string,
) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := fn(ctx, &workv1.AssignmentLifecycleRequest{
		UserId:       userID,
		AssignmentId: c.Param("assignmentId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, fallback)
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *WorkHandler) updateWorkStatus(
	c *gin.Context,
	fn func(ctx context.Context, in *workv1.WorkLifecycleRequest, opts ...grpc.CallOption) (*workv1.WorkLifecycleResponse, error),
	fallback string,
) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

	updateResp, err := fn(ctx, &workv1.WorkLifecycleRequest{
		UserId: userID,
		WorkId: c.Param("workId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, fallback)
		return
	}

	response.Success(c, http.StatusOK, updateResp)
}

func (h *WorkHandler) updateApplication(
	c *gin.Context,
	fn func(ctx context.Context, in *workv1.UpdateWorkApplicationRequest, opts ...grpc.CallOption) (*workv1.UpdateWorkApplicationResponse, error),
	fallback string,
) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

	updateResp, err := fn(ctx, &workv1.UpdateWorkApplicationRequest{
		UserId:        userID,
		WorkId:        c.Param("workId"),
		ApplicationId: c.Param("applicationId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, fallback)
		return
	}

	response.Success(c, http.StatusOK, updateResp)
}

func (h *WorkHandler) GetWorkDayRecords(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := h.workClient.GetWorkDayRecords(ctx, &workv1.GetWorkDayRecordsRequest{
		UserId:   userID,
		WorkId:   c.Param("workId"),
		WorkDate: c.Query("date"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to get work records")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *WorkHandler) SaveWorkDayRecords(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	var req struct {
		WorkDate string `json:"work_date"`
		Records  []struct {
			AssignmentID string `json:"assignment_id"`
			DayType      string `json:"day_type"`
		} `json:"records"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request", nil)
		return
	}
	records := make([]*workv1.SaveWorkDayRecordInput, 0, len(req.Records))
	for _, record := range req.Records {
		records = append(records, &workv1.SaveWorkDayRecordInput{
			AssignmentId: record.AssignmentID,
			DayType:      record.DayType,
		})
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := h.workClient.SaveWorkDayRecords(ctx, &workv1.SaveWorkDayRecordsRequest{
		UserId:   userID,
		WorkId:   c.Param("workId"),
		WorkDate: req.WorkDate,
		Records:  records,
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to save work records")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *WorkHandler) GetWorkerWorkSummary(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := h.workClient.GetWorkerWorkSummary(ctx, &workv1.GetWorkerWorkSummaryRequest{
		UserId: userID,
		WorkId: c.Param("workId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to get worker work summary")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *WorkHandler) GetContractorWorkRecords(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	var workDate *string
	if raw := c.Query("date"); raw != "" {
		workDate = &raw
	}
	resp, err := h.workClient.GetContractorWorkRecords(ctx, &workv1.GetContractorWorkRecordsRequest{
		UserId:   userID,
		WorkId:   c.Param("workId"),
		WorkDate: workDate,
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to get contractor work records")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *WorkHandler) SaveContractorWorkRecord(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	var req struct {
		WorkDate            string `json:"work_date"`
		DeployedWorkerCount int32  `json:"deployed_worker_count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := h.workClient.SaveContractorWorkRecord(ctx, &workv1.SaveContractorWorkRecordRequest{
		UserId:              userID,
		WorkId:              c.Param("workId"),
		WorkDate:            req.WorkDate,
		DeployedWorkerCount: req.DeployedWorkerCount,
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to save contractor work record")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *WorkHandler) GetContractorWorkSummary(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := h.workClient.GetContractorWorkSummary(ctx, &workv1.GetContractorWorkSummaryRequest{
		UserId: userID,
		WorkId: c.Param("workId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to get contractor work summary")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *WorkHandler) GetWorkPayments(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := h.workClient.GetWorkPayments(ctx, &workv1.GetWorkPaymentsRequest{
		UserId: userID,
		WorkId: c.Param("workId"),
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to get work payments")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *WorkHandler) RecordObligationPayment(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)
	userID, ok := gatewayUserID(c)
	if !ok {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	var req struct {
		AmountMinor   int64   `json:"amount_minor"`
		PaymentMethod string  `json:"payment_method"`
		Reference     *string `json:"reference"`
		PaidAt        *string `json:"paid_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request", nil)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)
	resp, err := h.workClient.RecordPayment(ctx, &workv1.RecordPaymentRequest{
		UserId:        userID,
		PaymentId:     c.Param("paymentId"),
		AmountMinor:   req.AmountMinor,
		PaymentMethod: req.PaymentMethod,
		Reference:     req.Reference,
		PaidAt:        req.PaidAt,
	})
	if err != nil {
		writeApplicationGRPCError(c, err, "failed to record payment")
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func gatewayUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	userIDString, valid := userID.(string)
	if !exists || !valid || userIDString == "" {
		return "", false
	}
	return userIDString, true
}

func writeApplicationGRPCError(c *gin.Context, err error, fallback string) {
	message := status.Convert(err).Message()
	switch status.Code(err) {
	case codes.PermissionDenied:
		response.Forbidden(c, message, nil)
	case codes.NotFound:
		response.NotFound(c, message, nil)
	case codes.AlreadyExists:
		response.Conflict(c, message, nil)
	case codes.InvalidArgument, codes.FailedPrecondition:
		response.BadRequest(c, message, nil)
	default:
		response.Internal(c, fallback, nil)
	}
}
