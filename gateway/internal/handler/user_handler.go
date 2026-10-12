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
	FirstName  string  `json:"first_name" binding:"required"`
	LastName   string  `json:"last_name" binding:"required"`
	Gender     *string `json:"gender"`
	UserType   string  `json:"user_type" binding:"required"`
	Country    string  `json:"country" binding:"required"`
	State      string  `json:"state" binding:"required"`
	City       string  `json:"city" binding:"required"`
	PostalCode string  `json:"postal_code" binding:"required"`
}

type UpdateProfileRequest struct {
	FirstName  string  `json:"first_name" binding:"required"`
	LastName   string  `json:"last_name" binding:"required"`
	Gender     *string `json:"gender"`
	Country    string  `json:"country" binding:"required"`
	State      string  `json:"state" binding:"required"`
	City       string  `json:"city" binding:"required"`
	PostalCode string  `json:"postal_code" binding:"required"`
	UserType   string  `json:"user_type" binding:"required"`
}

type WorkerTypeSelection struct {
	WorkTypeID string   `json:"work_type_id"`
	SkillIDs   []string `json:"skill_ids"`
}

type WorkerSegmentSelection struct {
	WorkSegmentID string                `json:"work_segment_id"`
	WorkTypes     []WorkerTypeSelection `json:"work_types"`
}

type WorkerCategorySelection struct {
	WorkCategoryID string                   `json:"work_category_id"`
	Segments       []WorkerSegmentSelection `json:"segments"`
}

type CreateWorkerProfileRequest struct {
	WorkerType         string                    `json:"worker_type" binding:"required"`
	CrewName           *string                   `json:"crew_name"`
	CrewSize           int32                     `json:"crew_size"`
	ExperienceYears    *float64                  `json:"experience_years"`
	ExpectedRate       *float64                  `json:"expected_rate"`
	ExpectedRateMin    *float64                  `json:"expected_rate_min"`
	ExpectedRateMax    *float64                  `json:"expected_rate_max"`
	RateBasis          *string                   `json:"rate_basis"`
	RateUnit           *string                   `json:"rate_unit"`
	AvailabilityStatus *string                   `json:"availability_status"`
	Bio                *string                   `json:"bio"`
	Selections         []WorkerCategorySelection `json:"selections"`
}

type UpdateWorkerProfileRequest struct {
	Selections []WorkerCategorySelection `json:"selections"`
	RateBasis  *string                   `json:"rate_basis"`
	RateUnit   *string                   `json:"rate_unit"`
}

func NewUserHandler(
	userClient userv1.UserServiceClient,
) *UserHandler {

	return &UserHandler{
		userClient: userClient,
	}
}

// ListWorkTypes godoc
//
//	@Summary	List all work types
//	@Tags		WorkTypes
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	response.SuccessResponse{data=userv1.ListWorkTypesResponse}
//	@Failure	401	{object}	response.ErrorResponse
//	@Failure	500	{object}	response.ErrorResponse
//	@Router		/work-types [get]
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

// ListWorkCategories godoc
//
//	@Summary	List all work categories (sectors)
//	@Tags		WorkTypes
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	response.SuccessResponse{data=userv1.ListWorkCategoriesResponse}
//	@Failure	401	{object}	response.ErrorResponse
//	@Failure	500	{object}	response.ErrorResponse
//	@Router		/work-categories [get]
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

// ListWorkTypesByCategory godoc
//
//	@Summary	List work types under a category
//	@Tags		WorkTypes
//	@Produce	json
//	@Security	BearerAuth
//	@Param		categoryId	path		string	true	"Work category ID"
//	@Success	200			{object}	response.SuccessResponse{data=userv1.ListWorkTypesResponse}
//	@Failure	401			{object}	response.ErrorResponse
//	@Failure	500			{object}	response.ErrorResponse
//	@Router		/work-categories/{categoryId}/work-types [get]
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

// ListWorkSegmentsByCategory godoc
//
//	@Summary	List work segments (groups) under a category
//	@Tags		WorkTypes
//	@Produce	json
//	@Security	BearerAuth
//	@Param		categoryId	path		string	true	"Work category ID"
//	@Success	200			{object}	response.SuccessResponse{data=userv1.ListWorkSegmentsResponse}
//	@Failure	401			{object}	response.ErrorResponse
//	@Failure	500			{object}	response.ErrorResponse
//	@Router		/work-categories/{categoryId}/segments [get]
func (h *UserHandler) ListWorkSegmentsByCategory(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	segmentsResp, err := h.userClient.ListWorkSegmentsByCategory(
		ctx,
		&userv1.ListWorkSegmentsByCategoryRequest{CategoryId: c.Param("categoryId")},
	)
	if err != nil {
		response.Internal(c, "failed to fetch work segments", nil)
		return
	}

	response.Success(c, http.StatusOK, segmentsResp)
}

