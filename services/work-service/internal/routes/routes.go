package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/handler"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/middleware"
)

func RegisterRoutes(
	router *gin.Engine,
	workHandler *handler.WorkHandler,
	jwtSecret string,
) {
	router.GET("/health", workHandler.Health)

	v1 := router.Group("/api/v1")
	{
		work := v1.Group("/work")
		work.Use(middleware.AuthMiddleware(jwtSecret))
		{
			work.POST("", workHandler.CreateWork)
		}
	}
}
