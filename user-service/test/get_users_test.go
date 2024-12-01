package test

import (
	"fmt"
	"net/http"

	"github.com/alimikegami/pos-microservices/user-service/pkg/utils"
	"github.com/stretchr/testify/require"
)

func (s *IntegrationTestSuite) TestGetUsers() {
	token, err := utils.CreateJWTToken(1, "test", "test", s.app.Config.JWTConfig.JWTSecret, s.app.Config.JWTConfig.JWTKid)
	require.NoError(s.T(), err)

	getUsersURL := fmt.Sprintf("http://localhost:%s/api/v1/users", s.app.Config.ServicePort)
	req, err := http.NewRequest("GET", getUsersURL, nil)
	require.NoError(s.T(), err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)
}
