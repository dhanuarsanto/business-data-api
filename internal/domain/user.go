package domain

import "context"

type User struct {
	UserID   int    `json:"user_id,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
	Rules    string `json:"rules"`
}

type UserRepository interface {
	GetByUsernamePG(ctx context.Context, tenant string, username string) (User, error)
	GetByUsernameMS(ctx context.Context, tenant string, username string) (User, error)
	InsertPG(ctx context.Context, tenant string, data User) error
	InsertMS(ctx context.Context, tenant string, data User) error
	UpdatePG(ctx context.Context, tenant string, username string, data User) error
	UpdateMS(ctx context.Context, tenant string, username string, data User) error
}
