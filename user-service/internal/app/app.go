package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alimikegami/pos-microservices/user-service/config"
	"github.com/alimikegami/pos-microservices/user-service/internal/controller"
	"github.com/alimikegami/pos-microservices/user-service/internal/infrastructure/tracing"
	"github.com/alimikegami/pos-microservices/user-service/internal/repository"
	"github.com/alimikegami/pos-microservices/user-service/internal/service"
	"github.com/alimikegami/pos-microservices/user-service/pkg/dto"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type App struct {
	DB     *sqlx.DB
	Config *config.Config
	Server *echo.Echo
}

func (app *App) Start() {
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).With().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = logger

	e := echo.New()
	traceProvider, err := tracing.InitTracing(app.Config.TracingConfig.CollectorHost)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize tracing")
	}

	defer func() {
		if err := traceProvider.Shutdown(context.Background()); err != nil {
			logger.Error().Err(err).Msg("Failed to shutdown tracing")
		}
	}()

	tracer := traceProvider.Tracer("user-service")

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// span creation and naming
			ctx, span := tracer.Start(c.Request().Context(), fmt.Sprintf("[%s] %s", c.Request().Method, c.Path()))
			defer span.End()

			// add the context to the request
			req := c.Request()
			c.SetRequest(req.WithContext(ctx))

			return next(c)
		}
	})

	g := e.Group("/api/v1")

	g.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:      true,
		LogStatus:   true,
		LogLatency:  true,
		LogRemoteIP: true,
		LogMethod:   true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info().
				Str("method", v.Method).
				Str("URI", v.URI).
				Int("status", v.Status).
				Int64("latency", v.Latency.Microseconds()).
				Str("remote IP", v.RemoteIP).
				Msg("Request")

			return nil
		},
	}))

	repo := repository.CreateNewRepository(app.DB)
	svc := service.CreateNewService(repo, *app.Config)
	controller.CreateController(g, svc)

	g.GET("/ping", func(c echo.Context) error {
		return dto.WriteSuccessResponse(c, "Hello, World!")
	})

	go func() {
		if err := e.Start(fmt.Sprintf(":%s", app.Config.ServicePort)); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("Server shutdown error")
		}
	}()

	app.Server = e
}
func (app *App) StopServer() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return app.Server.Shutdown(ctx)
}
