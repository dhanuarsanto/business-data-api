package domain

import "context"

type User struct {
	UserID   int    `json:"user_id,omitempty"`
	Username string `json:"username"`
	Password string `json:"-"`
	Rules    string `json:"rules"`
}

type UserRepository interface {
	GetByUsername(ctx context.Context, tenant string, username string) (User, error)
	Insert(ctx context.Context, tenant string, data User) error
	Update(ctx context.Context, tenant string, username string, password *string, rules *string) error
}
