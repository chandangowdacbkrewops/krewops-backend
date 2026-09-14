package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/handler"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/middleware"
)

func RegisterRoutes(
	router *gin.Engine,
	profileHandler *handler.ProfileHandler,
	ownerHandler *handler.OwnerHandler,
	jwtSecret string,
) {
	router.GET("/health", profileHandler.Health)

	v1 := router.Group("/api/v1")
	{
		users := v1.Group("/users")
		users.Use(middleware.AuthMiddleware(jwtSecret))
		{
			users.GET("/profile", profileHandler.GetMe)
			users.POST("/profile", profileHandler.CreateProfile)
			users.POST("/worker-profile", profileHandler.CreateWorkerProfile)
		}
	}
}
