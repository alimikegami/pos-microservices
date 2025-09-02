package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/alimikegami/point-of-sales/product-command-service/config"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/controller"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/handler"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/infrastructure/database/mongodb"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/infrastructure/message-queue/kafka"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/infrastructure/tracing"
	localmiddleware "github.com/alimikegami/point-of-sales/product-command-service/internal/middleware"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/repository"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/service"
	"github.com/alimikegami/point-of-sales/product-command-service/pkg/dto"
	pb "github.com/alimikegami/pos-microservices/proto-defs/pb"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = logger

	config := config.CreateNewConfig()
	db, err := mongodb.ConnectToMongoDB(config.MongoDBConfig.DBHost, config.MongoDBConfig.DBPort)
	if err != nil {
		panic(err)
	}

	kafkaProducer := kafka.CreateKafkaProducer(config)
	kafkaReader := kafka.CreateKafkaReader(config)

	defer db.Client().Disconnect(context.Background())

	e := echo.New()
	traceProvider, err := tracing.InitTracing(config.TracingConfig.CollectorHost)
	if err != nil {
		fmt.Println(err)
	}

	defer func() {
		if err := traceProvider.Shutdown(context.Background()); err != nil {
			fmt.Println(err)
		}
	}()

	tracer := traceProvider.Tracer("product-command-service")

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// span creation and naming
			ctx, span := tracer.Start(c.Request().Context(), fmt.Sprintf("[%s] %s", c.Request().Method, c.Path()))
			defer span.End()

			// add the context to the request
			req := c.Request()
			c.SetRequest(req.WithContext(ctx))

			return next(c)
		}
	})

	// Used empty string so that metrics are not prefixed with the service name making it easier to aggregate across services
	e.Use(echoprometheus.NewMiddleware(""))
	e.Use(localmiddleware.Logger)

	go func() {
		metrics := echo.New()
		metrics.GET("/metrics", echoprometheus.NewHandler())
		if err := metrics.Start(fmt.Sprintf(":%s", config.MetricsPort)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Failed to start metrics server")
		}
	}()

	g := e.Group("/api/v1")

	IsLoggedIn := middleware.JWTWithConfig(middleware.JWTConfig{
		SigningKey: []byte(config.JWTSecret),
		ErrorHandlerWithContext: func(err error, c echo.Context) error {
			// Custom logic to handle JWT validation errors
			// Return a custom JSON response
			errorResponse := map[string]interface{}{
				"status":  "error",
				"message": "Invalid or expired JWT",
				"errors":  nil,
			}
			return c.JSON(http.StatusUnauthorized, errorResponse)
		},
	})

	mongoDBRepo := repository.CreateNewMongoDBRepository(db)
	svc := service.CreateProductService(mongoDBRepo, *config, kafkaReader, kafkaProducer)
	controller.CreateProductController(g, svc, IsLoggedIn)

	g.GET("/ping", func(c echo.Context) error {
		return dto.WriteSuccessResponse(c, "Hello, World!")
	})

	go svc.ConsumeEvent()

	srv := grpc.NewServer()
	productGrpcServer := handler.CreateGRPCHandler(svc)
	pb.RegisterProductCommandServiceServer(srv, productGrpcServer)

	go func() {
		log.Info().Msgf("gRPC server is running on port %s", config.GrpcServicePort)
		lis, _ := net.Listen("tcp", fmt.Sprintf(":%s", config.GrpcServicePort))
		srv.Serve(lis)
	}()

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", config.ServicePort)))
}
