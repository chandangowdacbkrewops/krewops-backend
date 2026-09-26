package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/client"
	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/handler"
	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/middleware"
)

func main() {

	_ = godotenv.Load()

	authAddress :=
		os.Getenv("AUTH_GRPC_ADDRESS")

	userAddress :=
		os.Getenv("USER_GRPC_ADDRESS")

	workAddress :=
		os.Getenv("WORK_GRPC_ADDRESS")

	searchAddress :=
		os.Getenv("SEARCH_GRPC_ADDRESS")

	jwtSecret :=
		os.Getenv("JWT_SECRET")

	port :=
		os.Getenv("HTTP_PORT")

	if port == "" {
		port = "8089"
	}

	authClient, err :=
		client.NewAuthClient(authAddress)

	if err != nil {
		log.Fatal(err)
	}

	defer authClient.Conn.Close()

	userClient, err :=
		client.NewUserClient(userAddress)

	if err != nil {
		log.Fatal(err)
	}

	defer userClient.Conn.Close()

	workClient, err :=
		client.NewWorkClient(workAddress)

	if err != nil {
		log.Fatal(err)
	}

	defer workClient.Conn.Close()

	searchClient, err :=
		client.NewSearchClient(searchAddress)

	if err != nil {
		log.Fatal(err)
	}

	defer searchClient.Conn.Close()

	authHandler :=
		handler.NewAuthHandler(
			authClient.Client,
			userClient.Client,
		)

	userHandler :=
		handler.NewUserHandler(
			userClient.Client,
		)

	workHandler :=
		handler.NewWorkHandler(
			workClient.Client,
		)

	searchHandler :=
		handler.NewSearchHandler(
			searchClient.Client,
		)

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(middleware.RequestID())
	router.Use(middleware.RecoveryWithEnvelope())
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
	}))

	api := router.Group("/api/v1")
	api.GET(
		"/work-types",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkTypes,
	)

	api.GET(
		"/work-categories",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkCategories,
	)

	api.GET(
		"/work-categories/:categoryId/work-types",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkTypesByCategory,
	)

	api.GET(
		"/work-types/:workTypeId/fields",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkTypeFields,
	)

	api.GET(
		"/work-types/:workTypeId/payment-types",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkTypePaymentTypes,
	)

	api.GET(
		"/work-categories/:categoryId/payment-types",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkCategoryPaymentTypes,
	)

	authRoutes :=
		api.Group("/auth")

	authRoutes.POST(
		"/otp/request",
		authHandler.RequestOTP,
	)

	authRoutes.POST(
		"/otp/verify",
		authHandler.VerifyOTP,
	)

	userRoutes :=
		api.Group("/users")

	userRoutes.Use(
		middleware.AuthMiddleware(jwtSecret),
	)

	userRoutes.GET(
		"/profile",
		userHandler.GetProfile,
	)

	userRoutes.POST(
		"/profile",
		userHandler.CreateProfile,
	)

	userRoutes.PUT(
		"/profile",
		userHandler.UpdateProfile,
	)

	userRoutes.POST(
		"/worker-profile",
		userHandler.CreateWorkerProfile,
	)

	workRoutes :=
		api.Group("/work")

	workRoutes.Use(
		middleware.AuthMiddleware(jwtSecret),
	)

	workRoutes.POST(
		"",
		workHandler.CreateWork,
	)

	workRoutes.GET(
		"",
		workHandler.ListMyWorks,
	)

	workRoutes.GET(
		"/applications",
		workHandler.ListMyApplications,
	)

	workRoutes.POST(
		"/:workId/applications",
		workHandler.ApplyToWork,
	)

	workRoutes.GET(
		"/:workId/applications",
		workHandler.ListWorkApplications,
	)

	workRoutes.POST(
		"/:workId/applications/:applicationId/shortlist",
		workHandler.ShortlistWorkApplication,
	)

	workRoutes.POST(
		"/:workId/applications/:applicationId/accept",
		workHandler.AcceptWorkApplication,
	)

	workRoutes.POST(
		"/:workId/applications/:applicationId/reject",
		workHandler.RejectWorkApplication,
	)

	workRoutes.POST(
		"/:workId/applications/:applicationId/withdraw",
		workHandler.WithdrawWorkApplication,
	)

	workRoutes.POST(
		"/:workId/applications/:applicationId/cancel",
		workHandler.CancelWorkApplication,
	)

	searchRoutes :=
		api.Group("/search")

	searchRoutes.Use(
		middleware.AuthMiddleware(jwtSecret),
	)

	searchRoutes.GET(
		"/work",
		searchHandler.SearchWork,
	)

	searchRoutes.GET(
		"/workers",
		searchHandler.SearchWorkers,
	)

	log.Printf(
		"API Gateway listening on :%s",
		port,
	)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
