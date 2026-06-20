package game

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emibotz/chat-server/pkg/logging"
	"github.com/google/uuid"
)

type TickHandler func(ctx *TickContext) error
type TickMiddleware func(ctx *TickContext, tick TickHandler) TickHandler
type TickMiddlewareFactory func(g *Game) TickMiddleware

type TickContext struct {
	context.Context

	Delta time.Duration
	Game  *Game
}

func (c *TickContext) Value(k any) any {
	switch k {
	case "delta":
		return c.Delta
	case "game":
		return c.Game
	default:
	}

	return c.Context.Value(k)
}

type Clock struct {
	mu  sync.Mutex
	ctx context.Context

	ticker      *time.Ticker
	middlewares []TickMiddleware

	game *Game
}

func NewClock(
	ctx context.Context,
	tps int,
	game *Game,
) *Clock {
	duration := time.Second / time.Duration(tps)

	c := &Clock{
		mu:  sync.Mutex{},
		ctx: ctx,

		ticker:      time.NewTicker(duration),
		middlewares: nil,

		game: game,
	}

	go c.ticking()

	return c
}

func (c *Clock) UseMiddleware(middleware TickMiddleware) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.middlewares = append(c.middlewares, middleware)
}

func (c *Clock) handlePlayerMoveIntention(ctx *TickContext) error {

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
		if player == nil {
			continue
		}

		intention := intentions[player.GetID()]

		direction := intention.Direction.Normalized()
		velocity := direction.Multiply(speed * delta)

		player.SetPosition(player.GetPosition().Add(velocity))
	}

	return nil
}

func (c *Clock) gameTick(ctx *TickContext) error {

	if err := c.handlePlayerMoveIntention(ctx); err != nil {
		return err
	}

	return nil
}

func (c *Clock) ticking() {

	lastTick := time.Now()
	tick := time.Now()

tickLoop:
	for {
		select {

		// 如果时钟被关闭，退出循环
		case <-c.ctx.Done():
			logging.Error("clock: context is done, stopping...", c.ctx.Err())
			break tickLoop

		// 等待计时器
		case t := <-c.ticker.C:
			tick = t
		}

		// 计算两个游戏刻之间的间隔
		delta := tick.Sub(lastTick)
		lastTick = tick

		// 创建上下文
		ctx := &TickContext{
			Context: c.ctx,

			Delta: delta,
			Game:  c.game,
		}

		// 用中间件包裹游戏刻处理器
		tick := c.gameTick
		for _, handle := range c.middlewares {
			tick = handle(ctx, tick)
		}

		// 运行游戏刻处理器
		if err := tick(ctx); err != nil {
			fmt.Printf("some shit happend when ticking the game: %v\n", err)
		}
	}

	// [TODO] 时钟结束后处理
	c.ticker.Stop()
}
