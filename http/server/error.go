package server

import (
	"github.com/labstack/echo/v4"
)

func NewHttpError(code int, msgOrErr interface{}) *echo.HTTPError {
	internal, ok := msgOrErr.(error)
	if !ok {
		internal = nil
	}
	return &echo.HTTPError{Code: code, Message: internal.Error(), Internal: internal}
}

func NewHttpErrorWithInternal(code int, msgOrErr interface{}, internal error) *echo.HTTPError {
	return &echo.HTTPError{Code: code, Message: msgOrErr, Internal: internal}
}
