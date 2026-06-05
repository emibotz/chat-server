package user

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

// 错误信息
var (
	ErrInvalidUsername = fmt.Errorf("invalid username")
	ErrInvalidPassword = fmt.Errorf("invalid password")

	ErrUserExists   = fmt.Errorf("user exists")
	ErrUserNotExist = fmt.Errorf("user does not exist")
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
	validPassword = "emibotz"
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

	// 生成会话，持续 1 天
	token, err := s.sessions.Create(ctx, u.ID, 1*24*time.Hour)
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

	// 验证密码
	if err := VerifyPasswordWithAuth(password, u.Auth); err != nil {
		return "", err
	}

	// 创建会话，持续 3 天
	token, err := s.sessions.Create(ctx, u.ID, 3*24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("create session failed: %w", err)
	}

	return token, nil
}
