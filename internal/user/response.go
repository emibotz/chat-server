package user

type registerResponse struct {
	Token string `json:"token" form:"token"`
}

type loginResponse struct {
	Token string `json:"token" form:"token"`
}
