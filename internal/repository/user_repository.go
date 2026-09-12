package repository

import (
	"context"
	"database/sql"
	"fmt"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/pkg/database"
)

type userRepository struct {
	dbRegistry *database.DBRegistry
}

func (r *userRepository) GetByUsernamePG(ctx context.Context, tenant string, username string) (domain.User, error) {
	db := r.dbRegistry.Postgres(tenant)
	if db == nil {
		return domain.User{}, fmt.Errorf("tenant tidak ditemukan")
	}

	var u domain.User
	err := db.QueryRow(ctx, `SELECT userid, username, pass, rules FROM users WHERE username=$1`, username).Scan(&u.UserID, &u.Username, &u.Password, &u.Rules)
	return u, err
}

func (r *userRepository) GetByUsernameMS(ctx context.Context, tenant string, username string) (domain.User, error) {
	db := r.dbRegistry.MSSQL(tenant)
	if db == nil {
		return domain.User{}, fmt.Errorf("tenant tidak ditemukan")
	}

	var u domain.User
	err := db.QueryRowContext(ctx, `SELECT userid, username, pass, rules FROM users WHERE username=@username`, sql.Named("username", username)).Scan(&u.UserID, &u.Username, &u.Password, &u.Rules)
	return u, err
}

func (r *userRepository) InsertPG(ctx context.Context, tenant string, data domain.User) error {
	db := r.dbRegistry.Postgres(tenant)
	if db == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}
	_, err := db.Exec(ctx, `INSERT INTO users (username, pass, rules) VALUES ($1, $2, $3)`, data.Username, data.Password, data.Rules)
	return err
}

func (r *userRepository) InsertMS(ctx context.Context, tenant string, data domain.User) error {
	db := r.dbRegistry.MSSQL(tenant)
	if db == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}
	_, err := db.ExecContext(ctx, `INSERT INTO users (username, pass, rules) VALUES (@username, @password, @rules)`, sql.Named("username", data.Username), sql.Named("password", data.Password), sql.Named("rules", data.Rules))
	return err
}

func (r *userRepository) UpdatePG(ctx context.Context, tenant string, username string, data domain.User) error {
	db := r.dbRegistry.Postgres(tenant)
	if db == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}
	_, err := db.Exec(ctx, `UPDATE users SET pass=$1, rules=$2 WHERE username=$3`, data.Password, data.Rules, username)
	return err
}

func (r *userRepository) UpdateMS(ctx context.Context, tenant string, username string, data domain.User) error {
	db := r.dbRegistry.MSSQL(tenant)
	if db == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}
	_, err := db.ExecContext(ctx, `UPDATE users SET pass=@password, rules=@rules WHERE username=@username`, sql.Named("password", data.Password), sql.Named("rules", data.Rules), sql.Named("username", username))
	return err
}

func NewUserRepository(dbRegistry *database.DBRegistry) domain.UserRepository {
	return &userRepository{dbRegistry: dbRegistry}
}
