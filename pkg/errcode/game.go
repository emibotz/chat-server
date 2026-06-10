package errcode

var (
	GameAlreadyStarted = NewError(GameAPI, 01, "game is already started.")
	GameNotStarted     = NewError(GameAPI, 02, "game is not started yet.")

	UserNotInGame = NewError(GameAPI, 03, "user is not in game.")
)
