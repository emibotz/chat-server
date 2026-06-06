package middleware

import (
	"errors"
	"strings"

	"github.com/emibotz/chat-server/internal/user"
	"github.com/emibotz/chat-server/pkg/key"
	"github.com/emibotz/chat-server/pkg/response"
	"github.com/labstack/echo/v5"
)

func Auth(userService *user.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// 获取认证头
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return response.Unauthorized(c)
			}

			// 解析认证头
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return response.Unauthorized(c)
			}

			// 验证 Token 并获取用户 ID
			token := parts[1]
			id, err := userService.VerifyToken(c.Request().Context(), token)
			if err != nil {
				if errors.Is(err, user.ErrTokenUnauthorized) {
					return response.Unauthorized(c)
				}

				return response.InternalServerError(c, err)
			}

			c.Set(key.ContextUserID, id)

			return next(c)
		}
	}
}
