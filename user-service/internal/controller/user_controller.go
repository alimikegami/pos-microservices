package controller

import (
	"strconv"

	"github.com/alimikegami/e-commerce/user-service/internal/dto"
	"github.com/alimikegami/e-commerce/user-service/internal/service"
	pkgdto "github.com/alimikegami/e-commerce/user-service/pkg/dto"
	"github.com/alimikegami/e-commerce/user-service/pkg/errs"
	"github.com/alimikegami/e-commerce/user-service/pkg/response"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type Controller struct {
	service service.UserService
}

func CreateController(e *echo.Group, service service.UserService) {
	uc := Controller{
		service: service,
	}
	e.POST("/users/register", uc.AddUser)
	e.POST("/users/login", uc.Login)
	e.PUT("/users/:id", uc.UpdateUser)
	e.GET("/users", uc.GetUsers)
}

func (c *Controller) AddUser(e echo.Context) error {
	payload := dto.UserRequest{}
	err := e.Bind(&payload)
	if err != nil {
		log.Error().Err(err).Str("component", "AddUser").Msg("")
	}

	err = c.service.AddUser(e.Request().Context(), payload)

	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "", nil)
}

func (c *Controller) Login(e echo.Context) error {
	payload := dto.UserRequest{}
	err := e.Bind(&payload)
	if err != nil {
		log.Error().Err(err).Str("component", "Login").Msg("")
	}

	respPayload, err := c.service.Login(e.Request().Context(), payload)

	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "", respPayload)
}

func (c *Controller) UpdateUser(e echo.Context) error {
	id := e.Param("id")
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return response.WriteErrorResponse(e, errs.ErrClient, nil)
	}

	payload := dto.UserRequest{}
	err = e.Bind(&payload)
	if err != nil {
		log.Error().Err(err).Str("component", "UpdateUser").Msg("")
	}

	payload.ID = idInt
	err = c.service.UpdateUser(e.Request().Context(), payload)
	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "", nil)
}

func (c *Controller) GetUsers(e echo.Context) error {
	payload := pkgdto.Filter{}
	err := e.Bind(&payload)
	if err != nil {
		log.Error().Err(err).Str("component", "GetUsers").Msg("")
	}

	resp, err := c.service.GetUsers(e.Request().Context(), payload)
	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "", resp)
}
