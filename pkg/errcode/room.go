package errcode

var (
	RoomIsFull    = NewError(RoomAPI, 01, "room is full")
	UserNotInRoom = NewError(RoomAPI, 02, "user is not in this room")
)
