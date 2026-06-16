package room

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/emibotz/chat-server/internal/user"
	pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"
	"github.com/emibotz/chat-server/pkg/errcode"
	"github.com/emibotz/chat-server/pkg/key"
	"github.com/emibotz/chat-server/pkg/logging"
	"github.com/emibotz/chat-server/pkg/network"
	"github.com/google/uuid"
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

func (h *handler) createRoom(c *network.ClientRequestContext) error {

	// 从上下文中获取用户，应该被用户处理器提前注入。
	u, ok := c.Value(key.ContextUser).(*user.User)
	if !ok {
		return errcode.SendUnauthorized(c)
	}

	// 创建房间
	r, err := h.roomService.CreateRoom(c, u)
	if err != nil {

		// 如果用户已在房间内，返回用户已在房间内
		if errors.Is(err, ErrAlreadyInRoom) {
			return errcode.SendError(c, errcode.UserAlreadyInRoom)
		}

		// 否则返回内部错误
		logging.Error("create room failed", err)
		return errcode.SendInternalError(c)

	}

	id := u.ID.String()

	// 创建房主信息
	owner := &pbuf.ServerUserInfo{
		Id:   &id,
		Name: &u.Name,
	}

	// 由于这里房间内不应该有任何其他用户，所以直接返回房主信息
	users := []*pbuf.ServerUserInfo{
		&pbuf.ServerUserInfo{
			Id:   &id,
			Name: &u.Name,
		},
	}

	// 创建加入房间事件，发送房间信息
	event := &pbuf.ServerEvent{
		Data: &pbuf.ServerEvent_RoomJoined{
			RoomJoined: &pbuf.RoomJoined{
				Num:   &r.Num,
				Name:  &r.Name,
				Owner: owner,
				Users: users,
			},
		},
	}

	// 发送事件
	if err := c.Client.SendEvent(event); err != nil {
		logging.Error("send room created failed", err)
		return fmt.Errorf("client send room created failed: %w", err)
	}

	return nil
}

func (h *handler) getRooms(c *network.ClientRequestContext) error {

	// 查询房间
	rooms, err := h.roomService.GetRooms(c)
	if err != nil {
		return errcode.SendError(c, errcode.InternalError)
	}

	// 查询每个房间的房主名字
	ownerIDs := make([]uuid.UUID, len(rooms))
	for i, r := range rooms {
		ownerIDs[i] = r.Owner
	}

	ownersByIDs, err := h.userService.GetUsersByIDs(c, ownerIDs...)
	if err != nil {
		logging.Error("get users by ids failed", err)
		return errcode.SendInternalError(c)
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

		owner, ok := ownersByIDs[r.Owner]
		if !ok || owner == nil {
			logging.Error("failed to get owner by id. this should not happen!", nil)
		}

		userCount := int32(len(r.Users))
		maxUserCount := int32(0)

		roomInfo := pbuf.RoomInfo{
			Num:          &r.Num,
			Name:         &r.Name,
			Owner:        &owner.Name,
			UserCount:    &userCount,
			MaxUserCount: &maxUserCount,
		}

		roomInfos.Rooms = append(roomInfos.Rooms, &roomInfo)
	}

	// 发送信息
	if err := c.Client.SendEvent(event); err != nil {
		return fmt.Errorf("client send event failed: %w", err)
	}

	return nil
}

