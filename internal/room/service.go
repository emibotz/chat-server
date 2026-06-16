package room

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/emibotz/chat-server/internal/game"
	"github.com/emibotz/chat-server/internal/user"
	"github.com/google/uuid"
)

var (
	ErrAlreadyInRoom = fmt.Errorf("user is already in this room.")
	ErrRoomIsFull    = fmt.Errorf("this room is full.")

	ErrGameAlreadyStarted = fmt.Errorf("game is already started.")
	ErrGameNotStarted     = fmt.Errorf("game is not started.")
)

type Service struct {
	mu sync.RWMutex

	userService *user.Service
	gameService *game.Service

	roomsByNum    map[int64]*Room
	roomsByUserID map[uuid.UUID]*Room

	maxRoomNumber  int64
	freeRoomNumber []int64
}

func NewService(
	userService *user.Service,
	gameService *game.Service,
) *Service {
	return &Service{
		mu: sync.RWMutex{},

		userService: userService,
		gameService: gameService,

		roomsByNum:    make(map[int64]*Room),
		roomsByUserID: make(map[uuid.UUID]*Room),

		maxRoomNumber:  1,
		freeRoomNumber: nil,
	}
}

func (s *Service) CreateRoom(ctx context.Context, creator *user.User) (*Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 如果用户已经在某个房间内，返回用户已在房间内
	if _, ok := s.roomsByUserID[creator.ID]; ok {
		return nil, ErrAlreadyInRoom
	}

	// 计算房间号
	num := s.maxRoomNumber
	if len(s.freeRoomNumber) > 0 {
		num = s.freeRoomNumber[0]
		s.freeRoomNumber = s.freeRoomNumber[1:]
	} else {
		s.maxRoomNumber += 1
	}

	// 创建房间并添加到房间表里
	r := New(num, fmt.Sprintf("%s的房间", creator.Name), creator.ID)

	s.roomsByNum[num] = r
	s.roomsByUserID[creator.ID] = r

	return r, nil
}

func (s *Service) GetRooms(ctx context.Context) ([]*Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 查询房间
	result := make([]*Room, 0)
	for _, r := range s.roomsByNum {
		result = append(result, r)
	}

	return result, nil
}

func (s *Service) GetRoomByNum(ctx context.Context, num int64) (*Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 查询房间
	r, ok := s.roomsByNum[num]
	if !ok {
		return nil, nil
	}

	return r, nil
}

func (s *Service) GetRoomByUserID(ctx context.Context, userID uuid.UUID) (*Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 查询房间
	r, ok := s.roomsByUserID[userID]
	if !ok {
		return nil, nil
	}

	return r, nil
}

func (s *Service) UserJoinRoom(ctx context.Context, r *Room, u *user.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 如果当前用户已经在某个房间内，或房间里已经有当前用户，返回用户已经在房间里
	_, ok := s.roomsByUserID[u.ID]

	if ok || slices.Contains(r.Users, u.ID) {
		return ErrAlreadyInRoom
	}

	// 把玩家添加进房间中，并且建立联系
	r.Users = append(r.Users, u.ID)

	s.roomsByUserID[u.ID] = r

	// 如果房间中没有正在进行中的游戏，可以就此返回。
	if r.Game == nil {
		return nil
	}

	// 使用户加入进行中的游戏。
	if err := s.gameService.UserJoinGame(ctx, r.Game, u); err != nil {
		return err
	}

	return nil
}

// 删除房间的底层实现，没有加锁
func (s *Service) deleteRoom(room *Room) {

	// 如果房间有游戏，停止游戏运行
	if room.Game != nil {
		room.Game.Stop()
	}

	// 删除所有用户与当前房间的联系
	for _, userID := range room.Users {
		delete(s.roomsByUserID, userID)
	}

	// 从房间表中删除房间
	delete(s.roomsByNum, room.Num)

	// 把房间号标为空闲
	s.freeRoomNumber = append(s.freeRoomNumber, room.Num)
}

func (s *Service) UserLeaveRoom(ctx context.Context, r *Room, u *user.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 如果用户是房主，使房间内所有用户退出房间，然后删除房间
	if r.Owner == u.ID {
		s.deleteRoom(r)
		return nil
	}

	// 获取用户在房间中的索引
	i := slices.Index(r.Users, u.ID)

	// 如果玩家不在房间中，返回用户不在房间中
	if i < 0 {
		return nil
	}

	// 把用户从房间中移除，并且删除联系
	r.Users = append(r.Users[:i], r.Users[i+1:]...)

	delete(s.roomsByUserID, u.ID)

	// 如果房间中没有正在进行的游戏，可以就此返回。
	if r.Game == nil {
		return nil
	}

	// 使用户退出正在进行中的游戏。
	if err := s.gameService.UserLeaveGame(ctx, r.Game, u); err != nil {
		return err
	}

	return nil
}

func (s *Service) DeleteRoom(ctx context.Context, room *Room) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.deleteRoom(room)

	return nil
}

func (s *Service) RoomStartGame(ctx context.Context, r *Room) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 如果房间已经有游戏，返回游戏已开始
	if r.Game != nil {
		return ErrGameAlreadyStarted
	}

	// 创建游戏根上下文
	gameContext, cancel := context.WithCancel(context.Background())

	// 创建游戏
	g, err := game.New(gameContext, cancel)
	if err != nil {
		return err
	}
	r.Game = g

	// 获取房间内用户信息
	usersByIDs, err := s.userService.GetUsersByIDs(ctx, r.Users...)
	if err != nil {
		return err
	}

	// 构建用户列表
	users := make([]*user.User, 0, len(usersByIDs))
	for _, u := range usersByIDs {
		if u != nil {
			users = append(users, u)
		}
	}

	// 使用用户列表创建游戏。
	s.gameService.AddGameWithUsers(ctx, g, users...)

	return nil
}

func (s *Service) RoomStopGame(ctx context.Context, r *Room) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 如果房间没有游戏，返回游戏未开始
	if r.Game == nil {
		return ErrGameNotStarted
	}

	// 删除游戏
	if err := s.gameService.RemoveGame(ctx, r.Game); err != nil {
		return err
	}

	// 停止游戏
	r.Game.Stop()
	r.Game = nil

	return nil
}
