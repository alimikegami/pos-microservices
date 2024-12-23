package main

import (
	"context"
	"fmt"
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
	"github.com/go-co-op/gocron/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).With().Logger()
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

	cb := circuitbreaker.CreateCircuitBreaker("order-service")

	orderRepo := repository.CreateOrderRepository(db)
	orderSvc := service.CreateOrderService(orderRepo, midtransClient, kafkaReader, kafkaProducer, config, cb)
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
