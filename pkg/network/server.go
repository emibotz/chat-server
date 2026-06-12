package network

import (
	"context"

	"github.com/google/uuid"
)

type Server interface {
	GetClientByUserID(ctx context.Context, userID uuid.UUID) (Client, error)

	// 通过多个用户 ID 找到对应的客户端连接，返回 [map[uuid.UUID]*Client] 。
	// 当没有找到某个用户对应的客户端时，在表中对应值为空指针，需要自行检查。
	GetClientsByUserIDs(ctx context.Context, userIDs ...uuid.UUID) (map[uuid.UUID]Client, error)
}
