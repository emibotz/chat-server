package middleware

import (
	"fmt"

	"github.com/emibotz/chat-server/internal/game"
	"github.com/google/uuid"
)

// 在每个游戏刻开始前，使所有玩家按其发送的意图移动。
func PlayerController() game.TickMiddleware {
	return func(ctx *game.TickContext, tick game.TickHandler) game.TickHandler {
		return func(ctx *game.TickContext) error {

			// 获取所有玩家移动意图
			intentions, err := ctx.Game.PopPlayerMoveIntentionsByID(ctx)
			if err != nil {
				return fmt.Errorf("get player move intentions by id failed: %w", err)
			}

			// 获取所有意图对应的玩家
			keys := make([]uuid.UUID, 0, len(intentions))
			for u := range intentions {
				keys = append(keys, u)
			}

			players, err := ctx.Game.GetPlayersByIDs(ctx, keys...)
			if err != nil {
				return fmt.Errorf("get players by ids failed: %w", err)
			}

			// [FIXME] 硬编码移动速度
			delta := float64(ctx.Delta.Milliseconds()) / 1000.0
			speed := 500.0

			// 根据意图修改玩家的位置
			for _, player := range players {
				intention := intentions[player.GetID()]

				direction := intention.Direction.Normalized()
				velocity := direction.Multiply(speed * delta)

				player.SetPosition(player.GetPosition().Add(velocity))
			}

			// 处理游戏刻
			return tick(ctx)
		}
	}
}
