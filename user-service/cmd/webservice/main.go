package main

import (
	"os"

	"github.com/alimikegami/e-commerce/user-service/config"
	"github.com/alimikegami/e-commerce/user-service/internal/controller"
	postgresDriver "github.com/alimikegami/e-commerce/user-service/internal/infrastructure/database/postgres"
	"github.com/alimikegami/e-commerce/user-service/internal/infrastructure/message-queue/kafka"
	"github.com/alimikegami/e-commerce/user-service/internal/repository"
	"github.com/alimikegami/e-commerce/user-service/internal/service"
	"github.com/alimikegami/e-commerce/user-service/pkg/dto"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).With().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = logger

	config := config.CreateNewConfig()
	db, err := postgresDriver.GetDBInstance(config.PostgreSQLConfig.DBUsername, config.PostgreSQLConfig.DBPassword, config.PostgreSQLConfig.DBHost, config.PostgreSQLConfig.DBPort, config.PostgreSQLConfig.DBName)
	if err != nil {
		panic(err)
	}

	e := echo.New()
	g := e.Group("/api/v1")

	kafkaProducer := kafka.CreateKafkaProducer(config)

	repo := repository.CreateNewRepository(db)
	svc := service.CreateNewService(repo, *config, kafkaProducer)
	controller.CreateController(g, svc)

	g.GET("/ping", func(c echo.Context) error {
		return dto.WriteSuccessResponse(c, "Hello, World!")
	})

	e.Logger.Fatal(e.Start(":8080"))
}
