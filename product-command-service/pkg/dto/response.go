package dto

import (
	"net/http"

	"github.com/alimikegami/point-of-sales/product-command-service/pkg/errs"
	"github.com/labstack/echo/v4"
)

type Pagination struct {
	Previous *string     `json:"previous"`
	Next     *string     `json:"next"`
	Records  interface{} `json:"records"`
}

type PaginationMetadata struct {
	TotalCount uint64 `json:"total_count"`
	Page       uint64 `json:"page"`
	Limit      int    `json:"limit"`
}

type PaginationResponse struct {
	Metadata PaginationMetadata `json:"_metadata"`
	Records  interface{}        `json:"records"`
}

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
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

func WriteSuccessResponse(c echo.Context, message string) error {
	resp := SuccessResponse{}
	resp.Status = "success"
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
