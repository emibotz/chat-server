package room

import (
	"sync"

	"github.com/emibotz/chat-server/internal/game"
	"github.com/google/uuid"
)

// 理论上不应该有 room 包外的代码直接修改此结构体内的字段
// [TODO] 最大用户数量
type Room struct {
	mu sync.RWMutex

	ID   uuid.UUID
	Num  int64
	Name string

	Owner uuid.UUID
	Users []uuid.UUID

	Game *game.Game
}

func New(num int64, name string, creatorID uuid.UUID) *Room {
	return &Room{
		ID:   uuid.Must(uuid.NewV7()),
		Num:  num,
		Name: name,

		Owner: creatorID,
		Users: []uuid.UUID{creatorID},

		Game: nil,
	}
}
