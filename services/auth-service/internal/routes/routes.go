package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/services/auth-service/internal/handler"
)

func RegisterRoutes(router *gin.Engine, authHandler *handler.AuthHandler) {
	router.GET("/health", authHandler.Health)

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/otp/request", authHandler.RequestOTP)
			auth.POST("/otp/verify", authHandler.VerifyOTP)
			auth.POST("/token/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
		}
	}
}
