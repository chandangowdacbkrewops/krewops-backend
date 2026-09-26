package main

import (
	"context"
	"log"
	"net"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	workv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/work/v1"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/config"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/database"
	grpcserver "github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/grpc"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/handler"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/repository"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/routes"
	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/service"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load configuration: ", err)
	}

	if err := database.RunMigrations(
		cfg.DatabaseURL,
		"migrations",
		"work_schema_migrations",
	); err != nil {
		log.Fatal("failed to run migrations: ", err)
	}

	db, err := database.NewPostgresPool(
		ctx,
		cfg.DatabaseURL,
	)
	if err != nil {
		log.Fatal("failed to connect database: ", err)
	}

	defer db.Close()

	mongoClient, mongoDB, err := database.NewMongoDatabase(
		ctx,
		cfg.MongoURI,
		cfg.MongoDB,
	)
	if err != nil {
		log.Fatal("failed to connect mongo: ", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Printf("failed to disconnect mongo: %v", err)
		}
	}()

	// -------------------------
	// Repositories
	// -------------------------

	workRepository := repository.NewWorkRepository(mongoDB)
	if err := workRepository.EnsureIndexes(ctx); err != nil {
		log.Fatal("failed to ensure work posting indexes: ", err)
	}

	applicationRepository := repository.NewApplicationRepository(mongoDB)
	if err := applicationRepository.EnsureIndexes(ctx); err != nil {
		log.Fatal("failed to ensure work application indexes: ", err)
	}

	workOwnerRepository := repository.NewWorkOwnerRepository(db)
	workTypeRepository := repository.NewWorkTypeRepository(db)
	workCategoryRepository := repository.NewWorkCategoryRepository(db)
	workTypeFieldRepository := repository.NewWorkTypeFieldRepository(db)
	paymentTypeRepository := repository.NewPaymentTypeRepository(db)

	// -------------------------
	// Services
	// -------------------------

	workService := service.NewWorkService(
		workRepository,
		applicationRepository,
		workOwnerRepository,
		workTypeRepository,
		workCategoryRepository,
		workTypeFieldRepository,
		paymentTypeRepository,
	)

	// -------------------------
	// HTTP / Gin
	// -------------------------

	workHandler := handler.NewWorkHandler(
		workService,
	)

	router := gin.Default()

	routes.RegisterRoutes(
		router,
		workHandler,
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
			"failed to listen on gRPC port: ",
			err,
		)
	}

	grpcServer := grpc.NewServer()

	workv1.RegisterWorkServiceServer(
		grpcServer,
		grpcserver.NewWorkServer(
			workService,
		),
	)

	// Start gRPC concurrently
	go func() {
		log.Println(
			"work-service gRPC listening on :" + cfg.GRPCPort,
		)

		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal(
				"gRPC server failed: ",
				err,
			)
		}
	}()

	// Start HTTP server
	log.Printf(
		"work-service HTTP listening on :%s",
		cfg.Port,
	)

	if err := router.Run(
		":" + cfg.Port,
	); err != nil {
		log.Fatal(
			"HTTP server failed: ",
			err,
		)
	}
}
