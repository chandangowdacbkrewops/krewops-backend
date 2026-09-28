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
		v1.GET(
			"/work-types",
			middleware.AuthMiddleware(jwtSecret),
			profileHandler.ListWorkTypes,
		)

		v1.GET(
			"/work-categories",
			middleware.AuthMiddleware(jwtSecret),
			profileHandler.ListWorkCategories,
		)

		v1.GET(
			"/work-categories/:categoryId/work-types",
			middleware.AuthMiddleware(jwtSecret),
			profileHandler.ListWorkTypesByCategory,
		)

		v1.GET(
			"/work-types/:workTypeId/fields",
			middleware.AuthMiddleware(jwtSecret),
			profileHandler.ListWorkTypeFields,
		)

		v1.GET(
			"/work-types/:workTypeId/payment-types",
			middleware.AuthMiddleware(jwtSecret),
			profileHandler.ListWorkTypePaymentTypes,
		)

		v1.GET(
			"/work-types/:workTypeId/skills",
			middleware.AuthMiddleware(jwtSecret),
			profileHandler.ListWorkTypeSkills,
		)

		v1.GET(
			"/work-categories/:categoryId/payment-types",
			middleware.AuthMiddleware(jwtSecret),
			profileHandler.ListWorkCategoryPaymentTypes,
		)

		users := v1.Group("/users")
		users.Use(middleware.AuthMiddleware(jwtSecret))
		{
			users.GET("/profile", profileHandler.GetMe)
			users.POST("/profile", profileHandler.CreateProfile)
			users.PUT("/profile", profileHandler.UpdateProfile)
			users.POST("/worker-profile", profileHandler.CreateWorkerProfile)
			users.PUT("/worker-profile", profileHandler.UpdateWorkerProfile)
		}
	}
}
