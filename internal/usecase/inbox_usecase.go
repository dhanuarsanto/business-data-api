package usecase

import (
	"errors"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
)

type InboxUsecase interface {
	GetInbox(tenant string, dbSource string, filter domain.InboxFilter) ([]domain.Inbox, int64, error)
	InsertInbox(tenant string, dbSource string, data domain.Inbox) error
	UpdateInbox(tenant string, dbSource string, kode int64, req dto.UpdateInboxRequest) error
}

type inboxUsecase struct {
	repo domain.InboxRepository
}

func (u *inboxUsecase) GetInbox(tenant string, dbSource string, filter domain.InboxFilter) ([]domain.Inbox, int64, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 10
	}
	switch dbSource {
	case "postgres":
		return u.repo.GetInboxPG(tenant, filter)
	case "mssql":
		return u.repo.GetInboxMS(tenant, filter)
	default:
		return nil, 0, errors.New("sumber database tidak valid")
	}
}

func (u *inboxUsecase) InsertInbox(tenant string, dbSource string, data domain.Inbox) error {
	switch dbSource {
	case "postgres":
		return u.repo.InsertPG(tenant, data)
	case "mssql":
		return u.repo.InsertMS(tenant, data)
	default:
		return errors.New("sumber database tidak valid")
	}
}

func (u *inboxUsecase) UpdateInbox(tenant string, dbSource string, kode int64, req dto.UpdateInboxRequest) error {
	switch dbSource {
	case "postgres":
		return u.repo.UpdatePG(tenant, kode, req)
	case "mssql":
		return u.repo.UpdateMS(tenant, kode, req)
	default:
		return errors.New("sumber database tidak valid")
	}
}

func NewInboxUsecase(repo domain.InboxRepository) InboxUsecase {
	return &inboxUsecase{repo: repo}
}
