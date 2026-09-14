package main

import (
	"log"
	"os"

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

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(middleware.RequestID())
	router.Use(middleware.RecoveryWithEnvelope())

	api := router.Group("/api/v1")
	api.GET(
		"/work-types",
		middleware.AuthMiddleware(jwtSecret),
		userHandler.ListWorkTypes,
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

	log.Printf(
		"API Gateway listening on :%s",
		port,
	)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
