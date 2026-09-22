package main

import (
	"context"
	"log"
	"net"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	searchv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/search/v1"

	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/config"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/database"
	grpcserver "github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/grpc"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/handler"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/repository"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/routes"
	"github.com/chandangowdacbkrewops/krewops-backend/services/search-service/internal/service"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load configuration: ", err)
	}

	// search-service owns no tables, so it does not run any migrations. It
	// connects to the same shared database that work-service and
	// user-service manage the schema for, and only issues read queries.
	db, err := database.NewPostgresPool(
		ctx,
		cfg.DatabaseURL,
	)
	if err != nil {
		log.Fatal("failed to connect database: ", err)
	}

	defer db.Close()

	// -------------------------
	// Repositories
	// -------------------------

	workSearchRepository := repository.NewWorkSearchRepository(db)
	workerSearchRepository := repository.NewWorkerSearchRepository(db)

	// -------------------------
	// Services
	// -------------------------

	searchService := service.NewSearchService(
		workSearchRepository,
		workerSearchRepository,
	)

	// -------------------------
	// HTTP / Gin
	// -------------------------

	searchHandler := handler.NewSearchHandler(searchService)

	router := gin.Default()

	routes.RegisterRoutes(
		router,
		searchHandler,
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

	searchv1.RegisterSearchServiceServer(
		grpcServer,
		grpcserver.NewSearchServer(searchService),
	)

	go func() {
		log.Println(
			"search-service gRPC listening on :" + cfg.GRPCPort,
		)

		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal(
				"gRPC server failed: ",
				err,
			)
		}
	}()

	log.Printf(
		"search-service HTTP listening on :%s",
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
