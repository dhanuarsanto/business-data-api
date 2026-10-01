package usecase

import (
	"context"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
)

func pickInboxRepo(source string, pg, ms domain.InboxRepository) (domain.InboxRepository, error) {
	switch source {
	case domain.SourcePostgres:
		return pg, nil
	case domain.SourceMSSQL:
		return ms, nil
	default:
		return nil, domain.ErrSourceNotValid
	}
}

type InboxUsecase struct {
	repoPG domain.InboxRepository
	repoMS domain.InboxRepository
}

func (u *InboxUsecase) GetInbox(ctx context.Context, tenant string, dbSource string, filter domain.InboxFilter) ([]dto.InboxItem, error) {
	if filter.Limit <= 0 {
		filter.Limit = domain.DefaultLimit
	} else if filter.Limit > domain.MaxLimit {
		filter.Limit = domain.MaxLimit
	}

	repo, err := pickInboxRepo(dbSource, u.repoPG, u.repoMS)
	if err != nil {
		return nil, err
	}

	return repo.Get(ctx, tenant, filter)
}

func (u *InboxUsecase) CreateInbox(ctx context.Context, tenant string, dbSource string, data domain.Inbox) error {
	repo, err := pickInboxRepo(dbSource, u.repoPG, u.repoMS)
	if err != nil {
		return err
	}
	if err := repo.Insert(ctx, tenant, data); err != nil {
		return err
	}
	return nil
}

func (u *InboxUsecase) UpdateInbox(ctx context.Context, tenant string, dbSource string, kode int64, req dto.UpdateInboxRequest) error {
	repo, err := pickInboxRepo(dbSource, u.repoPG, u.repoMS)
	if err != nil {
		return err
	}
	if err := repo.Update(ctx, tenant, kode, req); err != nil {
		return err
	}
	return nil
}

func NewInboxUsecase(repoPG domain.InboxRepository, repoMS domain.InboxRepository) *InboxUsecase {
	return &InboxUsecase{repoPG: repoPG, repoMS: repoMS}
}
