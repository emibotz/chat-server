package errcode

var (
	InsufficientPermission = NewError(UserAPI, 01, "insufficient user permission.")
)
