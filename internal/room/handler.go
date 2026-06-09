package room

import (
	"errors"
	"fmt"

	"github.com/emibotz/chat-server/internal/network"
	"github.com/emibotz/chat-server/internal/user"
	pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"
	"github.com/emibotz/chat-server/pkg/errcode"
	"github.com/emibotz/chat-server/pkg/logger"
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
		return errcode.SendError(c, errcode.InternalError)
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
		logger.Error("get user by id failed", err)
		return errcode.SendInternalError(c)
	}

	// 如果没有用户，返回未认证
	if u == nil {
		return errcode.SendUnauthorized(c)
	}

	// 获取指定房间
	num := c.Request.GetJoinRoom().GetNum()

	r, err := h.roomService.GetRoomByNum(c, num)
	if err != nil {

		// 如果房间不存在，返回房间不存在
		if errors.Is(err, ErrRoomNotExist) {
			return errcode.SendError(c, errcode.RoomNotFound)
		}

		// 返回服务器内部错误
		logger.Error("get room by num failed", err)
		return errcode.SendInternalError(c)
	}

	// 用户加入房间
	if err := h.roomService.UserJoinRoom(c, r, u); err != nil {

		// 如果房间已满，返回房间已满
		if errors.Is(err, ErrRoomIsFull) {
			return errcode.SendError(c, errcode.RoomIsFull)
		}

		// 返回服务器内部错误
		logger.Error("user join room failed", err)
		return errcode.SendInternalError(c)
	}

	// [TODO] 广播用户加入信息

	// [TODO] 获取房间内用户信息

	// 发送加入成功信息
	id := r.ID.String()

	event := &pbuf.ServerEvent{
		Data: &pbuf.ServerEvent_RoomJoined{
			RoomJoined: &pbuf.RoomJoined{
				Id:   &id,
				Num:  &r.Num,
				Name: &r.Name,

				// [TODO]
				Owner: nil,
				Users: nil,
			},
		},
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
		// 如果获取失败，返回服务器内部错误
		logger.Error("get user by id failed", err)
		return errcode.SendInternalError(c)
	}

	// 如果没有用户，返回未认证
	if user == nil {
		return errcode.SendUnauthorized(c)
	}

	// 获取用户所在房间
	room, err := h.roomService.GetRoomByUserID(c, user.ID)
	if err != nil {

		// 如果用户不在房间中，返回用户不在房间中
		if errors.Is(err, ErrNotInRoom) {
			return errcode.SendError(c, errcode.UserNotInRoom)
		}

		// 返回服务器内部错误
		logger.Error("get room by user id failed", err)
		return errcode.SendInternalError(c)
	}

	// 用户退出房间
	if err := h.roomService.UserLeaveRoom(c, room, user); err != nil {

		// 如果用户不在房间中，返回用户不在房间中
		if errors.Is(err, ErrNotInRoom) {
			return errcode.SendError(c, errcode.UserNotInRoom)
		}

		// 返回服务器内部错误
		logger.Error("user leave room failed", err)
		return errcode.SendInternalError(c)
	}

	// [TODO] 广播用户退出信息

	// [TODO] 当用户是房主时，发出房间解散广播

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

func (h *handler) HandleWS(c *network.Context) (handled bool, err error) {

	if getRooms := c.Request.GetGetRooms(); getRooms != nil {
		return true, h.getRooms(c)
	} else if joinRoom := c.Request.GetJoinRoom(); joinRoom != nil {
		return true, h.joinRoom(c)
	} else if leaveRoom := c.Request.GetLeaveRoom(); leaveRoom != nil {
		return true, h.leaveRoom(c)
	}

	return false, nil
}
