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
			work.GET("", workHandler.ListMyWorks)
			work.GET("/applications", workHandler.ListMyApplications)
			work.POST("/:workId/applications", workHandler.ApplyToWork)
			work.GET("/:workId/applications", workHandler.ListWorkApplications)
			work.POST("/:workId/applications/:applicationId/shortlist", workHandler.ShortlistWorkApplication)
			work.POST("/:workId/applications/:applicationId/accept", workHandler.AcceptWorkApplication)
			work.POST("/:workId/applications/:applicationId/reject", workHandler.RejectWorkApplication)
			work.POST("/:workId/applications/:applicationId/withdraw", workHandler.WithdrawWorkApplication)
			work.POST("/:workId/applications/:applicationId/cancel", workHandler.CancelWorkApplication)
		}
	}
}
