package user

import (
	"context"

	"github.com/google/uuid"
)

type User struct {
	ID   uuid.UUID
	Name string
	Auth string
}

func New(name string, auth string) *User {
	return &User{
		ID:   uuid.Must(uuid.NewV7()),
		Name: name,
		Auth: auth,
	}
}

type Store interface {
	Create(ctx context.Context, user *User) error

	// 通过 ID 获取用户，没有指定用户时返回 (nil, nil)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)

	// 通过用户名获取用户，没有指定用户时返回 (nil, nil)
	GetByName(ctx context.Context, username string) (*User, error)

	Update(ctx context.Context, user *User) error

	Delete(ctx context.Context, user *User) error
}
