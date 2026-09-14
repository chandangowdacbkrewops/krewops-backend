package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/response"
	workv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/work/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type WorkHandler struct {
	workClient workv1.WorkServiceClient
}

type CreateWorkRequest struct {
	Status *string `json:"status"`

	Title       *string `json:"title"`
	WorkTypeID  *string `json:"work_type_id"`
	Description *string `json:"description"`

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

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
