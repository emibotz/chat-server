package room

import (
	"sync"

	"github.com/emibotz/chat-server/internal/game"
	"github.com/google/uuid"
)

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
