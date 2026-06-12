package valkey

import (
	"context"

	"github.com/emibotz/chat-server/internal/user"
	"github.com/valkey-io/valkey-go"
)

type valkeyDB struct {
	client valkey.Client

	sessions user.SessionStore
}

func New(ctx context.Context, addr string) (*valkeyDB, error) {

	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{addr},
	})
	if err != nil {
		return nil, err
	}

	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		return nil, err
	}

	return &valkeyDB{
		client: client,

		sessions: &sessions{client},
	}, nil
}

func (db *valkeyDB) Sessions() user.SessionStore {
	return db.sessions
}
