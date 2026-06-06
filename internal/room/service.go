package room

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/emibotz/chat-server/internal/user"
	"github.com/google/uuid"
)

var (
	ErrRoomNotExist = fmt.Errorf("room does not exist.")

	ErrAlreadyInRoom = fmt.Errorf("user is already in this room.")
	ErrRoomIsFull    = fmt.Errorf("this room is full.")

	ErrNotInRoom = fmt.Errorf("user is not in this room.")
)

type Service struct {
	mu sync.RWMutex

	roomsByNum    map[int64]*Room
	roomsByUserID map[uuid.UUID]*Room

	maxRoomNumber  int64
	freeRoomNumber []int64
}

func NewService() *Service {
	return &Service{
		mu: sync.RWMutex{},

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
		return nil, ErrRoomNotExist
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
		return nil, ErrNotInRoom
	}

	return r, nil
}

func (s *Service) UserJoinRoom(ctx context.Context, room *Room, user *user.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 如果当前用户已经在某个房间内，或房间里已经有当前用户，返回用户已经在房间里
	_, ok := s.roomsByUserID[user.ID]

	if ok || slices.Contains(room.Users, user.ID) {
		return ErrAlreadyInRoom
	}

	// 把玩家添加进房间中，并且建立联系
	room.Users = append(room.Users, user.ID)

	s.roomsByUserID[user.ID] = room

	return nil
}

// 删除房间的底层实现，没有加锁
func (s *Service) deleteRoom(room *Room) {
	// 删除所有用户与当前房间的联系
	for _, userID := range room.Users {
		delete(s.roomsByUserID, userID)
	}

	// 从房间表中删除房间
	delete(s.roomsByNum, room.Num)

	// 把房间号标为空闲
	s.freeRoomNumber = append(s.freeRoomNumber, room.Num)
}

func (s *Service) UserLeaveRoom(ctx context.Context, room *Room, user *user.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 如果用户是房主，使房间内所有用户退出房间，然后删除房间
	if room.Owner == user.ID {
		s.deleteRoom(room)
		return nil
	}

	// 获取用户在房间中的索引
	i := slices.Index(room.Users, user.ID)

	// 如果玩家不在房间中，返回用户不在房间中
	if i < 0 {
		return ErrNotInRoom
	}

	// 把用户从房间中移除，并且删除联系
	room.Users = append(room.Users[:i], room.Users[i+1:]...)

	delete(s.roomsByUserID, user.ID)

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
