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

func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	authUserID, ok := middleware.GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req model.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	profile, err := h.profileService.UpdateProfile(c.Request.Context(), authUserID, req)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrProfileNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		case errors.Is(err, service.ErrInvalidProfile):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to update profile"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": profile})
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

func (h *ProfileHandler) ListWorkCategories(c *gin.Context) {
	categories, err := h.profileService.ListWorkCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to fetch work categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": categories})
}

func (h *ProfileHandler) ListWorkTypesByCategory(c *gin.Context) {
	categoryID := c.Param("categoryId")

	types, err := h.profileService.ListWorkTypesByCategory(c.Request.Context(), categoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to fetch work types"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": types})
}

func (h *ProfileHandler) ListWorkTypeFields(c *gin.Context) {
	workTypeID := c.Param("workTypeId")

	fields, err := h.profileService.ListWorkTypeFields(c.Request.Context(), workTypeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to fetch work type fields"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": fields})
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
