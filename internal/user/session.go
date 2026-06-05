package user

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SessionStore interface {
	// 创建用户会话，返回 Token
	// TTL 代表会话持续时间
	Create(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, error)

	Get(ctx context.Context, token string) (uuid.UUID, error)

	RefreshTTL(ctx context.Context, token string, ttl time.Duration) error

	Delete(ctx context.Context, token string) error
}
