package middleware

import (
	"github.com/emibotz/chat-server/internal/game"
	pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"
	"github.com/emibotz/chat-server/pkg/logging"
	"github.com/google/uuid"
)

// 获取服务器状态，并将其广播到所有客户端。
// 最好最后注册这个中间件。
func Broadcast(broadcaster game.Broadcaster) game.TickMiddleware {
	return func(ctx *game.TickContext, tick game.TickHandler) game.TickHandler {
		return func(ctx *game.TickContext) error {

			// 首先运行游戏刻
			tickErr := tick(ctx)
			if tickErr != nil {

				// 游戏刻发生错误但没有 panic ，尝试继续处理。
				logging.Error("broadcast: error occurred when ticking game, still trying to broadcast.", tickErr)
			}

			// 初始化服务器刻事件
			serverTick := &pbuf.ServerTick{
				Players:  nil,
				Messages: nil,
			}

			userIDs := make([]uuid.UUID, 0)

			// 获取用户 ID 和玩家的对应表
			ctx.Game.WithPlayersByUserID(ctx, func(playersByUserID map[uuid.UUID]*game.Player) error {

				// 创建用户 ID 列表
				keys := make([]uuid.UUID, 0, len(playersByUserID))
				for u := range playersByUserID {
					keys = append(keys, u)
				}
				userIDs = keys

				// 遍历玩家
				for _, player := range playersByUserID {

					// 获取玩家数据
					id := player.GetID().String()
					name := player.GetName()
					x := player.GetPosition().X
					y := player.GetPosition().Y

					// 将玩家数据填充到服务器刻事件中
					playerState := &pbuf.PlayerState{
						Id:   &id,
						Name: &name,
						X:    &x,
						Y:    &y,
					}

					serverTick.Players = append(serverTick.Players, playerState)

				}

				return nil

			}) // WithPlayersByUserID

			// 遍历聊天消息
			for _, chat := range ctx.Game.PopChatMessages(ctx) {

				// 获取消息数据
				senderID := chat.GetSenderID().String()
				message := chat.GetMessage()

				// 将数据填充到服务器刻事件中
				chatMessage := &pbuf.ChatMessage{
					SenderId: &senderID,
					Message:  &message,
				}

				serverTick.Messages = append(serverTick.Messages, chatMessage)

			}

			broadcaster.Broadcast(ctx, serverTick, userIDs...)

			return tickErr
		}
	}
}
