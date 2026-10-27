package usecase

import (
	"context"

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
		return nil, domain.ErrSourceNotValid
	}
}

type OutboxUsecase struct {
	repoPG   domain.OutboxRepository
	repoMS   domain.OutboxRepository
	maxLimit int
}

func (u *OutboxUsecase) GetOutbox(ctx context.Context, tenant string, dbSource string, filter domain.OutboxFilter) ([]dto.OutboxItem, error) {
	if filter.Limit <= 0 {
		filter.Limit = domain.DefaultLimit
	} else if filter.Limit > u.maxLimit {
		filter.Limit = u.maxLimit
	}

	repo, err := pickOutboxRepo(dbSource, u.repoPG, u.repoMS)
	if err != nil {
		return nil, err
	}

	return repo.Get(ctx, tenant, filter)
}

func (u *OutboxUsecase) CreateOutbox(ctx context.Context, tenant string, dbSource string, data domain.Outbox) error {
	repo, err := pickOutboxRepo(dbSource, u.repoPG, u.repoMS)
	if err != nil {
		return err
	}
	if err := repo.Insert(ctx, tenant, data); err != nil {
		return err
	}
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
	return nil
}

func NewOutboxUsecase(repoPG domain.OutboxRepository, repoMS domain.OutboxRepository, maxLimit ...int) *OutboxUsecase {
	batas := domain.MaxLimit
	if len(maxLimit) > 0 {
		batas = maxLimit[0]
	}
	return &OutboxUsecase{repoPG: repoPG, repoMS: repoMS, maxLimit: batas}
}
