package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/middleware"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/repository"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/service"
)

type ProfileHandler struct {
	profileService *service.ProfileService
}

func NewProfileHandler(profileService *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{profileService: profileService}
}

func (h *ProfileHandler) GetMe(c *gin.Context) {
	authUserID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	profile, err := h.profileService.GetProfile(c.Request.Context(), authUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to fetch profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": profile})
}

func (h *ProfileHandler) CreateProfile(c *gin.Context) {
	authUserID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req model.CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	profile, err := h.profileService.CreateProfile(c.Request.Context(), authUserID, req)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrProfileAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": "profile already exists"})
		case errors.Is(err, service.ErrInvalidProfile):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to save profile"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": profile})
}

func (h *ProfileHandler) CreateWorkerProfile(c *gin.Context) {
	authUserID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req model.CreateWorkerProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	profile, err := h.profileService.CreateWorkerProfile(c.Request.Context(), authUserID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unable to save worker profile"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": profile})
}

func (h *ProfileHandler) ListWorkTypes(c *gin.Context) {
	types, err := h.profileService.ListWorkTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to fetch work types"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": types})
}

func (h *ProfileHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "user-service",
		"status":  "ok",
	})
}

func (h *ProfileHandler) NotImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
