package errcode

var (
	// 未认证
	Unauthorized = NewError(SystemAPI, 01, "unauthorized")

	// 服务器内部错误
	InternalError = NewError(SystemAPI, 02, "internal error")
)
