package usecase

import (
	"context"
	"errors"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
)

type OutboxUsecase interface {
	GetOutbox(ctx context.Context, tenant string, dbSource string, filter domain.OutboxFilter) ([]domain.Outbox, int, bool, error)
	InsertOutbox(ctx context.Context, tenant string, dbSource string, data domain.Outbox) error
	UpdateOutbox(ctx context.Context, tenant string, dbSource string, kode int64, req dto.UpdateOutboxRequest) error
}

type outboxUsecase struct {
	repo domain.OutboxRepository
}

func (u *outboxUsecase) GetOutbox(ctx context.Context, tenant string, dbSource string, filter domain.OutboxFilter) ([]domain.Outbox, int, bool, error) {
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 10
	}
	switch dbSource {
	case "postgres":
		return u.repo.GetOutboxPG(ctx, tenant, filter)
	case "mssql":
		return u.repo.GetOutboxMS(ctx, tenant, filter)
	default:
		return nil, 0, false, errors.New("sumber database tidak valid")
	}
}

func (u *outboxUsecase) InsertOutbox(ctx context.Context, tenant string, dbSource string, data domain.Outbox) error {
	switch dbSource {
	case "postgres":
		return u.repo.InsertPG(ctx, tenant, data)
	case "mssql":
		return u.repo.InsertMS(ctx, tenant, data)
	default:
		return errors.New("sumber database tidak valid")
	}
}

func (u *outboxUsecase) UpdateOutbox(ctx context.Context, tenant string, dbSource string, kode int64, req dto.UpdateOutboxRequest) error {
	switch dbSource {
	case "postgres":
		return u.repo.UpdatePG(ctx, tenant, kode, req)
	case "mssql":
		return u.repo.UpdateMS(ctx, tenant, kode, req)
	default:
		return errors.New("sumber database tidak valid")
	}
}

func NewOutboxUsecase(repo domain.OutboxRepository) OutboxUsecase {
	return &outboxUsecase{repo: repo}
}
