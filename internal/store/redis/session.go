package redis

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// 生成安全的 32 字节 base64 加密字符串
func generateToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawStdEncoding.EncodeToString(bytes), nil
}

func getKey(token string) string {
	return "session:" + token
}

type sessions struct {
	redis *redis.Client
}

func (s *sessions) Create(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generate token failed: %w", err)
	}

	key := getKey(token)
	if err := s.redis.Set(ctx, key, userID.String(), ttl).Err(); err != nil {
		return "", fmt.Errorf("redis set token to userID failed: %w", err)
	}

	return token, nil
}

func (s *sessions) Get(ctx context.Context, token string) (uuid.UUID, error) {
	key := getKey(token)

	rawID, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return uuid.Nil, fmt.Errorf("redis get userID by token failed: %w", err)
	}

	userID, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse uuid failed: %w", err)
	}

	return userID, nil
}

func (s *sessions) RefreshTTL(ctx context.Context, token string, ttl time.Duration) error {
	key := getKey(token)

	ok, err := s.redis.Expire(ctx, key, ttl).Result()

	if err != nil {
		return fmt.Errorf("redis set expire failed: %w", err)
	}

	if !ok {
		return fmt.Errorf("session `%s` does not exist.", key)
	}

	return nil
}

func (s *sessions) Delete(ctx context.Context, token string) error {
	key := getKey(token)

	if err := s.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del by token failed: %w", err)
	}

	return nil
}
