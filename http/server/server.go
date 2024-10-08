package server

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/rs/zerolog"
)

type Options struct {
	LoggerWriter io.Writer
	Logger       *zerolog.Logger
}

func New() *echo.Echo {
	return NewWithOptions(Options{})
}

func NewWithOptions(opts Options) *echo.Echo {
	if opts.LoggerWriter == nil {
		opts.LoggerWriter = os.Stdout
	}
	if opts.Logger == nil {
		opts.Logger = newLogger(opts.LoggerWriter)
	}

	e := echo.New()
	e.HideBanner = true
	e.Logger = newGommonLogger(opts.Logger, opts.LoggerWriter)
	e.Logger.SetLevel(log.INFO)
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize: 4 << 10,
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			opts.Logger.Error().Err(err).Str("stack", string(stack)).Msg("panic recovered")
			return nil
		},
	}))
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

			evt = evt.Int("status", v.Status).Err(v.Error).Str("latency", v.Latency.String())
			if v.RequestID != "" {
				evt = evt.Str("request_id", v.RequestID)
			}
			if v.ResponseSize > 0 {
				evt = evt.Str("size", humanize.Bytes(uint64(v.ResponseSize)))
			}

			evt.Msg(fmt.Sprintf("%s %s", v.Method, v.URI))
			return nil
		},
	}))

	return e
}

func AddGroup(e *echo.Echo, path string, routerFn func(g *echo.Group)) {
	g := e.Group(path)
	routerFn(g)
}
