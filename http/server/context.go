package server

import (
	"io"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

type Context struct {
	echo.Context
	Validator          *validator.Validate
	ConfigRaw          any
	ServerLoggerWriter io.Writer
	ServerLogger       *zerolog.Logger
}

func GetContext(c echo.Context) *Context {
	return c.(*Context)
}

func (c *Context) Logger() echo.Logger {
	return newGommonLogger(c.ServerLogger, c.ServerLoggerWriter)
}
