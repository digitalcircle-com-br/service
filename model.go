package service

type VO struct {
	ID uint64 `json:"id"`
}

type SecUser struct {
	VO
	Username string `json:"username"`
	Hash     string `json:"-"`
	Tenant   string `json:"tenant"`
	Enabled  *bool  `json:"enabled"`
}

type SecPerm struct {
	VO
	Username string `json:"username"`
	Hash     string `json:"hash"`
	Tenant   string `json:"tenant"`
}

const COOKIE = "X-SESSIONID"

type LoginRequest struct {
	Tenant   string `json:"tenant"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
}
