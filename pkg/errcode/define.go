package errcode

import pbuf "github.com/emibotz/chat-server/pkg/buf.gen/proto"

// XXXYYY
// XXX = 错误接口
// YYY = 具体类型

type apiType int32

var (
	SystemAPI apiType = 000
	RoomAPI   apiType = 001
)

type errType int32

func NewError(api apiType, e errType, err string) *pbuf.ServerError {
	code := int32(api)*1000 + int32(e)

	return &pbuf.ServerError{
		Code:  &code,
		Error: &err,
	}
}

func NewEvent(e *pbuf.ServerError) *pbuf.ServerEvent {
	return &pbuf.ServerEvent{
		Data: &pbuf.ServerEvent_Error{
			Error: e,
		},
	}
}
