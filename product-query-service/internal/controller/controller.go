package controller

import (
	"github.com/alimikegami/point-of-sales/product-query-service/internal/service"
	pkgdto "github.com/alimikegami/point-of-sales/product-query-service/pkg/dto"
	"github.com/alimikegami/point-of-sales/product-query-service/pkg/response"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type Controller struct {
	service service.ProductService
}

func CreateProductController(e *echo.Group, service service.ProductService, isLoggedIn echo.MiddlewareFunc) {
	c := Controller{
		service: service,
	}
	e.GET("/products", c.GetProducts)
	e.POST("/products/prices", c.GetProductsPrice)

}

func (c *Controller) GetProducts(e echo.Context) error {
	filter := pkgdto.Filter{}
	err := e.Bind(&filter)
	if err != nil {
		log.Error().Err(err).Str("component", "AddProduct").Msg("")
	}

	responsePayload, err := c.service.GetProducts(e.Request().Context(), filter)

	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "successfuly retrieved products record", responsePayload)
}

func (c *Controller) GetProductsPrice(e echo.Context) error {
	filter := pkgdto.Filter{}
	err := e.Bind(&filter)
	if err != nil {
		log.Error().Err(err).Str("component", "GetProductsPrice").Msg("")
	}

	responsePayload, err := c.service.GetProducts(e.Request().Context(), filter)

	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "successfuly retrieved products record", responsePayload)
}
