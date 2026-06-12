package errcode

import "github.com/emibotz/chat-server/pkg/network"

var (
	// 未认证
	Unauthorized = NewError(SystemAPI, 01, "unauthorized.")

	// 服务器内部错误
	InternalError = NewError(SystemAPI, 02, "internal error.")

	// 版本不兼容
	IncompatibleVersion = NewError(SystemAPI, 03, "incompatible version.")
)

func SendUnauthorized(c *network.ClientRequestContext) error {
	return SendError(c, Unauthorized)
}

func SendInternalError(c *network.ClientRequestContext) error {
	return SendError(c, InternalError)
}
