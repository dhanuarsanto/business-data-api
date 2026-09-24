package repository

import (
	"context"
	"database/sql"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/pkg/database"
)

type UserRepositories struct {
	PG domain.UserRepository
	MS domain.UserRepository
}

type userPGRepository struct {
	dbRegistry *database.DBRegistry
}

func (r *userPGRepository) GetByUsername(ctx context.Context, tenant string, username string) (domain.User, error) {
	db, err := r.dbRegistry.Postgres(tenant)
	if err != nil {
		return domain.User{}, err
	}

	var u domain.User
	err = db.QueryRow(ctx, `SELECT userid, username, pass, rules FROM users WHERE username=$1`, username).Scan(&u.UserID, &u.Username, &u.Password, &u.Rules)
	return u, err
}

func (r *userPGRepository) Insert(ctx context.Context, tenant string, data domain.User) error {
	db, err := r.dbRegistry.Postgres(tenant)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO users (username, pass, rules) VALUES ($1, $2, $3)`, data.Username, data.Password, data.Rules)
	return err
}

func (r *userPGRepository) Update(ctx context.Context, tenant string, username string, password *string, rules *string) error {
	db, err := r.dbRegistry.Postgres(tenant)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `UPDATE users SET pass=COALESCE($1, pass), rules=COALESCE($2, rules) WHERE username=$3`, password, rules, username)
	return err
}

type userMSRepository struct {
	dbRegistry *database.DBRegistry
}

func (r *userMSRepository) GetByUsername(ctx context.Context, tenant string, username string) (domain.User, error) {
	db, err := r.dbRegistry.MSSQL(tenant)
	if err != nil {
		return domain.User{}, err
	}

	var u domain.User
	err = db.QueryRowContext(ctx, `SELECT userid, username, pass, rules FROM users WHERE username=@username`, sql.Named("username", username)).Scan(&u.UserID, &u.Username, &u.Password, &u.Rules)
	return u, err
}

func (r *userMSRepository) Insert(ctx context.Context, tenant string, data domain.User) error {
	db, err := r.dbRegistry.MSSQL(tenant)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO users (username, pass, rules) VALUES (@username, @password, @rules)`, sql.Named("username", data.Username), sql.Named("password", data.Password), sql.Named("rules", data.Rules))
	return err
}

func (r *userMSRepository) Update(ctx context.Context, tenant string, username string, password *string, rules *string) error {
	db, err := r.dbRegistry.MSSQL(tenant)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `UPDATE users SET pass=COALESCE(@password, pass), rules=COALESCE(@rules, rules) WHERE username=@username`, sql.Named("password", password), sql.Named("rules", rules), sql.Named("username", username))
	return err
}

func NewUserRepositories(dbRegistry *database.DBRegistry) *UserRepositories {
	return &UserRepositories{
		PG: &userPGRepository{dbRegistry: dbRegistry},
		MS: &userMSRepository{dbRegistry: dbRegistry},
	}
}
