package game

import (
	"context"
	"slices"
	"sync"

	"github.com/google/uuid"
)

type Game struct {
	mu   sync.RWMutex
	ctx  context.Context
	stop context.CancelFunc

	id uuid.UUID

	players         []*Player
	playersByID     map[uuid.UUID]*Player
	playersByUserID map[uuid.UUID]*Player

	chatMessages []*ChatMessage

	playerMoveIntentions     []*playerMoveIntention
	playerMoveIntentionsByID map[uuid.UUID]*playerMoveIntention
}

func New(ctx context.Context, stop context.CancelFunc) (*Game, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return &Game{
		mu:   sync.RWMutex{},
		ctx:  ctx,
		stop: stop,

		id: id,

		players:         make([]*Player, 0),
		playersByID:     make(map[uuid.UUID]*Player),
		playersByUserID: make(map[uuid.UUID]*Player),
	}, nil
}

func (g *Game) GetGameContext() context.Context {
	return g.ctx
}

func (g *Game) Stop() {
	g.stop()
}

// 这里三个没有加锁的函数放在一起，因为他们处理的变量基本相同。

// 将玩家添加到玩家列表中，并且建立键值连接
// 添加玩家的底层实现，没有加锁和 duplicate 检测，需要自行处理。
func (g *Game) addPlayer(player *Player) {
	g.players = append(g.players, player)
	g.playersByID[player.GetID()] = player
	g.playersByUserID[player.GetUserID()] = player
}

// 将玩家从玩家列表中删除，并且建立键值连接
// 删除玩家的底层实现，没有加锁和存在检测，需要自行处理。
func (g *Game) removePlayer(player *Player) {
	i := slices.Index(g.players, player)

	g.players = slices.Delete(g.players, i, i+1)
	delete(g.playersByID, player.GetID())
	delete(g.playersByUserID, player.GetUserID())
}

// 直接用新的数组和表覆盖旧的，以做到移除引用的效果
// 清除玩家的底层实现，没有加锁，需要自行处理。
func (g *Game) clearPlayers() {
	g.players = make([]*Player, 0)
	g.playersByID = make(map[uuid.UUID]*Player)
	g.playersByUserID = make(map[uuid.UUID]*Player)
}

func (g *Game) AddPlayer(ctx context.Context, player *Player) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	g.addPlayer(player)

	return nil
}

func (g *Game) RemovePlayer(ctx context.Context, player *Player) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	g.removePlayer(player)

	return nil
}

func (g *Game) GetPlayerByID(ctx context.Context, playerID uuid.UUID) (*Player, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	p, ok := g.playersByID[playerID]

	if !ok {
		return nil, nil
	}

	return p, nil
}

// 通过多个玩家 ID 获取对应的玩家，返回 map[uuid.UUID\]*Player 。
// 当无法通过玩家 ID 找到对应的玩家时，表中对应值为 nil ，需要自行检查。
func (g *Game) GetPlayersByIDs(ctx context.Context, playerIDs ...uuid.UUID) (map[uuid.UUID]*Player, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	result := make(map[uuid.UUID]*Player)

	for _, playerID := range playerIDs {

		p, ok := g.playersByID[playerID]

		if !ok {
			result[playerID] = nil
			continue
		}

		result[playerID] = p
	}

	return result, nil
}

func (g *Game) GetPlayerByUserID(ctx context.Context, userID uuid.UUID) (*Player, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	p, ok := g.playersByUserID[userID]

	if !ok {
		return nil, nil
	}

	return p, nil
}

func (g *Game) PopPlayers(ctx context.Context) ([]*Player, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	players := g.players
	g.clearPlayers()

	return players, nil
}

func (g *Game) WithPlayersByUserID(ctx context.Context, handle func(playersByUserID map[uuid.UUID]*Player) error) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return handle(g.playersByUserID)
}

func (g *Game) AddChatMessage(ctx context.Context, senderID uuid.UUID, message string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	g.chatMessages = append(g.chatMessages, &ChatMessage{senderID: senderID, message: message})

	return nil
}

func (g *Game) PopChatMessages(ctx context.Context) []*ChatMessage {
	g.mu.Lock()
	defer g.mu.Unlock()

	chatMessages := g.chatMessages
	g.chatMessages = make([]*ChatMessage, 0)

	return chatMessages
}

// 创建玩家移动意图的内部实现。
// 这个函数没有加锁，也没有 duplication 检查，需要在调用前自行处理。
func (g *Game) addPlayerMoveIntention(playerID uuid.UUID, direction Vector2) *playerMoveIntention {
	intention := &playerMoveIntention{
		playerID:  playerID,
		Direction: direction,
	}

	g.playerMoveIntentions = append(g.playerMoveIntentions, intention)
	g.playerMoveIntentionsByID[playerID] = intention

	return intention
}

// 查找玩家移动意图的内部实现，必定返回一个可用的意图。
// 这个函数没有加锁，需要在调用前自行处理。
func (g *Game) getPlayerMoveIntention(playerID uuid.UUID) *playerMoveIntention {

	// 先采用最快的哈希表查找
	intention, ok := g.playerMoveIntentionsByID[playerID]
	if ok {
		return intention
	}

	// 如果哈希表中查不到，尝试在列表中查找，which should not be a real situation
	// [FIXME] 这可能是相当大的性能开销，也许需要移除。
	for _, intention := range g.playerMoveIntentions {

		// 如果列表中的玩家 ID 等于需要查找的 ID ，在哈希表中建立键值连接，然后返回
		if intention.playerID == playerID {
			g.playerMoveIntentionsByID[playerID] = intention
			return intention
		}

	}

	// 如果都查不到，创建新的意图
	return g.addPlayerMoveIntention(playerID, Vector2Zero)

}

func (g *Game) SetPlayerMoveIntention(ctx context.Context, playerID uuid.UUID, direction Vector2) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	intention := g.getPlayerMoveIntention(playerID)
	intention.Direction = direction

	return nil

}

func (g *Game) GetPlayerMoveIntention(ctx context.Context, playerID uuid.UUID) (*playerMoveIntention, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return g.getPlayerMoveIntention(playerID), nil

}

// 返回并清除玩家意图和玩家 ID 对应表
func (g *Game) PopPlayerMoveIntentionsByID(ctx context.Context) (map[uuid.UUID]*playerMoveIntention, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	intentions := g.playerMoveIntentionsByID

	g.playerMoveIntentions = make([]*playerMoveIntention, 0)
	g.playerMoveIntentionsByID = make(map[uuid.UUID]*playerMoveIntention)

	return intentions, nil
}
