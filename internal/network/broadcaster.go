package network

import (
	"context"

	pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"
	"github.com/emibotz/chat-server/pkg/logging"
	"github.com/google/uuid"
)

type ServerBroadcaster struct {
	server *Server
}

func (s *Server) Broadcaster() *ServerBroadcaster {
	return &ServerBroadcaster{
		server: s,
	}
}

// 给所有指定用户 ID 的客户端广播服务器刻信息
func (b *ServerBroadcaster) Broadcast(ctx context.Context, tick *pbuf.ServerTick, userIDs ...uuid.UUID) {

	// 获取所有用户 ID 对应的客户端
	clients, err := b.server.GetClientsByUserIDs(ctx, userIDs...)
	if err != nil {
		logging.Error("broadcast failed", err)
	}

	// 创建事件
	event := &pbuf.ServerEvent{
		Data: &pbuf.ServerEvent_ServerTick{
			ServerTick: tick,
		},
	}

	// 遍历所有客户端
	for _, client := range clients {

		// 如果客户端为空，无法广播，直接跳过。
		if client == nil {
			continue
		}

		// 发送信息
		if err := client.SendEvent(event); err != nil {

			// 单个客户端广播失败不应该影响其他客户端的事件广播
			logging.Error("client broadcast serverTick failed", err)
			continue
		}

	}
}
