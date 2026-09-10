package usecase

import (
	"errors"

	"go.internal/business-data-api/internal/domain"
)

type AuthUsecase interface {
	Login(tenant string, dbSource string, username string, password string) (domain.User, error)
	CreateUser(tenant string, dbSource string, data domain.User) error
	UpdateUser(tenant string, dbSource string, username string, data domain.User) error
}

type authUsecase struct {
	userRepo domain.UserRepository
}

func (u *authUsecase) Login(tenant string, dbSource string, username string, password string) (domain.User, error) {
	var user domain.User
	var err error

	switch dbSource {
	case "postgres":
		user, err = u.userRepo.GetByUsernamePG(tenant, username)
	case "mssql":
		user, err = u.userRepo.GetByUsernameMS(tenant, username)
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

func (u *authUsecase) CreateUser(tenant string, dbSource string, data domain.User) error {
	switch dbSource {
	case "postgres":
		return u.userRepo.InsertPG(tenant, data)
	case "mssql":
		return u.userRepo.InsertMS(tenant, data)
	default:
		return errors.New("sumber database tidak valid")
	}
}

func (u *authUsecase) UpdateUser(tenant string, dbSource string, username string, data domain.User) error {
	switch dbSource {
	case "postgres":
		return u.userRepo.UpdatePG(tenant, username, data)
	case "mssql":
		return u.userRepo.UpdateMS(tenant, username, data)
	default:
		return errors.New("sumber database tidak valid")
	}
}

func NewAuthUsecase(userRepo domain.UserRepository) AuthUsecase {
	return &authUsecase{userRepo: userRepo}
}
