package usecase

import (
	"context"
	"errors"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
)

func pickResellerRepo(source string, pg, ms domain.ResellerRepository) (domain.ResellerRepository, error) {
	switch source {
	case "postgres":
		return pg, nil
	case "mssql":
		return ms, nil
	default:
		return nil, errors.New("sumber database tidak valid")
	}
}

type MasterUsecase struct {
	repoPG domain.ResellerRepository
	repoMS domain.ResellerRepository
}

func (u *MasterUsecase) ListResellerForDropdown(ctx context.Context, tenant string, dbSource string) ([]dto.ResellerDropdown, error) {
	repo, err := pickResellerRepo(dbSource, u.repoPG, u.repoMS)
	if err != nil {
		return nil, err
	}
	return repo.ListForDropdown(ctx, tenant)
}

func NewMasterUsecase(repoPG domain.ResellerRepository, repoMS domain.ResellerRepository) *MasterUsecase {
	return &MasterUsecase{repoPG: repoPG, repoMS: repoMS}
}
