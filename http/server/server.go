package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/rs/zerolog"
)

type Options struct {
	Validator    *validator.Validate
	LoggerWriter io.Writer
	Logger       *zerolog.Logger
	Config       any
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
	e.Use(middleware.RequestID())
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestId := c.Request().Header.Get(echo.HeaderXRequestID)
			loggerBuilder := opts.Logger.With().Str("method", c.Request().Method).Str("uri", c.Request().RequestURI)
			if requestId != "" {
				loggerBuilder = loggerBuilder.Str("request_id", requestId)
			}
			logger := loggerBuilder.Logger()
			actx := &Context{Context: c, Validator: opts.Validator, ConfigRaw: opts.Config, ServerLoggerWriter: opts.LoggerWriter, ServerLogger: &logger}
			return next(actx)
		}
	})
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:       true,
		LogURI:          true,
		LogStatus:       true,
		LogError:        true,
		LogLatency:      true,
		LogResponseSize: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			ac := c.(*Context)
			evt := ac.ServerLogger.Info()
			if v.Error != nil {
				evt = ac.ServerLogger.Error()
			}

			evt = evt.Int("status", v.Status).Err(v.Error).Str("latency", v.Latency.String())
			if v.RequestID != "" {
				evt = evt.Str("request_id", v.RequestID)
			}
			if v.ResponseSize > 0 {
				evt = evt.Str("size", humanize.Bytes(uint64(v.ResponseSize)))
			}

			evt.Msg("request")
			return nil
		},
	}))
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (returnErr error) {
			defer func() {
				if r := recover(); r != nil {
					if r == http.ErrAbortHandler {
						panic(r)
					}

					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}

					stack := make([]byte, 4<<10)
					length := runtime.Stack(stack, false)
					stack = stack[:length]

					opts.Logger.Error().Msgf("panic: %v\n%s\n", err, stack)

					if he, ok := err.(*echo.HTTPError); ok {
						returnErr = he
					} else {
						returnErr = echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("panic recovered: %v", err))
					}
				}
			}()
			return next(c)
		}
	})
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Skipper: middleware.DefaultSkipper,
		Timeout: 60 * time.Second,
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
