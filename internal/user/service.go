package user

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// 错误信息
var (
	ErrInvalidUsername = fmt.Errorf("invalid username.")
	ErrInvalidPassword = fmt.Errorf("invalid password.")

	ErrUserExists   = fmt.Errorf("user exists.")
	ErrUserNotExist = fmt.Errorf("user does not exist.")

	ErrTokenUnauthorized = fmt.Errorf("token unauthorized.")
)

// 正则表达式
var (
	//  首位字母，后接字母数字下划线，总长度 3-16
	regexpUsername = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{2,15}$`)

	// 可打印 ASCII 字符，总长度 8-64
	regexpPassword = regexp.MustCompile(`^[ -~]{8,64}$`)
)

// 必定可用的用户名和密码
var (
	validUsername = "emi"
	validPassword = "emibotzpassword"
)

// 验证用户名和密码是否可用
func IsValidCredential(username string, password string) error {
	if !regexpUsername.MatchString(username) {
		return ErrInvalidUsername
	}
	if !regexpPassword.MatchString(password) {
		return ErrInvalidPassword
	}
	return nil
}

type Service struct {
	sessions SessionStore
	users    Store
}

func NewService(
	sessions SessionStore,
	users Store,
) *Service {
	return &Service{
		sessions: sessions,
		users:    users,
	}
}

func (s *Service) VerifyUsername(ctx context.Context, username string) error {
	// 验证用户名是否可用
	if err := IsValidCredential(username, validPassword); err != nil {
		return err
	}

	// 查询是否存在相同用户名的用户
	u, err := s.users.GetByName(ctx, username)
	if err != nil {
		return fmt.Errorf("get user by name failed: %w", err)
	}

	if u != nil {
		return ErrUserExists
	}

	return nil
}

func (s *Service) Register(ctx context.Context, username string, password string) (string, error) {
	// 验证用户名和密码是否可用
	if err := IsValidCredential(username, password); err != nil {
		return "", err
	}

	// 生成认证串
	auth, err := GenerateAuth(Config, password)
	if err != nil {
		return "", fmt.Errorf("generate auth failed: %w", err)
	}

	// 创建用户
	u := New(username, auth)
	if err := s.users.Create(ctx, u); err != nil {
		return "", fmt.Errorf("create user failed: %w", err)
	}

	// 生成会话，持续 1 小时
	token, err := s.sessions.Create(ctx, u.ID, 1*time.Hour)
	if err != nil {
		return "", fmt.Errorf("create session failed: %w", err)
	}

	return token, nil
}

func (s *Service) Login(ctx context.Context, username string, password string) (string, error) {
	// 验证用户名和密码是否是可用格式
	if err := IsValidCredential(username, password); err != nil {
		return "", err
	}

	// 获取指定用户
	u, err := s.users.GetByName(ctx, username)
	if err != nil {
		return "", fmt.Errorf("get user by name failed: %w", err)
	}

	if u == nil {
		return "", ErrUserNotExist
	}

	// 验证密码，如果不通过直接返回错误
	if err := VerifyPasswordWithAuth(password, u.Auth); err != nil {
		return "", err
	}

	// 如果已有会话，将其删除
	s.sessions.DeleteAllByUserID(ctx, u.ID)

	// 创建会话，持续 1 小时
	token, err := s.sessions.Create(ctx, u.ID, 1*time.Hour)
	if err != nil {
		return "", fmt.Errorf("sessions create failed: %w", err)
	}

	return token, nil
}

func (s *Service) VerifyToken(ctx context.Context, token string) (uuid.UUID, error) {
	// 验证是否存在指定 Token
	id, err := s.sessions.Get(ctx, token)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return uuid.Nil, ErrTokenUnauthorized
		}

		return uuid.Nil, fmt.Errorf("session get failed: %w", err)
	}

	if id == uuid.Nil {
		return uuid.Nil, ErrTokenUnauthorized
	}

	return id, nil
}

// [TODO] 是否需要处理结果？
func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	return s.users.GetByID(ctx, userID)
}

// 返回用户 ID 和用户记录的对应表，当没有查询到指定
// 用户时，表中对应值为 nil ，需要自行处理。
// 时间复杂度 O(n) 的实现。
func (s *Service) GetUsersByIDs(ctx context.Context, userIDs ...uuid.UUID) (map[uuid.UUID]*User, error) {

	// 查询用户
	users, err := s.users.GetsByIDs(ctx, userIDs...)
	if err != nil {
		return nil, err
	}

	// 创建表，预分配容量
	result := make(map[uuid.UUID]*User, len(userIDs))

	// 遍历用户，填入表中
	for _, u := range users {
		result[u.ID] = u
	}

	// 遍历传入的 ID 列表，如果 ID 不在表中，将对应值设置为 nil
	for _, userID := range userIDs {
		if _, ok := result[userID]; !ok {
			result[userID] = nil
		}
	}

	// 返回
	return result, nil

}