// ListWorkTypesBySegment godoc
//
//	@Summary	List work types under a segment (group)
//	@Tags		WorkTypes
//	@Produce	json
//	@Security	BearerAuth
//	@Param		segmentId	path		string	true	"Work segment ID"
//	@Success	200			{object}	response.SuccessResponse{data=userv1.ListWorkTypesResponse}
//	@Failure	401			{object}	response.ErrorResponse
//	@Failure	500			{object}	response.ErrorResponse
//	@Router		/work-segments/{segmentId}/work-types [get]
func (h *UserHandler) ListWorkTypesBySegment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	workTypesResp, err := h.userClient.ListWorkTypesBySegment(
		ctx,
		&userv1.ListWorkTypesBySegmentRequest{SegmentId: c.Param("segmentId")},
	)
	if err != nil {
		response.Internal(c, "failed to fetch work types", nil)
		return
	}

	response.Success(c, http.StatusOK, workTypesResp)
}

// ListWorkTypeFields godoc
//
//	@Summary	List dynamic attribute fields for a work type
//	@Tags		WorkTypes
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workTypeId	path		string	true	"Work type ID"
//	@Success	200			{object}	response.SuccessResponse{data=userv1.ListWorkTypeFieldsResponse}
//	@Failure	401			{object}	response.ErrorResponse
//	@Failure	500			{object}	response.ErrorResponse
//	@Router		/work-types/{workTypeId}/fields [get]
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

// ListWorkTypeRateTypes godoc
//
//	@Summary	List allowed rate types for a work type
//	@Tags		WorkTypes
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workTypeId	path		string	true	"Work type ID"
//	@Success	200			{object}	response.SuccessResponse{data=userv1.ListRateTypesResponse}
//	@Failure	401			{object}	response.ErrorResponse
//	@Failure	500			{object}	response.ErrorResponse
//	@Router		/work-types/{workTypeId}/rate-types [get]
func (h *UserHandler) ListWorkTypeRateTypes(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	workTypeID := c.Param("workTypeId")

	rateTypesResp, err := h.userClient.ListWorkTypeRateTypes(
		ctx,
		&userv1.ListWorkTypeRateTypesRequest{WorkTypeId: workTypeID},
	)
	if err != nil {
		response.Internal(c, "failed to fetch rate types", nil)
		return
	}

	response.Success(c, http.StatusOK, rateTypesResp)
}

// ListWorkSegmentRateTypes godoc
//
//	@Summary	List allowed rate types for a work segment
//	@Tags		WorkTypes
//	@Produce	json
//	@Security	BearerAuth
//	@Param		segmentId	path		string	true	"Work segment ID"
//	@Success	200			{object}	response.SuccessResponse{data=userv1.ListRateTypesResponse}
//	@Failure	401			{object}	response.ErrorResponse
//	@Failure	500			{object}	response.ErrorResponse
//	@Router		/work-segments/{segmentId}/rate-types [get]
func (h *UserHandler) ListWorkSegmentRateTypes(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	rateTypesResp, err := h.userClient.ListWorkSegmentRateTypes(
		ctx,
		&userv1.ListWorkSegmentRateTypesRequest{SegmentId: c.Param("segmentId")},
	)
	if err != nil {
		response.Internal(c, "failed to fetch rate types", nil)
		return
	}

	response.Success(c, http.StatusOK, rateTypesResp)
}

// ListWorkTypeSkills godoc
//
//	@Summary	List reusable skills mapped to a work type
//	@Tags		WorkTypes
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workTypeId	path		string	true	"Work type ID"
//	@Success	200			{object}	response.SuccessResponse{data=userv1.ListWorkTypeSkillsResponse}
//	@Failure	401			{object}	response.ErrorResponse
//	@Failure	500			{object}	response.ErrorResponse
//	@Router		/work-types/{workTypeId}/skills [get]
func (h *UserHandler) ListWorkTypeSkills(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	workTypeID := c.Param("workTypeId")

	skillsResp, err := h.userClient.ListWorkTypeSkills(
		ctx,
		&userv1.ListWorkTypeSkillsRequest{WorkTypeId: workTypeID},
	)
	if err != nil {
		response.Internal(c, "failed to fetch skills", nil)
		return
	}

	response.Success(c, http.StatusOK, skillsResp)
}