func (h *handler) joinRoom(c *network.ClientRequestContext) error {

	// 从上下文中获取用户，应该被用户处理器注入
	user, ok := c.Value(key.ContextUser).(*user.User)

	// 如果没有用户，返回未认证
	if !ok {
		return errcode.SendUnauthorized(c)
	}

	// 获取指定房间
	num := c.Request.GetJoinRoom().GetNum()

	room, err := h.roomService.GetRoomByNum(c, num)
	if err != nil {
		logging.Error("get room by num failed", err)
		return errcode.SendInternalError(c)
	}

	// 如果没有房间，返回房间不存在
	if room == nil {
		return errcode.SendError(c, errcode.RoomNotFound)
	}

	// 用户加入房间
	if err := h.roomService.UserJoinRoom(c, room, user); err != nil {

		// 如果用户已在房间内，返回用户已在房间内
		if errors.Is(err, ErrAlreadyInRoom) {
			return errcode.SendError(c, errcode.UserAlreadyInRoom)
		}

		// 如果房间已满，返回房间已满
		if errors.Is(err, ErrRoomIsFull) {
			return errcode.SendError(c, errcode.RoomIsFull)
		}

		// 返回服务器内部错误
		logging.Error("user join room failed", err)
		return errcode.SendInternalError(c)
	}

	// 获取房间内所有用户对应的客户端
	clients, err := c.Server.GetClientsByUserIDs(c, room.Users...)
	if err != nil {
		logging.Error("get clients by user ids failed", err)
		return errcode.SendInternalError(c)
	}

	// 创建用户进入房间事件
	id := user.ID.String()

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
			logging.Error("send user joined event to client failed", err)
		}

	}

	// 获取房间内用户信息
	users, err := h.userService.GetUsersByIDs(c, room.Users...)
	if err != nil {
		logging.Error("get users by ids failed", err)
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

func (h *handler) leaveRoom(c *network.ClientRequestContext) error {

	// 从上下文中获取用户，应该被用户处理器注入
	user, ok := c.Value(key.ContextUser).(*user.User)

	// 如果没有用户，返回未认证
	if !ok {
		return errcode.SendUnauthorized(c)
	}

	// 通过用户 ID 获取用户所在房间
	room, err := h.roomService.GetRoomByUserID(c, user.ID)
	if err != nil {
		// 否则返回服务器内部错误
		logging.Error("get room by user id failed", err)
		return errcode.SendInternalError(c)
	}

	// 如果没有房间，返回用户不在房间中
	if room == nil {
		return errcode.SendError(c, errcode.UserNotInRoom)
	}

	// 用户退出房间
	if err := h.roomService.UserLeaveRoom(c, room, user); err != nil {
		// 否则返回服务器内部错误
		logging.Error("user leave room failed", err)
		return errcode.SendInternalError(c)
	}

	// 获取房间内所有用户对应的客户端
	clients, err := c.Server.GetClientsByUserIDs(c, room.Users...)
	if err != nil {
		logging.Error("get clients by user ids failed", err)
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
			logging.Error("send user left failed", err)
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

func (h *handler) startGame(c *network.ClientRequestContext) error {

	// 从上下文中获取用户，应该被用户处理器注入
	user, ok := c.Value(key.ContextUser).(*user.User)

	// 如果没有用户，返回未认证
	if !ok {
		return errcode.SendUnauthorized(c)
	}

	// 获取用户所在房间
	room, err := h.roomService.GetRoomByUserID(c, user.ID)
	if err != nil {
		logging.Error("get room by user id failed", err)
		return errcode.SendInternalError(c)
	}

	// 如果没有房间，返回用户不在房间中
	if room == nil {
		return errcode.SendError(c, errcode.UserNotInRoom)
	}

	// 如果用户不是房主，返回用户权限不足
	if room.Owner != user.ID {
		return errcode.SendError(c, errcode.InsufficientPermission)
	}

	// 开始游戏
	if err := h.roomService.RoomStartGame(c, room); err != nil {

		// 如果游戏已经开始，返回游戏已经开始
		if errors.Is(err, ErrGameAlreadyStarted) {
			return errcode.SendError(c, errcode.GameAlreadyStarted)
		}

		// 否则返回服务器内部错误
		logging.Error("room start game failed", err)
		return errcode.SendInternalError(c)
	}

	// 通过房间内用户 ID 获取对应客户端
	clients, err := c.Server.GetClientsByUserIDs(c, room.Users...)
	if err != nil {
		logging.Error("get clients by user ids failed", err)
		return errcode.SendInternalError(c)
	}

	// 创建事件
	roomGameStarted := &pbuf.ServerEvent{
		Data: &pbuf.ServerEvent_RoomGameStarted{
			RoomGameStarted: &pbuf.RoomGameStarted{},
		},
	}

	// 遍历客户端
	for _, client := range clients {

		// 如果客户端为空，无法发送事件。
		if client == nil {
			continue
		}

		// 发送事件
		if err := client.SendEvent(roomGameStarted); err != nil {

			// 其他客户端发送失败不应该影响当前客户端连接处理
			logging.Error("client send roomGameStarted failed", err)
			continue
		}

	}

	return nil
}

func (h *handler) stopGame(c *network.ClientRequestContext) error {

	// 从上下文中获取用户，应该被用户处理器注入
	user, ok := c.Value(key.ContextUser).(*user.User)

	// 如果没有用户，返回未认证
	if !ok {
		return errcode.SendUnauthorized(c)
	}

	// 通过用户 ID 获取用户所在房间
	room, err := h.roomService.GetRoomByUserID(c, user.ID)
	if err != nil {
		logging.Error("get room by user id failed", err)
		return errcode.SendInternalError(c)
	}

	// 如果没有房间，返回用户不在房间中
	if room == nil {
		return errcode.SendError(c, errcode.UserNotInRoom)
	}

	// 如果用户不是房主，返回权限不足
	if user.ID != room.Owner {
		return errcode.SendError(c, errcode.InsufficientPermission)
	}

	// 如果房间内没有正在进行的游戏，返回游戏未开始
	if room.Game == nil {
		return errcode.SendError(c, errcode.GameNotStarted)
	}

	// 结束游戏
	room.Game.Stop()

	// 通过房间内用户 ID 获取对应客户端
	clients, err := c.Server.GetClientsByUserIDs(c, room.Users...)
	if err != nil {
		logging.Error("get clients by user ids failed", err)
		return errcode.SendInternalError(c)
	}

	// 创建事件
	event := &pbuf.ServerEvent{
		Data: &pbuf.ServerEvent_RoomGameStopped{
			RoomGameStopped: &pbuf.RoomGameStopped{},
		},
	}

	// 遍历客户端
	for _, client := range clients {

		// 如果客户端为空，无法发送，直接跳过。
		if client == nil {
			continue
		}

		// 发送事件
		if err := client.SendEvent(event); err != nil {
			// 单个客户端传输发生错误不应该影响
			// 其他客户端，打印日志后直接跳过。
			logging.Error("client send room game stopped failed", err)
			continue

		}

	}

	return nil
}

func (h *handler) HandleRequest(c *network.ClientRequestContext) (handled bool, err error) {

	if createRoom := c.Request.GetCreateRoom(); createRoom != nil {
		return true, h.createRoom(c)
	} else if getRooms := c.Request.GetGetRooms(); getRooms != nil {
		return true, h.getRooms(c)
	} else if joinRoom := c.Request.GetJoinRoom(); joinRoom != nil {
		return true, h.joinRoom(c)
	} else if leaveRoom := c.Request.GetLeaveRoom(); leaveRoom != nil {
		return true, h.leaveRoom(c)
	} else if startGame := c.Request.GetStartGame(); startGame != nil {
		return true, h.startGame(c)
	} else if stopGame := c.Request.GetStopGame(); stopGame != nil {
		return true, h.stopGame(c)
	}

	return false, nil
}

func (h *handler) HandleClose(c *network.ClientCloseContext) {

	// 从上下文中获取用户，应该被认证处理器提前注入
	u, ok := c.Value(key.ContextUser).(*user.User)
	if !ok {

		slog.Info(
			"found no user, returning.",
			slog.String("type", fmt.Sprintf("%T", c.Value(key.ContextUser))),
		)

		// 如果没有用户，无须额外处理，直接返回
		return
	}

	// 通过用户 ID 获取用户所在房间
	r, err := h.roomService.GetRoomByUserID(c, c.Client.GetUserID())
	if err != nil {
		logging.Error("get room by user id failed", err)
	}

	// 如果用户不在房间中，无须额外处理，直接返回
	if r == nil {
		return
	}

	// [FIXME] 这里手动广播了一次，是否需要优化呢

	// 获取房间内所有用户对应的客户端
	clients, err := c.Server.GetClientsByUserIDs(c, r.Users...)
	if err != nil {
		logging.Error("get clients by user ids failed", err)
	}

	// 创建用户退出事件
	id := u.ID.String()

	userLeft := &pbuf.ServerEvent{
		Data: &pbuf.ServerEvent_RoomUserLeft{
			RoomUserLeft: &pbuf.RoomUserLeft{
				User: &pbuf.ServerUserInfo{
					Id:   &id,
					Name: &u.Name,
				},
			},
		},
	}

	// 遍历所有客户端，发送用户退出事件
	for userID, client := range clients {

		// 如果客户端为空，那么无法发送事件。
		// 如果用户 ID 和请求者相同，那么无需发送事件。
		if client == nil || userID == u.ID {
			continue
		}

		// 发送事件
		if err := client.SendEvent(userLeft); err != nil {

			// 对其他客户端发送失败，不应该影响当前客户端的请求处理。
			logging.Error("send user left failed", err)
		}

	}

	// 使用户退出房间
	if err := h.roomService.UserLeaveRoom(c, r, u); err != nil {
		logging.Error("user leave room failed", err)
	}

}
