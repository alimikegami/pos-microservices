package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/alimikegami/pos-microservices/user-service/internal/dto"
	"github.com/stretchr/testify/require"
)

func (s *IntegrationTestSuite) TestUpdateUser() {
	payload := dto.UserRequest{
		ID:   1,
		Name: "Hello",
	}

	jsonPayload, err := json.Marshal(payload)
	require.NoError(s.T(), err)

	createUserURL := fmt.Sprintf("http://localhost:%s/api/v1/users/%d", s.app.Config.ServicePort, payload.ID)
	req, err := http.NewRequest("PUT", createUserURL, bytes.NewBuffer(jsonPayload))
	require.NoError(s.T(), err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)
}
