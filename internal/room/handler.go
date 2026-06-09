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

	user, err := h.userService.GetUserByID(c, userID)
	if err != nil {
		logger.Error("get user by id failed", err)
		return errcode.SendInternalError(c)
	}

	// 如果没有用户，返回未认证
	if user == nil {
		return errcode.SendUnauthorized(c)
	}

	// 获取指定房间
	num := c.Request.GetJoinRoom().GetNum()

	room, err := h.roomService.GetRoomByNum(c, num)
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
	if err := h.roomService.UserJoinRoom(c, room, user); err != nil {

		// 如果房间已满，返回房间已满
		if errors.Is(err, ErrRoomIsFull) {
			return errcode.SendError(c, errcode.RoomIsFull)
		}

		// 返回服务器内部错误
		logger.Error("user join room failed", err)
		return errcode.SendInternalError(c)
	}

	// 获取房间内所有用户对应的客户端
	clients, err := c.Server.GetClientsByUserIDs(c, room.Users...)
	if err != nil {
		logger.Error("get clients by user ids failed", err)
		return errcode.SendInternalError(c)
	}

	// 创建用户进入房间事件
	id := userID.String()

	userJoined := &pbuf.ServerEvent{
		Data: &pbuf.ServerEvent_RoomUserJoined{
			RoomUserJoined: &pbuf.RoomUserJoined{
				User: &pbuf.ServerUserInfo{
					Id:   &id,
					Name: &user.Name,
				},
			},
		},
	}

	// 广播用户进入房间事件
	for userID, client := range clients {

		// 如果客户端为空，无法发送事件。
		// 如果用户 ID 和请求者相同，无需发送事件。
		if client == nil || userID == user.ID {
			continue
		}

		// 发送用户进入房间
		if err := client.SendEvent(userJoined); err != nil {

			// 对其他客户端发送失败，不应该影响当前客户端的请求处理。
			logger.Error("send user joined event to client failed", err)
		}

	}

	// 获取房间内用户信息
	users, err := h.userService.GetUsersByIDs(c, room.Users...)
	if err != nil {
		logger.Error("get users by ids failed", err)
		return errcode.SendInternalError(c)
	}

	// 填充房间内用户信息
	owner := &pbuf.ServerUserInfo{}
	userInfos := make([]*pbuf.ServerUserInfo, 0)

	for _, user := range users {

		id := user.ID.String()

		// 如果用户 ID 和房主相同，填充房主信息
		if user.ID == room.Owner {
			owner.Id = &id
			owner.Name = &user.Name
		}

		// 添加用户信息到列表
		userInfos = append(userInfos, &pbuf.ServerUserInfo{
			Id:   &id,
			Name: &user.Name,
		})
	}

	// 创建加入成功事件
	event := &pbuf.ServerEvent{
		Data: &pbuf.ServerEvent_RoomJoined{
			RoomJoined: &pbuf.RoomJoined{
				Num:  &room.Num,
				Name: &room.Name,

				Owner: owner,
				Users: userInfos,
			},
		},
	}

	// 发送房间加入成功
	if err := c.Client.SendEvent(event); err != nil {
		return fmt.Errorf("client send event failed: %w", err)
	}

	return nil
}

func (h *handler) leaveRoom(c *network.Context) error {

	// 通过客户端获取用户 ID
	userID := c.Client.GetUserID()

	// 通过用户 ID 查询用户记录
	user, err := h.userService.GetUserByID(c, userID)
	if err != nil {

		// 如果查询失败，返回服务器内部错误
		logger.Error("get user by id failed", err)
		return errcode.SendInternalError(c)
	}

	// 如果没有用户，返回未认证
	if user == nil {
		return errcode.SendUnauthorized(c)
	}

	// 通过用户 ID 获取用户所在房间
	room, err := h.roomService.GetRoomByUserID(c, user.ID)
	if err != nil {

		// 如果用户不在房间中，返回用户不在房间中
		if errors.Is(err, ErrNotInRoom) {
			return errcode.SendError(c, errcode.UserNotInRoom)
		}

		// 否则返回服务器内部错误
		logger.Error("get room by user id failed", err)
		return errcode.SendInternalError(c)
	}

	// 用户退出房间
	if err := h.roomService.UserLeaveRoom(c, room, user); err != nil {

		// 如果用户不在房间中，返回用户不在房间中
		if errors.Is(err, ErrNotInRoom) {
			return errcode.SendError(c, errcode.UserNotInRoom)
		}

		// 否则返回服务器内部错误
		logger.Error("user leave room failed", err)
		return errcode.SendInternalError(c)
	}

	// 获取房间内所有用户对应的客户端
	clients, err := c.Server.GetClientsByUserIDs(c, room.Users...)
	if err != nil {
		logger.Error("get clients by user ids failed", err)
		return errcode.SendInternalError(c)
	}

	// 创建用户退出事件
	id := user.ID.String()

	userLeft := &pbuf.ServerEvent{
		Data: &pbuf.ServerEvent_RoomUserLeft{
			RoomUserLeft: &pbuf.RoomUserLeft{
				User: &pbuf.ServerUserInfo{
					Id:   &id,
					Name: &user.Name,
				},
			},
		},
	}

	// 遍历所有客户端，发送用户退出事件
	for userID, client := range clients {

		// 如果客户端为空，那么无法发送事件。
		// 如果用户 ID 和请求者相同，那么无需发送事件。
		if client == nil || userID == user.ID {
			continue
		}

		// 发送事件
		if err := client.SendEvent(userLeft); err != nil {

			// 对其他客户端发送失败，不应该影响当前客户端的请求处理。
			logger.Error("send user left failed", err)
		}

	}

	// 发送退出成功信息
	roomLeft := &pbuf.ServerEvent{
		Data: &pbuf.ServerEvent_RoomLeft{
			RoomLeft: &pbuf.RoomLeft{},
		},
	}

	if err := c.Client.SendEvent(roomLeft); err != nil {
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
