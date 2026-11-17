package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/response"
	searchv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/search/v1"
	"google.golang.org/grpc/metadata"
)

type SearchHandler struct {
	searchClient searchv1.SearchServiceClient
}

func NewSearchHandler(
	searchClient searchv1.SearchServiceClient,
) *SearchHandler {

	return &SearchHandler{
		searchClient: searchClient,
	}
}

// SearchWork godoc
//
//	@Summary	Search work postings
//	@Tags		Search
//	@Produce	json
//	@Security	BearerAuth
//	@Param		keyword				query		string	false	"Free-text keyword search"
//	@Param		work_type_id		query		string	false	"Filter by work type ID"
//	@Param		work_category_id	query		string	false	"Filter by work category ID"
//	@Param		work_segment_id		query		string	false	"Filter by work segment ID"
//	@Param		city				query		string	false	"Filter by city"
//	@Param		state				query		string	false	"Filter by state"
//	@Param		experience_level	query		string	false	"Filter by experience level"
//	@Param		rate_basis			query		string	false	"Filter by rate basis"
//	@Param		rate_unit			query		string	false	"Filter by rate unit"
//	@Param		min_budget_rate		query		number	false	"Minimum budget rate"
//	@Param		max_budget_rate		query		number	false	"Maximum budget rate"
//	@Param		page				query		int		false	"Page number (1-indexed)"
//	@Param		page_size			query		int		false	"Page size"
//	@Success	200					{object}	response.SuccessResponse{data=searchv1.SearchWorkResponse}
//	@Failure	401					{object}	response.ErrorResponse
//	@Failure	500					{object}	response.ErrorResponse
//	@Router		/search/work [get]
func (h *SearchHandler) SearchWork(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

	page, pageSize := queryPagination(c)

	searchResp, err := h.searchClient.SearchWork(
		ctx,
		&searchv1.SearchWorkRequest{
			Keyword:         optionalQueryParam(c, "keyword"),
			WorkTypeId:      optionalQueryParam(c, "work_type_id"),
			WorkCategoryId:  optionalQueryParam(c, "work_category_id"),
			WorkSegmentId:   optionalQueryParam(c, "work_segment_id"),
			City:            optionalQueryParam(c, "city"),
			State:           optionalQueryParam(c, "state"),
			ExperienceLevel: optionalQueryParam(c, "experience_level"),
			RateBasis:       optionalQueryParam(c, "rate_basis"),
			RateUnit:        optionalQueryParam(c, "rate_unit"),
			MinBudgetRate:   optionalQueryFloat(c, "min_budget_rate"),
			MaxBudgetRate:   optionalQueryFloat(c, "max_budget_rate"),
			Page:            page,
			PageSize:        pageSize,
		},
	)
	if err != nil {
		response.Internal(c, "failed to search work postings", nil)
		return
	}

	response.Success(c, http.StatusOK, searchResp)
}

// SearchWorkers godoc
//
//	@Summary	Search worker profiles
//	@Tags		Search
//	@Produce	json
//	@Security	BearerAuth
//	@Param		keyword					query		string	false	"Free-text keyword search"
//	@Param		work_type_id			query		string	false	"Filter by work type ID"
//	@Param		work_segment_id			query		string	false	"Filter by work segment ID"
//	@Param		city					query		string	false	"Filter by city"
//	@Param		state					query		string	false	"Filter by state"
//	@Param		worker_type				query		string	false	"Filter by worker type"
//	@Param		availability_status		query		string	false	"Filter by availability status"
//	@Param		min_experience_years	query		number	false	"Minimum years of experience"
//	@Param		page					query		int		false	"Page number (1-indexed)"
//	@Param		page_size				query		int		false	"Page size"
//	@Success	200						{object}	response.SuccessResponse{data=searchv1.SearchWorkersResponse}
//	@Failure	401						{object}	response.ErrorResponse
//	@Failure	500						{object}	response.ErrorResponse
//	@Router		/search/workers [get]
func (h *SearchHandler) SearchWorkers(c *gin.Context) {
	requestID := c.GetString(response.RequestIDContextKey)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

	page, pageSize := queryPagination(c)

	searchResp, err := h.searchClient.SearchWorkers(
		ctx,
		&searchv1.SearchWorkersRequest{
			Keyword:            optionalQueryParam(c, "keyword"),
			WorkTypeId:         optionalQueryParam(c, "work_type_id"),
			WorkSegmentId:      optionalQueryParam(c, "work_segment_id"),
			City:               optionalQueryParam(c, "city"),
			State:              optionalQueryParam(c, "state"),
			WorkerType:         optionalQueryParam(c, "worker_type"),
			AvailabilityStatus: optionalQueryParam(c, "availability_status"),
			MinExperienceYears: optionalQueryFloat(c, "min_experience_years"),
			Page:               page,
			PageSize:           pageSize,
		},
	)
	if err != nil {
		response.Internal(c, "failed to search workers", nil)
		return
	}

	response.Success(c, http.StatusOK, searchResp)
}

func optionalQueryParam(c *gin.Context, key string) *string {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return nil
	}
	return &value
}

func optionalQueryFloat(c *gin.Context, key string) *float64 {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func queryPagination(c *gin.Context) (*int32, *int32) {
	var page, pageSize *int32

	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			value := int32(parsed)
			page = &value
		}
	}

	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			value := int32(parsed)
			pageSize = &value
		}
	}

	return page, pageSize
}
