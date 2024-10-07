package controller

import (
	"github.com/alimikegami/point-of-sales/product-service/internal/dto"
	"github.com/alimikegami/point-of-sales/product-service/internal/service"
	pkgdto "github.com/alimikegami/point-of-sales/product-service/pkg/dto"
	"github.com/alimikegami/point-of-sales/product-service/pkg/response"
	"github.com/alimikegami/point-of-sales/product-service/pkg/utils"
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
	e.POST("/products", c.AddProduct, isLoggedIn)
	e.GET("/products", c.GetProducts)
	e.POST("/products/prices", c.GetProductsPrice)
	e.PUT("/products/quantity", c.UpdateProductsQuantity)
	e.DELETE("/products/:id", c.DeleteProduct)
	e.PUT("/products/:id", c.UpdateProduct)
	e.PUT("/products/:id/quantity", c.UpdateProductQuantity)
}

func (c *Controller) AddProduct(e echo.Context) error {
	_, userName, userID := utils.ExtractTokenUser(e)

	payload := dto.ProductRequest{}
	err := e.Bind(&payload)
	if err != nil {
		log.Error().Err(err).Str("component", "AddProduct").Msg("")
	}

	payload.UserID = userID
	payload.UserName = userName

	err = c.service.AddProduct(e.Request().Context(), payload)

	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "", nil)
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

func (c *Controller) UpdateProductsQuantity(e echo.Context) error {
	payload := dto.OrderRequest{}
	err := e.Bind(&payload)
	if err != nil {
		log.Error().Err(err).Str("component", "UpdateProductsQuantity").Msg("")
	}

	err = c.service.UpdateProductsQuantity(e.Request().Context(), payload)
	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "", nil)
}

func (c *Controller) DeleteProduct(e echo.Context) error {
	id := e.Param("id")
	err := c.service.DeleteProduct(e.Request().Context(), id)
	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "", nil)
}

func (c *Controller) UpdateProduct(e echo.Context) error {
	id := e.Param("id")
	payload := dto.ProductRequest{}
	err := e.Bind(&payload)
	if err != nil {
		log.Error().Err(err).Str("component", "UpdateProduct").Msg("")
	}

	payload.ID = id
	err = c.service.UpdateProduct(e.Request().Context(), payload)
	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "", nil)
}

func (c *Controller) UpdateProductQuantity(e echo.Context) error {
	id := e.Param("id")
	payload := dto.ProductQuantityRequest{}
	err := e.Bind(&payload)
	if err != nil {
		log.Error().Err(err).Str("component", "UpdateProduct").Msg("")
	}

	payload.ProductID = id
	err = c.service.UpdateProductQuantity(e.Request().Context(), payload)
	if err != nil {
		return response.WriteErrorResponse(e, err, nil)
	}

	return response.WriteSuccessResponse(e, "", nil)
}