// ListWorkTypeCommonRoles godoc
//
//	@Summary	List common roles mapped to a work type
//	@Tags		WorkTypes
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workTypeId	path		string	true	"Work type ID"
//	@Success	200			{object}	response.SuccessResponse{data=userv1.ListWorkTypeCommonRolesResponse}
//	@Failure	401			{object}	response.ErrorResponse
//	@Failure	500			{object}	response.ErrorResponse
//	@Router		/work-types/{workTypeId}/common-roles [get]
func (h *UserHandler) ListWorkTypeCommonRoles(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	workTypeID := c.Param("workTypeId")

	rolesResp, err := h.userClient.ListWorkTypeCommonRoles(
		ctx,
		&userv1.ListWorkTypeCommonRolesRequest{WorkTypeId: workTypeID},
	)
	if err != nil {
		response.Internal(c, "failed to fetch common roles", nil)
		return
	}

	response.Success(c, http.StatusOK, rolesResp)
}

// GetWorkTypeConfiguration godoc
//
//	@Summary	Get the bundled configuration for a work type
//	@Description	Returns skills, common roles, rate types, and dynamic attribute fields for a work type in one call.
//	@Tags		WorkTypes
//	@Produce	json
//	@Security	BearerAuth
//	@Param		workTypeId	path		string	true	"Work type ID"
//	@Success	200			{object}	response.SuccessResponse{data=userv1.GetWorkTypeConfigurationResponse}
//	@Failure	401			{object}	response.ErrorResponse
//	@Failure	500			{object}	response.ErrorResponse
//	@Router		/work-types/{workTypeId}/configuration [get]
func (h *UserHandler) GetWorkTypeConfiguration(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	workTypeID := c.Param("workTypeId")

	configResp, err := h.userClient.GetWorkTypeConfiguration(
		ctx,
		&userv1.GetWorkTypeConfigurationRequest{WorkTypeId: workTypeID},
	)
	if err != nil {
		response.Internal(c, "failed to fetch work type configuration", nil)
		return
	}

	response.Success(c, http.StatusOK, configResp)
}

// ListWorkCategoryRateTypes godoc
//
//	@Summary	List allowed rate types for a work category
//	@Tags		WorkTypes
//	@Produce	json
//	@Security	BearerAuth
//	@Param		categoryId	path		string	true	"Work category ID"
//	@Success	200			{object}	response.SuccessResponse{data=userv1.ListRateTypesResponse}
//	@Failure	401			{object}	response.ErrorResponse
//	@Failure	500			{object}	response.ErrorResponse
//	@Router		/work-categories/{categoryId}/rate-types [get]
func (h *UserHandler) ListWorkCategoryRateTypes(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	categoryID := c.Param("categoryId")

	rateTypesResp, err := h.userClient.ListWorkCategoryRateTypes(
		ctx,
		&userv1.ListWorkCategoryRateTypesRequest{CategoryId: categoryID},
	)
	if err != nil {
		response.Internal(c, "failed to fetch rate types", nil)
		return
	}

	response.Success(c, http.StatusOK, rateTypesResp)
}

// GetProfile godoc
//
//	@Summary	Get the current user's profile
//	@Tags		Profile
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	response.SuccessResponse{data=userv1.GetProfileResponse}
//	@Failure	401	{object}	response.ErrorResponse
//	@Failure	404	{object}	response.ErrorResponse	"Profile not found"
//	@Failure	500	{object}	response.ErrorResponse
//	@Router		/users/profile [get]
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

// CreateProfile godoc
//
//	@Summary	Create the current user's profile
//	@Tags		Profile
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		request	body		CreateProfileRequest	true	"Profile details"
//	@Success	201		{object}	response.SuccessResponse{data=userv1.CreateProfileResponse}
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	401		{object}	response.ErrorResponse
//	@Failure	409		{object}	response.ErrorResponse	"Profile already exists"
//	@Failure	500		{object}	response.ErrorResponse
//	@Router		/users/profile [post]
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
				Gender:     request.Gender,
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

