package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "time/tzdata"

	"github.com/alimikegami/point-of-sales/order-service/config"
	"github.com/alimikegami/point-of-sales/order-service/internal/controller"
	circuitbreaker "github.com/alimikegami/point-of-sales/order-service/internal/infrastructure/circuit-breaker"
	"github.com/alimikegami/point-of-sales/order-service/internal/infrastructure/database/postgres"
	"github.com/alimikegami/point-of-sales/order-service/internal/infrastructure/message-queue/kafka"
	paymentgateway "github.com/alimikegami/point-of-sales/order-service/internal/infrastructure/payment-gateway"
	"github.com/alimikegami/point-of-sales/order-service/internal/infrastructure/tracing"
	"github.com/alimikegami/point-of-sales/order-service/internal/repository"
	"github.com/alimikegami/point-of-sales/order-service/internal/service"
	"github.com/alimikegami/point-of-sales/order-service/pkg/dto"
	pb "github.com/alimikegami/pos-microservices/proto-defs/pb"
	"github.com/go-co-op/gocron/v2"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = logger

	config := config.CreateNewConfig()

	midtransClient := paymentgateway.CreateMidtransClient(config)

	kafkaProducer := kafka.CreateKafkaProducer(config)
	kafkaReader := kafka.CreateKafkaReader(config)

	db, err := postgres.GetDBInstance(config.PostgreSQLConfig.DBUsername, config.PostgreSQLConfig.DBPassword, config.PostgreSQLConfig.DBHost, config.PostgreSQLConfig.DBPort, config.PostgreSQLConfig.DBName)
	if err != nil {
		panic(err)
	}

	traceProvider, err := tracing.InitTracing(config.TracingConfig.CollectorHost)
	if err != nil {
		fmt.Println(err)
	}

	defer func() {
		if err := traceProvider.Shutdown(context.Background()); err != nil {
			fmt.Println(err)
		}
	}()

	tracer := traceProvider.Tracer("pos-order-service")

	e := echo.New()
	g := e.Group("/api/v1")

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

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:      true,
		LogStatus:   true,
		LogLatency:  true,
		LogRemoteIP: true,
		LogMethod:   true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info().
				Str("method", v.Method).
				Str("URI", v.URI).
				Int("status", v.Status).
				Int64("latency", v.Latency.Microseconds()).
				Str("remote IP", v.RemoteIP).
				Msg("Request")

			return nil
		},
	}))

	g.GET("/ping", func(c echo.Context) error {
		return dto.WriteSuccessResponse(c, "Hello, World!")
	})

	productCommandGrpcConn, err := grpc.NewClient(config.ProductCommandServiceHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to gRPC server")
	}
	defer productCommandGrpcConn.Close()

	productCommandGrpcClient := pb.NewProductCommandServiceClient(productCommandGrpcConn)

	productQueryGrpcConn, err := grpc.NewClient(config.ProductQueryServiceHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to gRPC server")
	}
	defer productQueryGrpcConn.Close()

	productQueryGrpcClient := pb.NewProductQueryServiceClient(productQueryGrpcConn)

	cb := circuitbreaker.CreateCircuitBreaker("order-service")

	orderRepo := repository.CreateOrderRepository(db)
	orderSvc := service.CreateOrderService(orderRepo, midtransClient, kafkaReader, kafkaProducer, config, cb, productCommandGrpcClient, productQueryGrpcClient)
	controller.CreateOrderController(g, orderSvc)
	s, err := gocron.NewScheduler()
	if err != nil {
		panic(err)
	}

	// add a job to the scheduler
	_, err = s.NewJob(
		gocron.DurationJob(
			10*time.Second,
		),
		gocron.NewTask(
			orderSvc.RestoreExpiredPaymentItemStocks,
		),
	)
	if err != nil {
		panic(err)
	}

	s.Start()

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", config.ServicePort)))
}
