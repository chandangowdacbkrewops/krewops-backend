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
			City:            optionalQueryParam(c, "city"),
			State:           optionalQueryParam(c, "state"),
			ExperienceLevel: optionalQueryParam(c, "experience_level"),
			PaymentType:     optionalQueryParam(c, "payment_type"),
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
