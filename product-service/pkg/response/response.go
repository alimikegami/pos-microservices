package response

import (
	"net/http"

	"github.com/alimikegami/point-of-sales/product-service/pkg/errs"
	"github.com/labstack/echo/v4"
)

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ValidationError struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
}

type ErrorResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors"`
}

func WriteSuccessResponse(c echo.Context, message string, data interface{}) error {
	resp := SuccessResponse{}
	resp.Status = "success"
	resp.Data = data
	resp.Message = message

	return c.JSON(http.StatusOK, resp)
}

func WriteErrorResponse(c echo.Context, err error, errors interface{}) error {
	statusCode := errs.GetErrorStatusCode(err)
	resp := ErrorResponse{}
	resp.Status = "error"
	resp.Message = err.Error()
	resp.Errors = errors

	return c.JSON(statusCode, resp)
}

type DataWithPaginationsResponse struct {
	Data       interface{} `json:"data,omitempty"`
	Pagination interface{} `json:"pagination,omitempty"`
}
