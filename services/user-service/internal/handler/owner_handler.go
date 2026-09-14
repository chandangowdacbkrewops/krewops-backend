package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/middleware"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/service"
)

type OwnerHandler struct {
	ownerService *service.OwnerService
}

func NewOwnerHandler(ownerService *service.OwnerService) *OwnerHandler {
	return &OwnerHandler{ownerService: ownerService}
}

func (h *OwnerHandler) ListBusinessTypes(c *gin.Context) {
	types, err := h.ownerService.ListBusinessTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to fetch business types"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": types})
}

func (h *OwnerHandler) CreateOwnerProfile(c *gin.Context) {
	authUserID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req model.CreateOwnerProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.ownerService.CreateOwnerProfile(c.Request.Context(), authUserID, req)
	log.Printf("CreateOwnerProfile error: %v", err)
	if err != nil {
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to create owner profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func isValidationError(err error) bool {
	msg := err.Error()
	validationPrefixes := []string{
		"registration_type",
		"business_name",
		"email",
		"business_type_id",
		"date_of_birth",
		"user must be",
	}
	for _, prefix := range validationPrefixes {
		if strings.HasPrefix(msg, prefix) {
			return true
		}
	}
	return false
}
