package test

import (
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/alimikegami/pos-microservices/user-service/config"
	"github.com/alimikegami/pos-microservices/user-service/internal/app"
	posgres "github.com/alimikegami/pos-microservices/user-service/internal/infrastructure/database/postgres"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite
	app app.App
}

func setupTestConfig() *config.Config {
	config := config.CreateNewConfig()
	config.ServicePort = "8080"
	config.Environment = "test"
	return config
}

func (s *IntegrationTestSuite) initializeServer(config *config.Config) {
	db, err := posgres.GetDBInstance(config.PostgreSQLConfig.DBUsername, config.PostgreSQLConfig.DBPassword,
		config.PostgreSQLConfig.DBHost, config.PostgreSQLConfig.DBPort, config.PostgreSQLConfig.DBName, config.Environment)
	if err != nil {
		log.Fatal(err.Error())
	}

	s.app.DB = db
	s.app.Start()
}

func checkServerHealth(config *config.Config) {
	pingURL := fmt.Sprintf("http://localhost:%s/api/v1/ping", config.ServicePort)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			log.Fatal("Timeout waiting for server to start")
		case <-ticker.C:
			resp, err := http.Get(pingURL)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return
			}
		}
	}
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.app.Config = setupTestConfig()

	s.initializeServer(s.app.Config)

	checkServerHealth(s.app.Config)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	err := s.app.StopServer()

	s.Require().NoError(err)
}

// Add this function to run the test suite
func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
