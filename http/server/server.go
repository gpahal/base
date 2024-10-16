package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/rs/zerolog"
)

type OnHttpErrorHandler func(c echo.Context, err *echo.HTTPError)

type Options struct {
	Validator    *validator.Validate
	Config       any
	LoggerWriter io.Writer
	Logger       *zerolog.Logger
	OnHttpError  OnHttpErrorHandler
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
	e.HTTPErrorHandler = newErrorHandler(e, opts.Logger, opts.OnHttpError)
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.RequestID())
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sctx := &Context{Context: c, Validator: opts.Validator, ConfigRaw: opts.Config, ServerLoggerWriter: opts.LoggerWriter, ServerLogger: newContextLogger(c, opts.Logger)}
			return next(sctx)
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
			sctx := c.(*Context)
			evt := sctx.ServerLogger.Info()
			if v.Error != nil {
				evt = sctx.ServerLogger.Error()
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

					opts.Logger.Error().Err(err).Msg("recovery handler")
					returnErr = err
				}
			}()
			return next(c)
		}
	})
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Skipper: middleware.DefaultSkipper,
		Timeout: 30 * time.Second,
	}))

	return e
}

type StartOptions struct {
	GracefulShutdownTimeout time.Duration
}

func Start(ctx context.Context, e *echo.Echo, port int) error {
	return StartWithOptions(ctx, e, port, StartOptions{
		GracefulShutdownTimeout: 10 * time.Second,
	})
}

func StartWithOptions(ctx context.Context, e *echo.Echo, port int, opts StartOptions) error {
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

func newErrorHandler(e *echo.Echo, logger *zerolog.Logger, onHttpError OnHttpErrorHandler) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		he, ok := err.(*echo.HTTPError)
		if ok {
			if he.Internal != nil {
				if herr, ok := he.Internal.(*echo.HTTPError); ok {
					he = herr
				}
			}
		} else {
			he = echo.NewHTTPError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		}

		if onHttpError != nil {
			onHttpError(c, he)
		}

		code := he.Code
		message := he.Message

		switch m := message.(type) {
		case string:
			if e.Debug {
				message = echo.Map{"error": m, "description": err.Error()}
			} else {
				message = echo.Map{"error": m}
			}
		case json.Marshaler:
			message = echo.Map{"error": m}
		case error:
			message = echo.Map{"error": m.Error()}
		}

		if c.Request().Method == http.MethodHead {
			err = c.NoContent(he.Code)
		} else {
			err = c.JSON(code, message)
		}

		logger = newContextLogger(c, logger)
		if err != nil {
			logger.Error().Err(err).Msg("error handler")
		}
	}
}

func newContextLogger(c echo.Context, logger *zerolog.Logger) *zerolog.Logger {
	requestId := c.Request().Header.Get(echo.HeaderXRequestID)
	loggerBuilder := logger.With().Str("method", c.Request().Method).Str("uri", c.Request().RequestURI)
	if requestId != "" {
		loggerBuilder = loggerBuilder.Str("request_id", requestId)
	}
	loggerStruct := loggerBuilder.Logger()
	return &loggerStruct
}
