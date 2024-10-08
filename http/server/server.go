package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
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

type StartOptions struct {
	GracefulShutdownTimeout time.Duration
}

func Start(e *echo.Echo, port int) error {
	return StartWithOptions(e, port, StartOptions{
		GracefulShutdownTimeout: 10 * time.Second,
	})
}

func StartWithOptions(e *echo.Echo, port int, opts StartOptions) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), opts.GracefulShutdownTimeout)
		defer cancel()

		if err := e.Shutdown(ctx); err != nil {
			e.Logger.Fatal(err)
		}
	}()

	if err := e.Start(fmt.Sprintf(":%d", port)); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

type Router interface {
	Group(prefix string, m ...echo.MiddlewareFunc) *echo.Group
	Use(middleware ...echo.MiddlewareFunc)
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	Any(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) []*echo.Route
	Match(methods []string, path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) []*echo.Route
	Add(method, path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) *echo.Route
}

func AddSubRouter(r Router, path string, subRouterFn func(r Router)) {
	sr := r.Group(path)
	subRouterFn(sr)
}
