package usecase

import (
	"context"
	"errors"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
)

type InboxUsecase interface {
	GetInbox(ctx context.Context, tenant string, dbSource string, filter domain.InboxFilter) ([]domain.Inbox, int, bool, error)
	InsertInbox(ctx context.Context, tenant string, dbSource string, data domain.Inbox) error
	UpdateInbox(ctx context.Context, tenant string, dbSource string, kode int64, req dto.UpdateInboxRequest) error
}

type inboxUsecase struct {
	repo domain.InboxRepository
}

func (u *inboxUsecase) GetInbox(ctx context.Context, tenant string, dbSource string, filter domain.InboxFilter) ([]domain.Inbox, int, bool, error) {
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 10
	}
	switch dbSource {
	case "postgres":
		return u.repo.GetInboxPG(ctx, tenant, filter)
	case "mssql":
		return u.repo.GetInboxMS(ctx, tenant, filter)
	default:
		return nil, 0, false, errors.New("sumber database tidak valid")
	}
}

func (u *inboxUsecase) InsertInbox(ctx context.Context, tenant string, dbSource string, data domain.Inbox) error {
	switch dbSource {
	case "postgres":
		return u.repo.InsertPG(ctx, tenant, data)
	case "mssql":
		return u.repo.InsertMS(ctx, tenant, data)
	default:
		return errors.New("sumber database tidak valid")
	}
}

func (u *inboxUsecase) UpdateInbox(ctx context.Context, tenant string, dbSource string, kode int64, req dto.UpdateInboxRequest) error {
	switch dbSource {
	case "postgres":
		return u.repo.UpdatePG(ctx, tenant, kode, req)
	case "mssql":
		return u.repo.UpdateMS(ctx, tenant, kode, req)
	default:
		return errors.New("sumber database tidak valid")
	}
}

func NewInboxUsecase(repo domain.InboxRepository) InboxUsecase {
	return &inboxUsecase{repo: repo}
}
