package valkey

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
)

// 生成安全的 32 字节 base64 加密字符串
func generateToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawStdEncoding.EncodeToString(bytes), nil
}

var sessionPrefix = "session:"

func getKey(token string) string {
	return sessionPrefix + token
}

type sessions struct {
	client valkey.Client
}

func (s *sessions) Create(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generate token failed: %w", err)
	}

	key := getKey(token)
	if err := s.client.Do(ctx, s.client.B().Set().Key(key).Value(userID.String()).Ex(ttl).Build()).Error(); err != nil {
		return "", fmt.Errorf("valkey set token failed: %w", err)
	}

	return token, nil
}

func (s *sessions) Get(ctx context.Context, token string) (uuid.UUID, error) {

	key := getKey(token)

	result := s.client.Do(ctx, s.client.B().Get().Key(key).Build())
	rawID, err := result.ToString()
	if err != nil {
		return uuid.Nil, fmt.Errorf("valkey get userID by token failed: %w", err)
	}

	userID, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("uuid parse failed: %w", err)
	}

	return userID, nil
}

func (s *sessions) RefreshTTL(ctx context.Context, token string, ttl time.Duration) error {

	key := getKey(token)

	n, err := s.client.Do(ctx, s.client.B().Expire().Key(key).Seconds(int64(ttl.Seconds())).Build()).AsInt64()
	if err != nil {
		return fmt.Errorf("valkey expire failed: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("session `%s` does not exist.", token)
	}

	return nil
}

func (s *sessions) Delete(ctx context.Context, token string) error {

	key := getKey(token)

	n, err := s.client.Do(ctx, s.client.B().Del().Key(key).Build()).AsInt64()
	if err != nil {
		return fmt.Errorf("valkey del failed: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("session `%s` does not exist.", token)
	}

	return nil
}

func (s *sessions) DeleteAllByUserID(ctx context.Context, userID uuid.UUID) error {

	id := userID.String()
	keysToDelete := make([]string, 0)

scanLoop:
	for {

		scanEntry, err := s.client.Do(ctx, s.client.B().Scan().Cursor(0).Match(sessionPrefix+"*").Count(16).Build()).AsScanEntry()
		if err != nil {
			return fmt.Errorf("valkey scan failed: %w", err)
		}

		for _, key := range scanEntry.Elements {

			value, err := s.client.Do(ctx, s.client.B().Get().Key(key).Build()).ToString()
			if err != nil {
				return fmt.Errorf("valkey get failed: %w", err)
			}

			if value == id {
				keysToDelete = append(keysToDelete, key)
			}

		}

		if scanEntry.Cursor == 0 {
			break scanLoop
		}
	}

	_, err := s.client.Do(ctx, s.client.B().Del().Key(keysToDelete...).Build()).AsInt64()
	if err != nil {
		return fmt.Errorf("valkey del failed: %w", err)
	}

	return nil

}
