package errcode

var (
	// 未认证
	WebSocketUnauthorized = NewError(SystemAPI, 01, "unauthorized")
)
