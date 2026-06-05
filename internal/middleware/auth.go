package middleware

import (
	"errors"
	"strings"

	"github.com/emibotz/chat-server/internal/user"
	"github.com/emibotz/chat-server/pkg/response"
	"github.com/labstack/echo/v5"
)

func Auth(service user.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// 获取请求上下文
			ctx := c.Request().Context()

			// 获取认证头
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return response.Unauthorized(c)
			}

			// 从认证头中提取 Token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return response.Unauthorized(c)
			}
			token := parts[1]

			// 验证 Token 并获取用户 ID
			id, err := service.VerifyToken(ctx, token)
			if err != nil {
				if errors.Is(err, user.ErrTokenUnauthorized) {
					return response.Unauthorized(c)
				}

				return response.InternalServerError(c, err)
			}

			// 将用户 ID 注入到上下文中
			c.Set(user.ContextKeyUserID, id)

			return next(c)
		}
	}
}
