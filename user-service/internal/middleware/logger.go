package middleware

import (
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func Logger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		requestID := uuid.New().String()

		ctx := c.Request().Context()

		logger := log.With().Str("request_id", requestID).Logger()
		ctx = logger.WithContext(ctx)

		c.SetRequest(c.Request().WithContext(ctx))

		err := next(c)

		latency := time.Since(start).Milliseconds()

		req := c.Request()
		res := c.Response()

		log.Ctx(c.Request().Context()).Info().
			Str("method", req.Method).
			Str("endpoint", req.URL.Path).
			Int("status", res.Status).
			Int64("latency", latency).
			Msg("Request processed")

		return err
	}
}
