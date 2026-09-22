package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/handler"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/middleware"
)

func RegisterRoutes(
	router *gin.Engine,
	searchHandler *handler.SearchHandler,
	jwtSecret string,
) {
	router.GET("/health", searchHandler.Health)

	v1 := router.Group("/api/v1")
	{
		search := v1.Group("/search")
		search.Use(middleware.AuthMiddleware(jwtSecret))
		{
			search.GET("/work", searchHandler.SearchWork)
			search.GET("/workers", searchHandler.SearchWorkers)
		}
	}
}
