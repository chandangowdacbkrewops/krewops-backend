package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/model"
	"github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) RequestOTP(c *gin.Context) {
	var req model.RequestOTPRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.authService.RequestOTP(c.Request.Context(), req.PhoneNumber); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to send OTP"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent successfully"})
}

func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req model.VerifyOTPRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.authService.VerifyOTP(c.Request.Context(), req.PhoneNumber, req.OTP)
	if err != nil {
		if strings.Contains(err.Error(), "invalid or expired OTP") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired OTP"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to verify OTP"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req model.RefreshTokenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req model.LogoutRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.authService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

func (h *AuthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "auth-service",
		"status":  "ok",
	})
}
