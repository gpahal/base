package server

import (
	"fmt"
	"io"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/rs/zerolog"
)

type ServerOptions struct {
	Logger       *zerolog.Logger
	LoggerWriter io.Writer
}

func NewServer(opts ServerOptions) *echo.Echo {
	e := echo.New()
	e.Logger = newGommonLogger(opts.Logger, opts.LoggerWriter)
	e.Logger.SetLevel(log.ERROR)
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Skipper: middleware.DefaultSkipper,
		Timeout: 60 * time.Second,
	}))

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:       true,
		LogURI:          true,
		LogStatus:       true,
		LogError:        true,
		LogLatency:      true,
		LogResponseSize: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			evt := opts.Logger.Info()
			if v.Error != nil {
				evt = opts.Logger.Error()
			}
			evt.
				Int("status", v.Status).
				Err(v.Error).
				Str("latency", v.Latency.String()).
				Str("size", humanize.Bytes(uint64(v.ResponseSize))).
				Msg(fmt.Sprintf("%s %s", v.Method, v.URI))
			return nil
		},
	}))

	return e
}
