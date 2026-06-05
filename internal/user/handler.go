package user

import (
	"errors"
	"net/http"

	"github.com/emibotz/chat-server/pkg/response"
	"github.com/labstack/echo/v5"
)

var (
	ContextKeyUserID = "user_id"
)

type handler struct {
	service *Service
}

func NewHandler(
	service *Service,
) *handler {
	return &handler{
		service: service,
	}
}

// [TODO] 请求频率限制器

func (h *handler) Register(c *echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c)
	}

	ctx := c.Request().Context()

	token, err := h.service.Register(ctx, req.Username, req.Password)
	if err != nil {
		// 处理用户名或密码格式错误
		if errors.Is(err, ErrInvalidUsername) || errors.Is(err, ErrInvalidPassword) {
			return response.HTTPFail(c, http.StatusBadRequest, -1, err)
		}

		return response.HTTPFail(c, http.StatusInternalServerError, -1, err)

	}

	return response.HTTPSuccess(c, http.StatusCreated, 0, "user registered", registerResponse{Token: token})
}

func (h *handler) Login(c *echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c)
	}

	ctx := c.Request().Context()

	token, err := h.service.Login(ctx, req.Username, req.Password)
	if err != nil {
		// 处理用户名或密码格式错误
		if errors.Is(err, ErrInvalidUsername) || errors.Is(err, ErrInvalidPassword) {
			return response.HTTPFail(c, http.StatusBadRequest, -1, err)
		}

		return response.HTTPFail(c, http.StatusInternalServerError, -1, err)
	}

	return response.HTTPSuccess(c, http.StatusOK, 0, "user logged in", loginResponse{Token: token})
}
