package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/alimikegami/point-of-sales/product-command-service/config"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/controller"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/infrastructure/database/mongodb"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/infrastructure/message-queue/kafka"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/repository"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/service"
	"github.com/alimikegami/point-of-sales/product-command-service/pkg/dto"
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
	db, err := mongodb.ConnectToMongoDB(fmt.Sprintf("mongodb://%s:%s", config.MongoDBConfig.DBHost, config.MongoDBConfig.DBPort))
	if err != nil {
		panic(err)
	}

	kafkaProducer := kafka.CreateKafkaProducer(config)
	kafkaReader := kafka.CreateKafkaReader(config)

	defer db.Client().Disconnect(context.Background())

	e := echo.New()
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

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", config.ServicePort)))
}
