package main

import (
	"context"
	"log"
	"net"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	userv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/user/v1"

	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/config"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/database"
	grpcserver "github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/grpc"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/handler"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/repository"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/routes"
	"github.com/chandangowdacbkrewops/krewops-backend/services/user-service/internal/service"
)

func main() {
	ctx := context.Background()

	// -------------------------
	// Configuration
	// -------------------------

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(
			"failed to load configuration: ",
			err,
		)
	}

	// -------------------------
	// Database migrations
	// -------------------------

	if err := database.RunMigrations(
		cfg.DatabaseURL,
		"migrations",
		"user_schema_migrations",
	); err != nil {
		log.Fatal(
			"failed to run migrations: ",
			err,
		)
	}

	// -------------------------
	// Database connection
	// -------------------------

	db, err := database.NewPostgresPool(
		ctx,
		cfg.DatabaseURL,
	)
	if err != nil {
		log.Fatal(
			"failed to connect database: ",
			err,
		)
	}

	defer db.Close()

	// -------------------------
	// Repositories
	// -------------------------

	profileRepository :=
		repository.NewProfileRepository(db)

	ownerRepository :=
		repository.NewOwnerRepository(db)

	businessTypeRepository :=
		repository.NewBusinessTypeRepository(db)

	workTypeRepository :=
		repository.NewWorkTypeRepository(db)

	workCategoryRepository :=
		repository.NewWorkCategoryRepository(db)

	workTypeFieldRepository :=
		repository.NewWorkTypeFieldRepository(db)

	paymentTypeRepository :=
		repository.NewPaymentTypeRepository(db)

	// -------------------------
	// Services
	// -------------------------

	ownerService :=
		service.NewOwnerService(
			ownerRepository,
			businessTypeRepository,
		)

	profileService :=
		service.NewProfileService(
			profileRepository,
			ownerService,
			workTypeRepository,
			workCategoryRepository,
			workTypeFieldRepository,
			paymentTypeRepository,
		)

	userService :=
		service.NewUserService(
			profileService,
		)

	// -------------------------
	// HTTP / Gin
	// -------------------------

	profileHandler :=
		handler.NewProfileHandler(
			profileService,
		)

	ownerHandler :=
		handler.NewOwnerHandler(
			ownerService,
		)

	router := gin.Default()

	routes.RegisterRoutes(
		router,
		profileHandler,
		ownerHandler,
		cfg.JWTSecret,
	)

	// -------------------------
	// gRPC
	// -------------------------

	listener, err := net.Listen(
		"tcp",
		":"+cfg.GRPCPort,
	)

	if err != nil {
		log.Fatal(
			"failed to listen on gRPC port "+cfg.GRPCPort+": ",
			err,
		)
	}

	grpcServer := grpc.NewServer()

	userGRPCServer :=
		grpcserver.NewUserServer(
			userService,
		)

	userv1.RegisterUserServiceServer(
		grpcServer,
		userGRPCServer,
	)

	// Start gRPC server concurrently
	go func() {
		log.Println(
			"user-service gRPC listening on :" + cfg.GRPCPort,
		)

		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal(
				"user-service gRPC server failed: ",
				err,
			)
		}
	}()

	// -------------------------
	// HTTP server
	// -------------------------

	log.Printf(
		"user-service HTTP listening on :%s",
		cfg.Port,
	)

	if err := router.Run(
		":" + cfg.Port,
	); err != nil {
		log.Fatal(
			"user-service HTTP server failed: ",
			err,
		)
	}
}
