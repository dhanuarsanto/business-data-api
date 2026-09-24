package usecase

import (
	"context"
	"errors"
	"strings"

	"go.internal/business-data-api/internal/domain"
)

var ErrInvalidInput = errors.New("input tidak valid")

func pickUserRepo(source string, pg, ms domain.UserRepository) (domain.UserRepository, error) {
	switch source {
	case "postgres":
		return pg, nil
	case "mssql":
		return ms, nil
	default:
		return nil, errors.New("sumber database tidak valid")
	}
}

type AuthUsecase struct {
	userRepoPG domain.UserRepository
	userRepoMS domain.UserRepository
}

func (u *AuthUsecase) Login(ctx context.Context, tenant string, dbSource string, username string, password string) (domain.User, error) {
	repo, err := pickUserRepo(dbSource, u.userRepoPG, u.userRepoMS)
	if err != nil {
		return domain.User{}, err
	}

	user, err := repo.GetByUsername(ctx, tenant, username)
	if err != nil {
		return user, errors.New("user tidak ditemukan")
	}

	if user.Password != password {
		return user, errors.New("password salah")
	}

	return user, nil
}

func (u *AuthUsecase) CreateUser(ctx context.Context, tenant string, dbSource string, data domain.User) error {
	if strings.TrimSpace(data.Username) == "" || strings.TrimSpace(data.Password) == "" || strings.TrimSpace(data.Rules) == "" {
		return ErrInvalidInput
	}
	repo, err := pickUserRepo(dbSource, u.userRepoPG, u.userRepoMS)
	if err != nil {
		return err
	}
	return repo.Insert(ctx, tenant, data)
}

func (u *AuthUsecase) UpdateUser(ctx context.Context, tenant string, dbSource string, username string, password *string, rules *string) error {
	if password != nil && *password == "" {
		return ErrInvalidInput
	}
	if rules != nil && *rules == "" {
		return ErrInvalidInput
	}
	repo, err := pickUserRepo(dbSource, u.userRepoPG, u.userRepoMS)
	if err != nil {
		return err
	}
	return repo.Update(ctx, tenant, username, password, rules)
}

func NewAuthUsecase(userRepoPG domain.UserRepository, userRepoMS domain.UserRepository) *AuthUsecase {
	return &AuthUsecase{userRepoPG: userRepoPG, userRepoMS: userRepoMS}
}
