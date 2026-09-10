package domain

type User struct {
	UserID   int    `json:"user_id,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
	Rules    string `json:"rules"`
}

type UserRepository interface {
	GetByUsernamePG(tenant string, username string) (User, error)
	GetByUsernameMS(tenant string, username string) (User, error)
	InsertPG(tenant string, data User) error
	InsertMS(tenant string, data User) error
	UpdatePG(tenant string, username string, data User) error
	UpdateMS(tenant string, username string, data User) error
}