// UpdateProfile godoc
//
//	@Summary	Update the current user's profile
//	@Tags		Profile
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		request	body		UpdateProfileRequest	true	"Profile details"
//	@Success	200		{object}	response.SuccessResponse{data=userv1.UpdateProfileResponse}
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	401		{object}	response.ErrorResponse
//	@Failure	404		{object}	response.ErrorResponse	"Profile not found"
//	@Failure	500		{object}	response.ErrorResponse
//	@Router		/users/profile [put]
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
		Gender:     request.Gender,
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

// CreateWorkerProfile godoc
//
//	@Summary	Create the current user's worker profile
//	@Description	Onboards the user as a worker and records their initial sector/segment selections.
//	@Tags		Profile
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		request	body		CreateWorkerProfileRequest	true	"Worker profile details and category/segment selections"
//	@Success	201		{object}	response.SuccessResponse{data=userv1.CreateWorkerProfileResponse}
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	401		{object}	response.ErrorResponse
//	@Failure	409		{object}	response.ErrorResponse	"Worker profile already exists"
//	@Failure	500		{object}	response.ErrorResponse
//	@Router		/users/worker-profile [post]
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
			ExpectedRateMin:    request.ExpectedRateMin,
			ExpectedRateMax:    request.ExpectedRateMax,
			RateBasis:          request.RateBasis,
			RateUnit:           request.RateUnit,
			AvailabilityStatus: request.AvailabilityStatus,
			Bio:                request.Bio,
			Selections:         workerSelectionsToProto(request.Selections),
		},
	)
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			response.BadRequest(c, status.Convert(err).Message(), nil)
		case codes.AlreadyExists:
			response.Conflict(c, "worker profile already exists", nil)
		default:
			response.Internal(c, "failed to save worker profile", nil)
		}
		return
	}

	response.Success(c, http.StatusCreated, createResp)
}

// UpdateWorkerProfile godoc
//
//	@Summary	Update the current user's worker profile selections
//	@Description	Updates the worker's selected work types, skills, and rate preferences.
//	@Tags		Profile
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		request	body		UpdateWorkerProfileRequest	true	"Updated work type/skill selections and rate preferences"
//	@Success	200		{object}	response.SuccessResponse{data=userv1.UpdateWorkerProfileResponse}
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	401		{object}	response.ErrorResponse
//	@Failure	404		{object}	response.ErrorResponse	"Worker profile not found"
//	@Failure	500		{object}	response.ErrorResponse
//	@Router		/users/worker-profile [put]
func (h *UserHandler) UpdateWorkerProfile(c *gin.Context) {
	var request UpdateWorkerProfileRequest
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

	updateResp, err := h.userClient.UpdateWorkerProfile(
		ctx,
		&userv1.UpdateWorkerProfileRequest{
			UserId:     userIDString,
			Selections: workerSelectionsToProto(request.Selections),
			RateBasis:  request.RateBasis,
			RateUnit:   request.RateUnit,
		},
	)
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound:
			response.NotFound(c, "worker profile not found", nil)
		case codes.InvalidArgument:
			response.BadRequest(c, status.Convert(err).Message(), nil)
		default:
			response.Internal(c, "failed to update worker profile", nil)
		}
		return
	}

	response.Success(c, http.StatusOK, updateResp)
}

func workerSelectionsToProto(selections []WorkerCategorySelection) []*userv1.WorkerCategorySelection {
	result := make([]*userv1.WorkerCategorySelection, 0, len(selections))
	for _, selection := range selections {
		segments := make([]*userv1.WorkerSegmentSelection, 0, len(selection.Segments))
		for _, segment := range selection.Segments {
			workTypes := make([]*userv1.WorkerTypeSelection, 0, len(segment.WorkTypes))
			for _, workType := range segment.WorkTypes {
				workTypes = append(workTypes, &userv1.WorkerTypeSelection{
					WorkTypeId: workType.WorkTypeID,
					SkillIds:   workType.SkillIDs,
				})
			}
			segments = append(segments, &userv1.WorkerSegmentSelection{
				WorkSegmentId: segment.WorkSegmentID,
				WorkTypes:     workTypes,
			})
		}
		result = append(result, &userv1.WorkerCategorySelection{
			WorkCategoryId: selection.WorkCategoryID,
			Segments:       segments,
		})
	}
	return result
}
