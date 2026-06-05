package room

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/emibotz/chat-server/internal/user"
)

var (
	ErrRoomNotExist = fmt.Errorf("room does not exist.")

	ErrAlreadyInRoom = fmt.Errorf("user is already in this room.")
	ErrRoomIsFull    = fmt.Errorf("this room is full.")

	ErrNotInRoom = fmt.Errorf("user is not in this room.")
)

type Service struct {
	mu sync.RWMutex

	rooms map[int64]*Room

	maxRoomNumber  int64
	freeRoomNumber []int64
}

func NewService() *Service {
	return &Service{
		rooms: make(map[int64]*Room),

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

	s.rooms[num] = r

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
	for _, r := range s.rooms {
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
	r, ok := s.rooms[num]
	if !ok {
		return nil, ErrRoomNotExist
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

	if slices.Contains(room.Users, user.ID) {
		return ErrAlreadyInRoom
	}

	room.Users = append(room.Users, user.ID)

	return nil
}

func (s *Service) UserLeaveRoom(ctx context.Context, room *Room, user *user.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if room.Owner == user.ID {
		return s.DeleteRoom(ctx, room)
	}

	i := slices.Index(room.Users, user.ID)

	if i < 0 {
		return ErrNotInRoom
	}

	room.Users = append(room.Users[:i], room.Users[i+1:]...)

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

	// 删除房间
	delete(s.rooms, room.Num)

	return nil
}
