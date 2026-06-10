package errcode

var (
	RoomIsFull    = NewError(RoomAPI, 01, "room is full.")
	RoomNotFound  = NewError(RoomAPI, 02, "room not found.")
	UserNotInRoom = NewError(RoomAPI, 03, "user is not in this room.")
)
