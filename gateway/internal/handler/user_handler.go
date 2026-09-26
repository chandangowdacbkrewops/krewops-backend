package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/response"
	userv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserHandler struct {
	userClient userv1.UserServiceClient
}

type CreateProfileRequest struct {
	FirstName  string `json:"first_name" binding:"required"`
	LastName   string `json:"last_name" binding:"required"`
	UserType   string `json:"user_type" binding:"required"`
	Country    string `json:"country" binding:"required"`
	State      string `json:"state" binding:"required"`
	City       string `json:"city" binding:"required"`
	PostalCode string `json:"postal_code" binding:"required"`
}

type UpdateProfileRequest struct {
	FirstName  string `json:"first_name" binding:"required"`
	LastName   string `json:"last_name" binding:"required"`
	Country    string `json:"country" binding:"required"`
	State      string `json:"state" binding:"required"`
	City       string `json:"city" binding:"required"`
	PostalCode string `json:"postal_code" binding:"required"`
	UserType   string `json:"user_type" binding:"required"`
}

type CreateWorkerProfileRequest struct {
	WorkerType         string   `json:"worker_type" binding:"required"`
	CrewName           *string  `json:"crew_name"`
	CrewSize           int32    `json:"crew_size"`
	ExperienceYears    *float64 `json:"experience_years"`
	ExpectedRate       *float64 `json:"expected_rate"`
	RateType           *string  `json:"rate_type"`
	AvailabilityStatus *string  `json:"availability_status"`
	Bio                *string  `json:"bio"`
	WorkCategoryID     string   `json:"work_category_id" binding:"required"`
}

func NewUserHandler(
	userClient userv1.UserServiceClient,
) *UserHandler {

	return &UserHandler{
		userClient: userClient,
	}
}

func (h *UserHandler) ListWorkTypes(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	workTypesResp, err := h.userClient.ListWorkTypes(
		ctx,
		&userv1.ListWorkTypesRequest{},
	)
	if err != nil {
		response.Internal(c, "failed to fetch work types", nil)
		return
	}

	response.Success(c, http.StatusOK, workTypesResp)
}

func (h *UserHandler) ListWorkCategories(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	workCategoriesResp, err := h.userClient.ListWorkCategories(
		ctx,
		&userv1.ListWorkCategoriesRequest{},
	)
	if err != nil {
		response.Internal(c, "failed to fetch work categories", nil)
		return
	}

	response.Success(c, http.StatusOK, workCategoriesResp)
}

func (h *UserHandler) ListWorkTypesByCategory(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	categoryID := c.Param("categoryId")

	workTypesResp, err := h.userClient.ListWorkTypesByCategory(
		ctx,
		&userv1.ListWorkTypesByCategoryRequest{CategoryId: categoryID},
	)
	if err != nil {
		response.Internal(c, "failed to fetch work types", nil)
		return
	}

	response.Success(c, http.StatusOK, workTypesResp)
}

func (h *UserHandler) ListWorkTypeFields(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	workTypeID := c.Param("workTypeId")

	fieldsResp, err := h.userClient.ListWorkTypeFields(
		ctx,
		&userv1.ListWorkTypeFieldsRequest{WorkTypeId: workTypeID},
	)
	if err != nil {
		response.Internal(c, "failed to fetch work type fields", nil)
		return
	}

	response.Success(c, http.StatusOK, fieldsResp)
}

func (h *UserHandler) ListWorkTypePaymentTypes(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	workTypeID := c.Param("workTypeId")

	paymentTypesResp, err := h.userClient.ListWorkTypePaymentTypes(
		ctx,
		&userv1.ListWorkTypePaymentTypesRequest{WorkTypeId: workTypeID},
	)
	if err != nil {
		response.Internal(c, "failed to fetch payment types", nil)
		return
	}

	response.Success(c, http.StatusOK, paymentTypesResp)
}

func (h *UserHandler) ListWorkCategoryPaymentTypes(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	categoryID := c.Param("categoryId")

	paymentTypesResp, err := h.userClient.ListWorkCategoryPaymentTypes(
		ctx,
		&userv1.ListWorkCategoryPaymentTypesRequest{CategoryId: categoryID},
	)
	if err != nil {
		response.Internal(c, "failed to fetch payment types", nil)
		return
	}

	response.Success(c, http.StatusOK, paymentTypesResp)
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	userIDString, valid := userID.(string)

	if !exists || !valid || userIDString == "" {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		3*time.Second,
	)
	defer cancel()

	profileResp, err := h.userClient.GetProfile(
		ctx,
		&userv1.GetProfileRequest{
			UserId: userIDString,
		},
	)

	if err != nil {
		switch status.Code(err) {
		case codes.NotFound:
			response.NotFound(c, "profile not found", nil)
		default:
			response.Internal(c, "failed to fetch profile", nil)
		}
		return
	}

	if profileResp == nil || profileResp.Profile == nil {
		response.NotFound(c, "profile not found", nil)
		return
	}

	response.Success(
		c,
		http.StatusOK,
		profileResp,
	)
}

