package room

import (
	"errors"
	"fmt"

	"github.com/emibotz/chat-server/internal/network"
	"github.com/emibotz/chat-server/internal/user"
	pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"
	"github.com/emibotz/chat-server/pkg/errcode"
)

type handler struct {
	userService *user.Service
	roomService *Service
}

func NewHandler(
	userService *user.Service,
	roomService *Service,
) *handler {
	return &handler{
		userService: userService,
		roomService: roomService,
	}
}

func (h *handler) getRooms(c *network.Context) error {

	// 查询房间
	rooms, err := h.roomService.GetRooms(c)
	if err != nil {
		return fmt.Errorf("get rooms failed: %w", err)
	}

	// 构建回复
	event := &pbuf.ServerEvent{
		Data: &pbuf.ServerEvent_Rooms{
			Rooms: &pbuf.RoomInfos{
				Rooms: nil,
			},
		},
	}
	roomInfos := event.GetRooms()

	for _, r := range rooms {
		roomInfo := pbuf.RoomInfo{
			Num:  &r.Num,
			Name: &r.Name,
		}

		roomInfos.Rooms = append(roomInfos.Rooms, &roomInfo)
	}

	// 发送信息
	if err := c.Client.SendEvent(event); err != nil {
		return fmt.Errorf("client send event failed: %w", err)
	}

	return nil
}

func (h *handler) joinRoom(c *network.Context) error {

	// 获取用户
	userID := c.Client.GetUserID()

	u, err := h.userService.GetUserByID(c, userID)
	if err != nil {
		return fmt.Errorf("get user by id failed: %w", err)
	}

	// 如果没有用户，返回错误信息
	if u == nil {
		event := errcode.NewEvent(errcode.WebSocketUnauthorized)

		if err := c.Client.SendEvent(event); err != nil {
			return fmt.Errorf("client send event failed: %w", err)
		}

		return nil
	}

	// 获取指定房间
	num := c.Request.GetJoinRoom().GetNum()

	r, err := h.roomService.GetRoomByNum(c, num)
	if err != nil {
		return fmt.Errorf("get room by num failed: %w", err)
	}

	// 用户加入房间
	if err := h.roomService.UserJoinRoom(c, r, u); err != nil {

		// 返回房间已满信息
		if errors.Is(err, ErrRoomIsFull) {
			event := errcode.NewEvent(errcode.RoomIsFull)

			if err := c.Client.SendEvent(event); err != nil {
				return fmt.Errorf("client send room error failed: %w", err)
			}

			return nil
		}

		// 返回常规错误信息
		return fmt.Errorf("user join room failed: %w", err)
	}

	// 发送加入成功信息
	event := &pbuf.ServerEvent{
		Data: &pbuf.ServerEvent_RoomJoined{},
	}

	if err := c.Client.SendEvent(event); err != nil {
		return fmt.Errorf("client send event failed: %w", err)
	}

	return nil
}

func (h *handler) leaveRoom(c *network.Context) error {

	// 获取用户
	userID := c.Client.GetUserID()

	user, err := h.userService.GetUserByID(c, userID)
	if err != nil {
		return fmt.Errorf("get user by id failed: %w", err)
	}

	// 如果没有用户，返回错误信息
	if user == nil {
		event := errcode.NewEvent(errcode.WebSocketUnauthorized)

		if err := c.Client.SendEvent(event); err != nil {
			return fmt.Errorf("client send error failed: %w", err)
		}
	}

	// 获取用户所在房间
	room, err := h.roomService.GetRoomByUserID(c, user.ID)
	if err != nil {
		return fmt.Errorf("get room by user id failed: %w", err)
	}

	// 用户退出房间
	if err := h.roomService.UserLeaveRoom(c, room, user); err != nil {
		if errors.Is(err, ErrNotInRoom) {
			event := errcode.NewEvent(errcode.UserNotInRoom)

			if err := c.Client.SendEvent(event); err != nil {
				return fmt.Errorf("client send error failed: %w", err)
			}

			return nil
		}

		return fmt.Errorf("user leave room failed: %w", err)
	}

	// 发送退出成功信息
	event := &pbuf.ServerEvent{
		Data: &pbuf.ServerEvent_RoomLeft{
			RoomLeft: &pbuf.RoomLeft{},
		},
	}

	if err := c.Client.SendEvent(event); err != nil {
		return fmt.Errorf("client send event failed: %w", err)
	}

	return nil
}

func (h *handler) Handle(c *network.Context) error {

	if getRooms := c.Request.GetGetRooms(); getRooms != nil {
		return h.getRooms(c)
	} else if joinRoom := c.Request.GetJoinRoom(); joinRoom != nil {
		return h.joinRoom(c)
	} else if leaveRoom := c.Request.GetLeaveRoom(); leaveRoom != nil {
		return h.leaveRoom(c)
	}

	return nil
}
