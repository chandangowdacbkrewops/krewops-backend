package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/chandangowdacbkrewops/krewops-backend/gateway/docs"
	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/client"
	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/handler"
	"github.com/chandangowdacbkrewops/krewops-backend/gateway/internal/middleware"
)

// @title						KrewOps API Gateway
// @version						1.0
// @description					Public REST API exposed by the KrewOps API Gateway. The gateway fronts the auth,
// @description					user, work, and search gRPC services, exposing them as a single JSON/REST surface
// @description					for mobile and web clients. All responses are wrapped in a consistent envelope
// @description					(see response.SuccessResponse / response.ErrorResponse).
// @BasePath					/api/v1
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description					Bearer JWT access token issued by /auth/otp/verify. Pass as "Bearer <token>".
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
		"/work-categories/:categoryId/segments",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkSegmentsByCategory,
	)

	api.GET(
		"/work-segments/:segmentId/work-types",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkTypesBySegment,
	)

	api.GET(
		"/work-types/:workTypeId/fields",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkTypeFields,
	)

	api.GET(
		"/work-types/:workTypeId/rate-types",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkTypeRateTypes,
	)

	api.GET(
		"/work-segments/:segmentId/rate-types",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkSegmentRateTypes,
	)

	api.GET(
		"/work-types/:workTypeId/skills",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkTypeSkills,
	)

	api.GET(
		"/work-types/:workTypeId/common-roles",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkTypeCommonRoles,
	)

	api.GET(
		"/work-types/:workTypeId/configuration",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.GetWorkTypeConfiguration,
	)

	api.GET(
		"/work-categories/:categoryId/rate-types",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkCategoryRateTypes,
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

	userRoutes.PUT(
		"/worker-profile",
		userHandler.UpdateWorkerProfile,
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

	workRoutes.POST(
		"/:workId/start",
		workHandler.StartWork,
	)

	workRoutes.POST(
		"/:workId/complete",
		workHandler.CompleteWork,
	)

	workRoutes.POST(
		"/:workId/confirm-completion",
		workHandler.ConfirmWorkCompletion,
	)

	workRoutes.POST(
		"/:workId/complete-job",
		workHandler.CompleteJob,
	)

	workRoutes.POST(
		"/:workId/close-applications",
		workHandler.CloseApplications,
	)

	workRoutes.POST(
		"/:workId/close",
		workHandler.CloseWork,
	)

	workRoutes.POST(
		"/:workId/cancel",
		workHandler.CancelWork,
	)

	workRoutes.GET(
		"/:workId/daily-logs",
		workHandler.ListWorkDailyLogs,
	)

	workRoutes.GET(
		"/:workId/settlement",
		workHandler.GetWorkSettlement,
	)

	workRoutes.POST(
		"/:workId/assignments/:assignmentId/payments",
		workHandler.RecordAssignmentPayment,
	)

	workRoutes.GET("/:workId/work-records/summary", workHandler.GetWorkerWorkSummary)
	workRoutes.GET("/:workId/work-records", workHandler.GetWorkDayRecords)
	workRoutes.POST("/:workId/work-records", workHandler.SaveWorkDayRecords)
	workRoutes.GET("/:workId/contractor-work-records/summary", workHandler.GetContractorWorkSummary)
	workRoutes.GET("/:workId/contractor-work-records", workHandler.GetContractorWorkRecords)
	workRoutes.POST("/:workId/contractor-work-records", workHandler.SaveContractorWorkRecord)
	workRoutes.GET("/:workId/payments", workHandler.GetWorkPayments)

	worksRoutes := api.Group("/works")
	worksRoutes.Use(middleware.AuthMiddleware(jwtSecret))
	worksRoutes.POST("/:workId/start", workHandler.StartWork)
	worksRoutes.POST("/:workId/complete", workHandler.CompleteWork)
	worksRoutes.POST("/:workId/close", workHandler.CloseWork)
	worksRoutes.GET("/:workId/work-records/summary", workHandler.GetWorkerWorkSummary)
	worksRoutes.GET("/:workId/work-records", workHandler.GetWorkDayRecords)
	worksRoutes.POST("/:workId/work-records", workHandler.SaveWorkDayRecords)
	worksRoutes.GET("/:workId/contractor-work-records/summary", workHandler.GetContractorWorkSummary)
	worksRoutes.GET("/:workId/contractor-work-records", workHandler.GetContractorWorkRecords)
	worksRoutes.POST("/:workId/contractor-work-records", workHandler.SaveContractorWorkRecord)
	worksRoutes.GET("/:workId/payments", workHandler.GetWorkPayments)

	obligationRoutes := api.Group("/payment-obligations")
	obligationRoutes.Use(middleware.AuthMiddleware(jwtSecret))
	obligationRoutes.POST("/:paymentId/record-payment", workHandler.RecordObligationPayment)

	assignmentRoutes := api.Group("/work-assignments")
	assignmentRoutes.Use(middleware.AuthMiddleware(jwtSecret))
	assignmentRoutes.POST("/:assignmentId/start", workHandler.StartAssignment)
	assignmentRoutes.POST("/:assignmentId/complete", workHandler.CompleteAssignment)
	assignmentRoutes.PUT("/:assignmentId/daily-logs/:date", workHandler.UpsertDailyLog)
	assignmentRoutes.GET("/:assignmentId/daily-logs", workHandler.ListAssignmentDailyLogs)
	assignmentRoutes.GET("/:assignmentId/payments", workHandler.ListAssignmentPayments)
	assignmentRoutes.POST("/:assignmentId/payments/:paymentId/confirm", workHandler.ConfirmAssignmentPayment)
	assignmentRoutes.POST("/:assignmentId/payments/:paymentId/dispute", workHandler.DisputeAssignmentPayment)

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

	router.GET(
		"/swagger/*any",
		ginSwagger.WrapHandler(swaggerFiles.Handler),
	)

	log.Printf(
		"API Gateway listening on :%s",
		port,
	)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
