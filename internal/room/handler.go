package room

import (
	"log/slog"

	"github.com/emibotz/chat-server/internal/network"
	pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"
	"google.golang.org/protobuf/proto"
)

type handler struct {
	roomService *Service
}

func NewHandler(
	roomService *Service,
) *handler {
	return &handler{
		roomService: roomService,
	}
}

func (h *handler) getRooms(c *network.Context) {

	// 查询房间
	rooms, err := h.roomService.GetRooms(c)
	if err != nil {
		slog.Error(
			"get rooms failed",
			slog.String("error", err.Error()),
		)
		return
	}

	// 构建回复
	roomInfos := pbuf.RoomInfos{
		Rooms: nil,
	}
	for _, r := range rooms {
		roomInfo := pbuf.RoomInfo{
			Num:  &r.Num,
			Name: &r.Name,
		}

		roomInfos.Rooms = append(roomInfos.Rooms, &roomInfo)
	}

	// 结构化回复
	bytes, err := proto.Marshal(&roomInfos)
	if err != nil {
		slog.Error(
			"marshal room infos failed",
			slog.String("error", err.Error()),
		)
		return
	}

	// 发送信息
	if err := c.Client.Send(bytes); err != nil {
		slog.Error(
			"send ws message failed",
			slog.String("error", err.Error()),
		)
	}
}

func (h *handler) Handle(c *network.Context) {
	if getRooms := c.Request.GetGetRooms(); getRooms != nil {
		h.getRooms(c)
	}
}
