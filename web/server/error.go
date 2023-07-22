package server

import (
	"fmt"
	"net/http"
)

// Errors
var (
	ErrBadRequest                  = NewStatusCodeHTTPError(http.StatusBadRequest)
	ErrUnauthorized                = NewStatusCodeHTTPError(http.StatusUnauthorized)
	ErrForbidden                   = NewStatusCodeHTTPError(http.StatusForbidden)
	ErrNotFound                    = NewStatusCodeHTTPError(http.StatusNotFound)
	ErrMethodNotAllowed            = NewStatusCodeHTTPError(http.StatusMethodNotAllowed)
	ErrRequestTimeout              = NewStatusCodeHTTPError(http.StatusRequestTimeout)
	ErrStatusRequestEntityTooLarge = NewStatusCodeHTTPError(http.StatusRequestEntityTooLarge)
	ErrUnsupportedMediaType        = NewStatusCodeHTTPError(http.StatusUnsupportedMediaType)
	ErrTooManyRequests             = NewStatusCodeHTTPError(http.StatusTooManyRequests)

	ErrInternalServerError         = NewStatusCodeHTTPError(http.StatusInternalServerError)
	ErrBadGateway                  = NewStatusCodeHTTPError(http.StatusBadGateway)
	ErrServiceUnavailable          = NewStatusCodeHTTPError(http.StatusServiceUnavailable)

	ErrRendererNotRegistered       = NewMsgHTTPError("renderer not registered")
	ErrCookieNotFound              = NewMsgHTTPError("cookie not found")
)

const (
	ErrCodeUnknown = "UNKNOWN"
	ErrMsgUnknown  = "unknown error"
)

type ReportTarget uint8

const (
	LogReportTarget ReportTarget = iota
	UserReportTarget
)

var (
	OnlyLogReportTargets = []ReportTarget{LogReportTarget}
	OnlyUserReportTargets = []ReportTarget{UserReportTarget}
	AllReportTargets = []ReportTarget{LogReportTarget, UserReportTarget}
)

type HTTPError interface {
	error
	Code() string
	StatusCode() int
	ReportTargets() []ReportTarget
}

type httpError struct {
	statusCode int
	msg        string
}

func newHTTPError(statusCode int, msg string) HTTPError {
	return &httpError{statusCode: statusCode, msg: msg}
}

func NewStatusCodeHTTPError(statusCode int) HTTPError {
	return newHTTPError(statusCode, http.StatusText(statusCode))
}

func NewMsgHTTPError(msg string) HTTPError {
	return newHTTPError(http.StatusInternalServerError, msg)
}

func (e *httpError) Error() string {
	return e.msg
}

func (e *httpError) Code() string {
	return fmt.Sprintf("HTTP%d", e.statusCode)
}

func (e *httpError) StatusCode() int {
	return e.statusCode
}

func (e *httpError) ReportTargets() []ReportTarget {
	return AllReportTargets
}

type Error struct {
	ErrorCode string
	StatusCode int
	Message string
}

type ErrorHandler interface {
	HandleError(c *C, err *Error)
}

type defaultErrorHandler struct {}

func (eh *defaultErrorHandler) HandleError(c *C, err *Error) {
	_ = c.JSON(err.StatusCode, err)
}

func handleError(handler ErrorHandler, c *C, err error) {
	herr, ok := err.(HTTPError)
	if !ok {
		herr = NewMsgHTTPError(err.Error())
	}
	handleHTTPError(handler, c, herr)
}

func handleHTTPError(handler ErrorHandler, c *C, herr HTTPError) {
	if containsReportTarget(herr, LogReportTarget) {
		logger := c.Logger()
		logger.Error().Err(herr).Msg("")
	}

	err := &Error{}
	if containsReportTarget(herr, UserReportTarget) {
		err.ErrorCode = herr.Code()
		err.StatusCode = herr.StatusCode()
		err.Message = herr.Error()
	} else {
		err.ErrorCode = ErrCodeUnknown
		err.StatusCode = http.StatusInternalServerError
		err.Message = ErrMsgUnknown
	}

	if handler == nil {
		handler = &defaultErrorHandler{}
	}
	handler.HandleError(c, err)
}

func containsReportTarget(herr HTTPError, rt ReportTarget) bool {
	for _, el := range herr.ReportTargets() {
		if el == rt {
			return true
		}
	}
	return false
}
