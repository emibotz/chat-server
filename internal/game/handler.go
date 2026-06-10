package game

import (
	"github.com/emibotz/chat-server/internal/network"
	"github.com/emibotz/chat-server/internal/user"
	"github.com/emibotz/chat-server/pkg/errcode"
	"github.com/emibotz/chat-server/pkg/key"
	"github.com/emibotz/chat-server/pkg/logger"
)

type handler struct {
	gameService *Service
}

func NewHandler(
	gameService *Service,
) *handler {
	return &handler{
		gameService: gameService,
	}
}

func (h *handler) move(c *network.Context) error {

	// 从上下文中获取用户，应该在用户处理器中注入
	user, ok := c.Value(key.ContextUser).(*user.User)
	if !ok {
		return errcode.SendUnauthorized(c)
	}

	// 通过用户 ID 获取其所在游戏
	game, err := h.gameService.GetGameByUserID(c, user.ID)
	if err != nil {
		logger.Error("get game by user id failed", err)
		return errcode.SendInternalError(c)
	}

	// 通过用户 ID 从游戏中获取玩家
	player, err := game.GetPlayerByUserID(c, user.ID)
	if err != nil {
		logger.Error("get player by user id failed", err)
		return errcode.SendInternalError(c)
	}

	// 获取移动请求数据
	moveRequest := c.Request.GetGameRequest().GetMove()

	// 设置玩家移动意图
	if err := game.SetPlayerMoveIntention(c, player.GetID(), Vector2{X: moveRequest.GetX(), Y: moveRequest.GetY()}); err != nil {
		logger.Error("set player move intention failed", err)
		return errcode.SendInternalError(c)
	}

	return nil
}

func (h *handler) chat(c *network.Context) error {

	// 从上下文中获取用户，应该在用户处理器中注入
	user, ok := c.Value(key.ContextUser).(*user.User)
	if !ok {
		return errcode.SendUnauthorized(c)
	}

	// 通过用户 ID 获取用户所在游戏
	game, err := h.gameService.GetGameByUserID(c, user.ID)
	if err != nil {
		logger.Error("get game by user id failed", err)
		return errcode.SendInternalError(c)
	}

	// 通过用户 ID 从游戏中获取玩家
	player, err := game.GetPlayerByUserID(c, user.ID)
	if err != nil {
		logger.Error("get player by user id failed", err)
		return errcode.SendInternalError(c)
	}

	// 获取聊天消息数据
	chatRequest := c.Request.GetGameRequest().GetChat()

	// 添加聊天消息
	if err := game.AddChatMessage(c, player.GetID(), chatRequest.GetMessage()); err != nil {
		logger.Error("add chat message failed", err)
		return errcode.SendInternalError(c)
	}

	return nil
}

func (h *handler) HandleWS(c *network.Context) (handled bool, err error) {

	if gameRequest := c.Request.GetGameRequest(); gameRequest != nil {

		if move := gameRequest.GetMove(); move != nil {
			return true, h.move(c)
		} else if chat := gameRequest.GetChat(); chat != nil {
			return true, h.chat(c)
		}

	}

	return false, nil
}
