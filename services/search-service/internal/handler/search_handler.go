package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/service"
)

type SearchHandler struct {
	searchService *service.SearchService
}

func NewSearchHandler(searchService *service.SearchService) *SearchHandler {
	return &SearchHandler{searchService: searchService}
}

func (h *SearchHandler) SearchWork(c *gin.Context) {
	filters := model.WorkSearchFilters{
		Keyword:         optionalQuery(c, "keyword"),
		WorkTypeID:      optionalQuery(c, "work_type_id"),
		WorkCategoryID:  optionalQuery(c, "work_category_id"),
		City:            optionalQuery(c, "city"),
		State:           optionalQuery(c, "state"),
		ExperienceLevel: optionalQuery(c, "experience_level"),
		PaymentType:     optionalQuery(c, "payment_type"),
		MinBudgetRate:   optionalQueryFloat(c, "min_budget_rate"),
		MaxBudgetRate:   optionalQueryFloat(c, "max_budget_rate"),
		Page:            queryInt(c, "page", model.DefaultPage),
		PageSize:        queryInt(c, "page_size", model.DefaultPageSize),
	}

	result, err := h.searchService.SearchWork(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to search work postings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *SearchHandler) SearchWorkers(c *gin.Context) {
	filters := model.WorkerSearchFilters{
		Keyword:            optionalQuery(c, "keyword"),
		WorkTypeID:         optionalQuery(c, "work_type_id"),
		City:               optionalQuery(c, "city"),
		State:              optionalQuery(c, "state"),
		WorkerType:         optionalQuery(c, "worker_type"),
		AvailabilityStatus: optionalQuery(c, "availability_status"),
		MinExperienceYears: optionalQueryFloat(c, "min_experience_years"),
		Page:               queryInt(c, "page", model.DefaultPage),
		PageSize:           queryInt(c, "page_size", model.DefaultPageSize),
	}

	result, err := h.searchService.SearchWorkers(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to search workers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *SearchHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "search-service",
		"status":  "ok",
	})
}

func optionalQuery(c *gin.Context, key string) *string {
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

func queryInt(c *gin.Context, key string, fallback int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}
