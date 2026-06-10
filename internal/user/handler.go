package user

import (
	"errors"
	"net/http"
	"strings"

	"github.com/emibotz/chat-server/internal/network"
	"github.com/emibotz/chat-server/pkg/errcode"
	"github.com/emibotz/chat-server/pkg/key"
	"github.com/emibotz/chat-server/pkg/logger"
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

// 在请求版本不兼容或用户未认证时提前中断处理。
// 在用户认证成功时，将用户指针注入上下文。
// 这个处理器应该在所有处理器之前注册。
func (h *handler) HandleWS(c *network.Context) (bool, error) {

	// 如果版本不匹配，返回版本不兼容
	if strings.Compare(c.Request.GetVersion(), network.APIVersion) != 0 {
		return true, errcode.SendError(c, errcode.IncompatibleVersion)
	}

	// 通过客户端中维护的用户 ID 获取用户
	user, err := h.service.GetUserByID(c, c.Client.GetUserID())
	if err != nil {
		logger.Error("get user by id failed", err)
		return true, errcode.SendInternalError(c)
	}

	// 如果没有用户记录，返回未认证
	if user == nil {
		return true, errcode.SendUnauthorized(c)
	}

	// 将用户注入到上下文中
	c.Set(key.ContextUser, user)

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
