package main

import (
	"net/http"
	"os"

	"github.com/alimikegami/point-of-sales/order-service/config"
	"github.com/alimikegami/point-of-sales/order-service/internal/controller"
	"github.com/alimikegami/point-of-sales/order-service/internal/infrastructure/database/postgres"
	paymentgateway "github.com/alimikegami/point-of-sales/order-service/internal/infrastructure/payment-gateway"
	"github.com/alimikegami/point-of-sales/order-service/internal/repository"
	"github.com/alimikegami/point-of-sales/order-service/internal/service"
	"github.com/alimikegami/point-of-sales/order-service/pkg/dto"
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

	db, err := postgres.GetDBInstance(config)
	if err != nil {
		panic(err)
	}

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

	e := echo.New()
	g := e.Group("/api/v1")

	g.GET("/ping", func(c echo.Context) error {
		return dto.WriteSuccessResponse(c, "Hello, World!")
	})

	orderRepo := repository.CreateOrderRepository(db)
	orderSvc := service.CreateOrderService(orderRepo, midtransClient)
	controller.CreateOrderController(g, orderSvc, IsLoggedIn)

	e.Logger.Fatal(e.Start(":8080"))
}
