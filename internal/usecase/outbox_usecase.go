package usecase

import (
	"errors"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
)

type OutboxUsecase interface {
	GetOutbox(tenant string, dbSource string, filter domain.OutboxFilter) ([]domain.Outbox, int64, error)
	InsertOutbox(tenant string, dbSource string, data domain.Outbox) error
	UpdateOutbox(tenant string, dbSource string, kode int64, req dto.UpdateOutboxRequest) error
}

type outboxUsecase struct {
	repo domain.OutboxRepository
}

func (u *outboxUsecase) GetOutbox(tenant string, dbSource string, filter domain.OutboxFilter) ([]domain.Outbox, int64, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 10
	}
	switch dbSource {
	case "postgres":
		return u.repo.GetOutboxPG(tenant, filter)
	case "mssql":
		return u.repo.GetOutboxMS(tenant, filter)
	default:
		return nil, 0, errors.New("sumber database tidak valid")
	}
}

func (u *outboxUsecase) InsertOutbox(tenant string, dbSource string, data domain.Outbox) error {
	switch dbSource {
	case "postgres":
		return u.repo.InsertPG(tenant, data)
	case "mssql":
		return u.repo.InsertMS(tenant, data)
	default:
		return errors.New("sumber database tidak valid")
	}
}

func (u *outboxUsecase) UpdateOutbox(tenant string, dbSource string, kode int64, req dto.UpdateOutboxRequest) error {
	switch dbSource {
	case "postgres":
		return u.repo.UpdatePG(tenant, kode, req)
	case "mssql":
		return u.repo.UpdateMS(tenant, kode, req)
	default:
		return errors.New("sumber database tidak valid")
	}
}

func NewOutboxUsecase(repo domain.OutboxRepository) OutboxUsecase {
	return &outboxUsecase{repo: repo}
}
