package dto

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Rules    string `json:"rules"`
}

type UpdateUserRequest struct {
	Password *string `json:"password,omitempty"`
	Rules    *string `json:"rules,omitempty"`
}
