package response

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
)

// 这个包是为 HTTP 请求准备的

type HTTPResponse struct {
	Code    int32  `json:"code" form:"code"`
	Message string `json:"message" form:"message"`
	Data    any    `json:"data"`
}

func HTTPSuccess(c *echo.Context, httpStatus int, code int32, message string, data any) error {
	return c.JSON(httpStatus, HTTPResponse{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

func HTTPFail(c *echo.Context, httpStatus int, code int32, err error) error {
	return c.JSON(httpStatus, HTTPResponse{
		Code:    code,
		Message: err.Error(),
		Data:    nil,
	})
}

var (
	ErrUnauthorized = fmt.Errorf("unauthorized.")
	ErrBadRequest   = fmt.Errorf("bad request.")
)

func Unauthorized(c *echo.Context) error {
	return HTTPFail(c, http.StatusUnauthorized, -1, ErrUnauthorized)
}

func BadRequest(c *echo.Context) error {
	return HTTPFail(c, http.StatusBadRequest, -1, ErrBadRequest)
}

func InternalServerError(c *echo.Context, err error) error {
	return HTTPFail(c, http.StatusInternalServerError, -1, err)
}
