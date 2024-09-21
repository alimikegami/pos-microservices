package controller

import (
	"fmt"

	"github.com/alimikegami/point-of-sales/order-service/internal/dto"
	"github.com/alimikegami/point-of-sales/order-service/internal/service"
	"github.com/alimikegami/point-of-sales/order-service/pkg/response"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type Controller struct {
	service service.OrderService
}

func CreateOrderController(e *echo.Group, service service.OrderService, isLoggedIn echo.MiddlewareFunc) {
	c := Controller{
		service: service,
	}

	e.POST("/orders", c.AddOrder)
	e.POST("/orders/payments/notifications", c.MidtransPaymentWebhook)

}

func (c *Controller) AddOrder(e echo.Context) error {
	// _, userName, userID := utils.ExtractTokenUser(e)

	payload := dto.OrderRequest{}
	err := e.Bind(&payload)
	if err != nil {
		log.Error().Err(err).Str("component", "AddOrder").Msg("")
	}

	err = c.service.AddOrder(e.Request().Context(), payload)

	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "", nil)
}

func (c *Controller) MidtransPaymentWebhook(e echo.Context) error {
	fmt.Println("here")
	payload := dto.PaymentNotification{}
	err := e.Bind(&payload)
	if err != nil {
		log.Error().Err(err).Str("component", "MidtransPaymentWebhook").Msg("")
	}

	err = c.service.MidtransPaymentWebhook(e.Request().Context(), payload)

	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "", nil)
}