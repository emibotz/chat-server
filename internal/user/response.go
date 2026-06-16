package user

type registerResponse struct {
	Token string `json:"token"`
}

type loginResponse struct {
	Token string `json:"token"`
}
