package usecase

import (
	"context"
	"errors"

	"go.internal/business-data-api/internal/domain"
)

type AuthUsecase interface {
	Login(ctx context.Context, tenant string, dbSource string, username string, password string) (domain.User, error)
	CreateUser(ctx context.Context, tenant string, dbSource string, data domain.User) error
	UpdateUser(ctx context.Context, tenant string, dbSource string, username string, data domain.User) error
}

type authUsecase struct {
	userRepo domain.UserRepository
}

func (u *authUsecase) Login(ctx context.Context, tenant string, dbSource string, username string, password string) (domain.User, error) {
	var user domain.User
	var err error

	switch dbSource {
	case "postgres":
		user, err = u.userRepo.GetByUsernamePG(ctx, tenant, username)
	case "mssql":
		user, err = u.userRepo.GetByUsernameMS(ctx, tenant, username)
	default:
		return user, errors.New("sumber database tidak valid")
	}

	if err != nil {
		return user, errors.New("user tidak ditemukan")
	}

	if user.Password != password {
		return user, errors.New("password salah")
	}

	return user, nil
}

func (u *authUsecase) CreateUser(ctx context.Context, tenant string, dbSource string, data domain.User) error {
	switch dbSource {
	case "postgres":
		return u.userRepo.InsertPG(ctx, tenant, data)
	case "mssql":
		return u.userRepo.InsertMS(ctx, tenant, data)
	default:
		return errors.New("sumber database tidak valid")
	}
}

func (u *authUsecase) UpdateUser(ctx context.Context, tenant string, dbSource string, username string, data domain.User) error {
	switch dbSource {
	case "postgres":
		return u.userRepo.UpdatePG(ctx, tenant, username, data)
	case "mssql":
		return u.userRepo.UpdateMS(ctx, tenant, username, data)
	default:
		return errors.New("sumber database tidak valid")
	}
}

func NewAuthUsecase(userRepo domain.UserRepository) AuthUsecase {
	return &authUsecase{userRepo: userRepo}
}
