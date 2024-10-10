package server

import (
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

type Context struct {
	echo.Context
	Validator    *validator.Validate
	ConfigRaw    any
	ServerLogger *zerolog.Logger
}

func GetContext(c echo.Context) *Context {
	return c.(*Context)
}
