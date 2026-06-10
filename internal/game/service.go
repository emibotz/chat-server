package game

import (
	"context"
	"slices"
	"sync"

	"github.com/emibotz/chat-server/internal/user"
	"github.com/google/uuid"
)

type Service struct {
	mu sync.RWMutex

	games         []*Game
	gamesByUserID map[uuid.UUID]*Game
}

func NewService() *Service {
	return &Service{
		mu: sync.RWMutex{},

		games:         make([]*Game, 0),
		gamesByUserID: make(map[uuid.UUID]*Game),
	}
}

// 添加游戏的底层实现，没有加锁，需要自行处理。
func (s *Service) addGame(game *Game, userIDs ...uuid.UUID) {

	// 添加游戏到列表中
	s.games = append(s.games, game)

	// 为游戏和每个用户 ID 之间建立键值连接，加速查找。
	for _, userID := range userIDs {
		s.gamesByUserID[userID] = game
	}

}

// 添加游戏，并且为即将加入游戏的用户们创建玩家
func (s *Service) AddGameWithUsers(ctx context.Context, game *Game, users ...*user.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 用户 ID 列表，用来添加游戏
	userIDs := make([]uuid.UUID, len(users))

	// 遍历每个用户
	for i, user := range users {

		// 通过用户的 ID 和名称创建玩家
		player, err := NewPlayer(user.ID, user.Name)
		if err != nil {
			return err
		}

		// 把玩家添加到游戏中
		if err := game.AddPlayer(ctx, player); err != nil {
			return err
		}

		// 把用户 ID 添加到列表中
		userIDs[i] = user.ID
	}

	// 添加游戏
	s.addGame(game, userIDs...)

	return nil
}

// [FIXME] 这代码可能风险有点大……
func (s *Service) RemoveGame(ctx context.Context, game *Game) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 将游戏从列表中删除
	i := slices.Index(s.games, game)

	s.games = slices.Delete(s.games, i, i+1)

	// 删除游戏和其中每个玩家对应的用户 ID 之间的键值连接
	return game.WithPlayersByUserID(ctx, func(playersByUserID map[uuid.UUID]*Player) error {

		for userID, _ := range playersByUserID {
			delete(s.gamesByUserID, userID)
		}

		return nil
	})
}

func (s *Service) GetGameByUserID(ctx context.Context, userID uuid.UUID) (*Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	game, ok := s.gamesByUserID[userID]

	if !ok {
		return nil, nil
	}

	return game, nil
}
