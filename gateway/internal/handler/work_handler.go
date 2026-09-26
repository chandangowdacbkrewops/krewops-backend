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

	PaymentType           *string  `json:"payment_type"`
	BudgetRate            *float64 `json:"budget_rate"`
	PaymentNotes          *string  `json:"payment_notes"`
	AccommodationProvided *bool    `json:"accommodation_provided"`
	MealsProvided         *bool    `json:"meals_provided"`

	Attributes map[string]string `json:"attributes"`
}

func NewWorkHandler(
	workClient workv1.WorkServiceClient,
) *WorkHandler {

	return &WorkHandler{
		workClient: workClient,
	}
}

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
			PaymentType:           request.PaymentType,
			BudgetRate:            request.BudgetRate,
			PaymentNotes:          request.PaymentNotes,
			AccommodationProvided: request.AccommodationProvided,
			MealsProvided:         request.MealsProvided,
			Attributes:            request.Attributes,
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
	PriceType              string   `json:"price_type"`
	EstimatedDurationHours *float64 `json:"estimated_duration_hours"`
	Message                *string  `json:"message"`
}

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
			PriceType:              request.Quotation.PriceType,
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

func (h *WorkHandler) ShortlistWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workClient.ShortlistWorkApplication, "failed to shortlist application")
}

func (h *WorkHandler) AcceptWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workClient.AcceptWorkApplication, "failed to accept application")
}

func (h *WorkHandler) RejectWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workClient.RejectWorkApplication, "failed to reject application")
}

func (h *WorkHandler) WithdrawWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workClient.WithdrawWorkApplication, "failed to withdraw application")
}

func (h *WorkHandler) CancelWorkApplication(c *gin.Context) {
	h.updateApplication(c, h.workClient.CancelWorkApplication, "failed to cancel application")
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
