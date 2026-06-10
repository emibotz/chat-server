package room_test

import (
	"testing"

	"github.com/emibotz/chat-server/internal/game"
	"github.com/emibotz/chat-server/internal/room"
	"github.com/emibotz/chat-server/internal/store/mock"
	"github.com/emibotz/chat-server/internal/user"
)

func failIfErr(t *testing.T, err error) {
	if err != nil {
		t.Errorf("%v", err)
	}
}

var (
	// 仓库
	sessionStore = mock.SessionStore()
	userStore    = mock.UserStore()

	// 服务
	userService = user.NewService(sessionStore, userStore)
	gameService = game.NewService()
	roomService = room.NewService(userService, gameService)
)

func TestRoomGameFlow(t *testing.T) {
	// 获取上下文
	ctx := t.Context()

	// 创建模拟用户
	token1, err := userService.Register(ctx, "emi", "emibotzpassword")
	failIfErr(t, err)

	token2, err := userService.Register(ctx, "emi2", "emibotzpassword")
	failIfErr(t, err)

	// 获取模拟用户
	userID1, err := userService.VerifyToken(ctx, token1)
	failIfErr(t, err)
	user1, err := userService.GetUserByID(ctx, userID1)
	failIfErr(t, err)

	userID2, err := userService.VerifyToken(ctx, token2)
	failIfErr(t, err)
	user2, err := userService.GetUserByID(ctx, userID2)
	failIfErr(t, err)

	// 创建房间
	room, err := roomService.CreateRoom(ctx, user1)
	failIfErr(t, err)

	// 加入房间
	failIfErr(t, roomService.UserJoinRoom(ctx, room, user2))

	// 开始游戏
	failIfErr(t, roomService.RoomStartGame(ctx, room))

	// 房间内应该有两名玩家
	player1, err := room.Game.GetPlayerByUserID(ctx, user1.ID)
	failIfErr(t, err)
	player2, err := room.Game.GetPlayerByUserID(ctx, user2.ID)
	failIfErr(t, err)

	if player1 == nil || player2 == nil {
		t.Errorf("get player by user id failed.\n")
	}

	// 停止游戏
	failIfErr(t, roomService.RoomStopGame(ctx, room))
}