func (h *UserHandler) CreateProfile(
	c *gin.Context,
) {
	requestID := c.GetString(response.RequestIDContextKey)
	log.Printf("create profile started: request_id=%s", requestID)

	var request CreateProfileRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("create profile rejected: request_id=%s stage=bind error=%v", requestID, err)
		response.BadRequest(c, "invalid request", nil)
		return
	}

	userID, exists := c.Get("user_id")
	userIDString, valid := userID.(string)

	if !exists || !valid || userIDString == "" {
		log.Printf("create profile rejected: request_id=%s stage=auth reason=missing_user_id", requestID)
		response.Unauthorized(c, "unauthorized", nil)
		return
	}
	log.Printf("create profile request accepted: request_id=%s user_id=%s user_type=%s", requestID, userIDString, request.UserType)

	ctx, cancel :=
		context.WithTimeout(
			c.Request.Context(),
			3*time.Second,
		)

	defer cancel()

	createResp, err :=
		h.userClient.CreateProfile(
			ctx,
			&userv1.CreateProfileRequest{
				UserId:     userIDString,
				FirstName:  request.FirstName,
				LastName:   request.LastName,
				UserType:   request.UserType,
				Country:    request.Country,
				State:      request.State,
				City:       request.City,
				PostalCode: request.PostalCode,
			},
		)

	if err != nil {
		log.Printf("create profile failed: request_id=%s user_id=%s grpc_code=%s error=%v", requestID, userIDString, status.Code(err), err)
		switch status.Code(err) {
		case codes.AlreadyExists:
			response.Conflict(c, "profile already exists", nil)
		case codes.InvalidArgument:
			response.BadRequest(c, "invalid profile data", nil)
		default:
			response.Internal(c, "failed to save profile", nil)
		}
		return
	}
	log.Printf("create profile completed: request_id=%s user_id=%s profile_id=%s", requestID, userIDString, createResp.GetProfile().GetId())

	response.Success(
		c,
		http.StatusCreated,
		createResp,
	)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var request UpdateProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid request", nil)
		return
	}

	userID, exists := c.Get("user_id")
	userIDString, valid := userID.(string)
	if !exists || !valid || userIDString == "" {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	updateResp, err := h.userClient.UpdateProfile(ctx, &userv1.UpdateProfileRequest{
		UserId:     userIDString,
		FirstName:  request.FirstName,
		LastName:   request.LastName,
		Country:    request.Country,
		State:      request.State,
		City:       request.City,
		PostalCode: request.PostalCode,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound:
			response.NotFound(c, "profile not found", nil)
		case codes.InvalidArgument:
			response.BadRequest(c, "invalid profile data", nil)
		default:
			response.Internal(c, "failed to update profile", nil)
		}
		return
	}

	response.Success(c, http.StatusOK, updateResp)
}

func (h *UserHandler) CreateWorkerProfile(c *gin.Context) {
	var request CreateWorkerProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "invalid worker profile data", nil)
		return
	}

	userID, exists := c.Get("user_id")
	userIDString, valid := userID.(string)
	if !exists || !valid || userIDString == "" {
		response.Unauthorized(c, "unauthorized", nil)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	createResp, err := h.userClient.CreateWorkerProfile(
		ctx,
		&userv1.CreateWorkerProfileRequest{
			UserId:             userIDString,
			WorkerType:         request.WorkerType,
			CrewName:           request.CrewName,
			CrewSize:           request.CrewSize,
			ExperienceYears:    request.ExperienceYears,
			ExpectedRate:       request.ExpectedRate,
			RateType:           request.RateType,
			AvailabilityStatus: request.AvailabilityStatus,
			Bio:                request.Bio,
			WorkCategoryId:     request.WorkCategoryID,
		},
	)
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			response.BadRequest(c, "invalid worker profile data", nil)
		case codes.AlreadyExists:
			response.Conflict(c, "worker profile already exists", nil)
		default:
			response.Internal(c, "failed to save worker profile", nil)
		}
		return
	}

	response.Success(c, http.StatusCreated, createResp)
}
