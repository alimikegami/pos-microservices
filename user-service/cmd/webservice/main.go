package main

import (
	"github.com/alimikegami/pos-microservices/user-service/config"
	"github.com/alimikegami/pos-microservices/user-service/internal/app"

	postgresDriver "github.com/alimikegami/pos-microservices/user-service/internal/infrastructure/database/postgres"
)

func main() {
	config := config.CreateNewConfig()
	db, err := postgresDriver.GetDBInstance(config.PostgreSQLConfig.DBUsername, config.PostgreSQLConfig.DBPassword, config.PostgreSQLConfig.DBHost, config.PostgreSQLConfig.DBPort, config.PostgreSQLConfig.DBName, config.Environment)
	if err != nil {
		panic(err)
	}

	server := app.App{
		DB:     db,
		Config: config,
	}

	server.Start()
}
