package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/alimikegami/pos-microservices/user-service/internal/dto"
	"github.com/labstack/echo/v4"
)

func (s *IntegrationTestSuite) Test_UserLogin() {
	type TestCase struct {
		Name           string
		Request        dto.UserRequest
		ExpectedStatus int
		AssertResponse func(s *IntegrationTestSuite, resp *http.Response)
	}

	testCases := []TestCase{
		{
			Name: "Valid request",
			Request: dto.UserRequest{
				Email:    "admin@example.com",
				Password: "123456",
			},
			ExpectedStatus: http.StatusOK,
			AssertResponse: func(s *IntegrationTestSuite, resp *http.Response) {
			},
		},
		{
			Name: "Wrong password",
			Request: dto.UserRequest{
				Email:    "test@gmail.com",
				Password: "1234",
			},
			ExpectedStatus: http.StatusUnauthorized,
			AssertResponse: func(s *IntegrationTestSuite, resp *http.Response) {
			},
		},
		{
			Name: "Invalid email",
			Request: dto.UserRequest{
				Email:    "test",
				Password: "123456",
			},
			ExpectedStatus: http.StatusNotFound,
			AssertResponse: func(s *IntegrationTestSuite, resp *http.Response) {
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			reqBody, err := json.Marshal(tc.Request)
			s.NoError(err)

			req, err := http.NewRequest(http.MethodPost,
				fmt.Sprintf("http://localhost:%s/api/v1/users/login", s.app.Config.ServicePort),
				bytes.NewBuffer(reqBody),
			)
			s.NoError(err)

			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

			client := http.Client{}
			resp, err := client.Do(req)
			s.NoError(err)

			s.Equal(tc.ExpectedStatus, resp.StatusCode)

			if tc.AssertResponse != nil {
				tc.AssertResponse(s, resp)
			}
		})
	}
}
