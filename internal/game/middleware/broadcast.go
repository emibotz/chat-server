package middleware

import (
	"github.com/emibotz/chat-server/internal/game"
	pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"
	"github.com/emibotz/chat-server/pkg/logging"
	"github.com/emibotz/chat-server/pkg/network"
	"github.com/google/uuid"
)

// 获取服务器状态，并将其广播到所有客户端。
// 最好最后注册这个中间件。
func Broadcast(server network.Server) game.TickMiddlewareFactory {
	return func(g *game.Game) game.TickMiddleware {
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

					// 遍历玩家
					for userID, player := range playersByUserID {

						// 将玩家对应的用户 ID 添加到列表中
						userIDs = append(userIDs, userID)

						// 获取玩家数据
						id := player.GetID().String()
						userID := player.GetUserID().String()
						name := player.GetName()
						x := player.GetPosition().X
						y := player.GetPosition().Y

						// 将玩家数据填充到服务器刻事件中
						playerState := &pbuf.PlayerState{
							Id:     &id,
							UserId: &userID,
							Name:   &name,
							X:      &x,
							Y:      &y,
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

				// 通过用户 ID 获取对应的客户端
				clients, err := server.GetClientsByUserIDs(ctx, userIDs...)
				if err != nil {
					return err
				}

				// 创建事件以供发送
				event := &pbuf.ServerEvent{
					Data: &pbuf.ServerEvent_ServerTick{
						ServerTick: serverTick,
					},
				}

				for _, client := range clients {

					// 如果客户端为空，无法发送事件，直接跳过。
					if client == nil {
						continue
					}

					if err := client.SendEvent(event); err != nil {

						// 单个客户端广播失败，不应该影响其他客户端的广播
						logging.Error("client send server tick failed", err)
						continue
					}

				}

				// 返回游戏刻错误？
				return tickErr
			}
		}
	}
}
