package usecase

import (
	"context"
	"errors"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
)

func pickInboxRepo(source string, pg, ms domain.InboxRepository) (domain.InboxRepository, error) {
	switch source {
	case "postgres":
		return pg, nil
	case "mssql":
		return ms, nil
	default:
		return nil, errors.New("sumber database tidak valid")
	}
}

type InboxUsecase struct {
	repoPG domain.InboxRepository
	repoMS domain.InboxRepository
}

func (u *InboxUsecase) GetInbox(ctx context.Context, tenant string, dbSource string, filter domain.InboxFilter) ([]domain.Inbox, bool, error) {
	if filter.PageSize <= 0 || filter.PageSize > 500 {
		filter.PageSize = 10
	}
	if filter.LimitTotal != nil {
		if *filter.LimitTotal <= 0 {
			filter.LimitTotal = nil
		} else if *filter.LimitTotal > 100000 {
			v := 100000
			filter.LimitTotal = &v
		}
	}

	repo, err := pickInboxRepo(dbSource, u.repoPG, u.repoMS)
	if err != nil {
		return nil, false, err
	}

	if filter.LimitTotal != nil {
		key := filterCacheKey("inbox", dbSource, tenant, *filter.LimitTotal, inboxFilterID(filter))
		if lb, ok := cacheGet(key); ok {
			filter.LowerBound = lb
		} else {
			lb, err := repo.LowerBound(ctx, tenant, filter)
			if err != nil {
				return nil, false, err
			}
			filter.LowerBound = lb
			cacheSet(key, lb)
		}
	}

	data, hasNext, err := repo.Get(ctx, tenant, filter)
	if err != nil {
		return nil, false, err
	}
	return data, hasNext, nil
}

func (u *InboxUsecase) CreateInbox(ctx context.Context, tenant string, dbSource string, data domain.Inbox) error {
	repo, err := pickInboxRepo(dbSource, u.repoPG, u.repoMS)
	if err != nil {
		return err
	}
	return repo.Insert(ctx, tenant, data)
}

func (u *InboxUsecase) UpdateInbox(ctx context.Context, tenant string, dbSource string, kode int64, req dto.UpdateInboxRequest) error {
	repo, err := pickInboxRepo(dbSource, u.repoPG, u.repoMS)
	if err != nil {
		return err
	}
	return repo.Update(ctx, tenant, kode, req)
}

func NewInboxUsecase(repoPG domain.InboxRepository, repoMS domain.InboxRepository) *InboxUsecase {
	return &InboxUsecase{repoPG: repoPG, repoMS: repoMS}
}