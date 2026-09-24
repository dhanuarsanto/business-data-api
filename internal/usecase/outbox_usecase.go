package usecase

import (
	"context"
	"errors"
	"fmt"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
)

func pickOutboxRepo(source string, pg, ms domain.OutboxRepository) (domain.OutboxRepository, error) {
	switch source {
	case domain.SourcePostgres:
		return pg, nil
	case domain.SourceMSSQL:
		return ms, nil
	default:
		return nil, errors.New("sumber database tidak valid")
	}
}

type OutboxUsecase struct {
	repoPG domain.OutboxRepository
	repoMS domain.OutboxRepository
}

func (u *OutboxUsecase) GetOutbox(ctx context.Context, tenant string, dbSource string, filter domain.OutboxFilter) ([]domain.Outbox, bool, error) {
	if filter.PageSize <= 0 {
		filter.PageSize = domain.DefaultPageSize
	} else if filter.PageSize > domain.MaxPageSize {
		filter.PageSize = domain.MaxPageSize
	}
	if filter.LimitTotal != nil {
		if *filter.LimitTotal <= 0 {
			filter.LimitTotal = nil
		} else if *filter.LimitTotal > domain.MaxLimitTotal {
			v := domain.MaxLimitTotal
			filter.LimitTotal = &v
		}
	}

	repo, err := pickOutboxRepo(dbSource, u.repoPG, u.repoMS)
	if err != nil {
		return nil, false, err
	}

	if filter.LimitTotal != nil {
		key := filterCacheKey("outbox", dbSource, tenant, *filter.LimitTotal, outboxFilterID(filter))
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

func (u *OutboxUsecase) CreateOutbox(ctx context.Context, tenant string, dbSource string, data domain.Outbox) error {
	repo, err := pickOutboxRepo(dbSource, u.repoPG, u.repoMS)
	if err != nil {
		return err
	}
	if err := repo.Insert(ctx, tenant, data); err != nil {
		return err
	}
	cacheInvalidatePrefix(fmt.Sprintf("outbox|%s|%s", dbSource, tenant))
	return nil
}

func (u *OutboxUsecase) UpdateOutbox(ctx context.Context, tenant string, dbSource string, kode int64, req dto.UpdateOutboxRequest) error {
	repo, err := pickOutboxRepo(dbSource, u.repoPG, u.repoMS)
	if err != nil {
		return err
	}
	if err := repo.Update(ctx, tenant, kode, req); err != nil {
		return err
	}
	cacheInvalidatePrefix(fmt.Sprintf("outbox|%s|%s", dbSource, tenant))
	return nil
}

func NewOutboxUsecase(repoPG domain.OutboxRepository, repoMS domain.OutboxRepository) *OutboxUsecase {
	return &OutboxUsecase{repoPG: repoPG, repoMS: repoMS}
}
