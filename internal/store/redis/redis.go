package redis

import (
	"context"

	"github.com/emibotz/chat-server/internal/user"
	"github.com/redis/go-redis/v9"
)

type redisDB struct {
	redisClient *redis.Client

	sessions user.SessionStore
}

func New(ctx context.Context, addr string) (*redisDB, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &redisDB{
		redisClient: rdb,

		sessions: &sessions{rdb},
	}, nil
}

func (db *redisDB) Sessions() user.SessionStore {
	return db.sessions
}
