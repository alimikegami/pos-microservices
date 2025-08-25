package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/alimikegami/point-of-sales/product-query-service/config"
	"github.com/alimikegami/point-of-sales/product-query-service/internal/controller"
	"github.com/alimikegami/point-of-sales/product-query-service/internal/handler"
	"github.com/alimikegami/point-of-sales/product-query-service/internal/infrastructure/message-queue/kafka"
	"github.com/alimikegami/point-of-sales/product-query-service/internal/infrastructure/tracing"
	"github.com/alimikegami/point-of-sales/product-query-service/internal/repository"
	"github.com/alimikegami/point-of-sales/product-query-service/internal/service"
	"github.com/alimikegami/point-of-sales/product-query-service/pkg/dto"
	pb "github.com/alimikegami/pos-microservices/proto-defs/pb"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

func main() {
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).With().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = logger

	config := config.CreateNewConfig()

	kafkaProducer := kafka.CreateKafkaProducer(config)
	kafkaReader := kafka.CreateKafkaReader(config)

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

	tracer := traceProvider.Tracer("product-query-service")

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

	elasticSearchRepo := repository.CreateNewElasticSearchRepository(config)
	svc := service.CreateProductService(elasticSearchRepo, *config, kafkaReader, kafkaProducer)
	controller.CreateProductController(g, svc, IsLoggedIn)

	g.GET("/ping", func(c echo.Context) error {
		return dto.WriteSuccessResponse(c, "Hello, World!")
	})

	srv := grpc.NewServer()
	productGrpcServer := handler.CreateGRPCHandler(svc)
	pb.RegisterProductQueryServiceServer(srv, productGrpcServer)

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%s", config.GrpcServicePort))
		if err != nil {
			log.Error().Err(err).Msg("Failed to listen on port 50051")
			return
		}
		log.Info().Msg("gRPC server started on port 50051")
		if err := srv.Serve(lis); err != nil {
			log.Error().Err(err).Msg("Failed to serve gRPC server")
		}
	}()

	go svc.ConsumeEvent()

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", config.ServicePort)))
}
