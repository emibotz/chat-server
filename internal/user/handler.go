package user

import (
	"errors"
	"net/http"
	"strings"

	"github.com/emibotz/chat-server/internal/network"
	"github.com/emibotz/chat-server/pkg/errcode"
	"github.com/emibotz/chat-server/pkg/response"
	"github.com/labstack/echo/v5"
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

// 处理 WebSocket 请求，应该在所有处理器之前注册
// 在请求版本不兼容或用户未认证时自动中断
func (h *handler) HandleWS(c *network.Context) (bool, error) {

	if strings.Compare(c.Request.GetVersion(), network.APIVersion) != 0 {
		return true, errcode.SendError(c, errcode.IncompatibleVersion)
	}

	// [TODO] 认证

	return false, nil

}

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
