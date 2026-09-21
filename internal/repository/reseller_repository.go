package repository

import (
	"context"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
	"go.internal/business-data-api/pkg/database"
)

type ResellerRepositories struct {
	PG domain.ResellerRepository
	MS domain.ResellerRepository
}

type resellerPGRepository struct {
	dbRegistry *database.DBRegistry
}

func (r *resellerPGRepository) ListForDropdown(ctx context.Context, tenant string) ([]dto.ResellerDropdown, error) {
	db, err := r.dbRegistry.Postgres(tenant)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(ctx, `SELECT kode, nama FROM reseller ORDER BY nama`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []dto.ResellerDropdown
	for rows.Next() {
		var r dto.ResellerDropdown
		if err := rows.Scan(&r.Kode, &r.Nama); err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, rows.Err()
}

type resellerMSRepository struct {
	dbRegistry *database.DBRegistry
}

func (r *resellerMSRepository) ListForDropdown(ctx context.Context, tenant string) ([]dto.ResellerDropdown, error) {
	db, err := r.dbRegistry.MSSQL(tenant)
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `SELECT kode, nama FROM reseller ORDER BY nama`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []dto.ResellerDropdown
	for rows.Next() {
		var r dto.ResellerDropdown
		if err := rows.Scan(&r.Kode, &r.Nama); err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, rows.Err()
}

func NewResellerRepositories(dbRegistry *database.DBRegistry) *ResellerRepositories {
	return &ResellerRepositories{
		PG: &resellerPGRepository{dbRegistry: dbRegistry},
		MS: &resellerMSRepository{dbRegistry: dbRegistry},
	}
}