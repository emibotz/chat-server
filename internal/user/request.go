package user

type registerRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

type loginRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}
