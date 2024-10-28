package controller

import (
	"github.com/alimikegami/point-of-sales/order-service/internal/dto"
	"github.com/alimikegami/point-of-sales/order-service/internal/service"
	pkgdto "github.com/alimikegami/point-of-sales/order-service/pkg/dto"
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
	e.GET("/orders", c.GetOrders)
}

func (c *Controller) AddOrder(e echo.Context) error {
	// _, userName, userID := utils.ExtractTokenUser(e)
	log.Info().Msg("add order req start")

	payload := dto.OrderRequest{}
	err := e.Bind(&payload)
	if err != nil {
		log.Error().Err(err).Str("component", "AddOrder").Msg("")
	}

	resp, err := c.service.AddOrder(e.Request().Context(), payload)
	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "", resp)
}

func (c *Controller) MidtransPaymentWebhook(e echo.Context) error {
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

func (c *Controller) GetOrders(e echo.Context) error {
	filter := pkgdto.Filter{}
	err := e.Bind(&filter)
	if err != nil {
		log.Error().Err(err).Str("component", "AddProduct").Msg("")
	}

	responsePayload, err := c.service.GetOrders(e.Request().Context(), filter)

	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "successfuly retrieved orders record", responsePayload)
}
