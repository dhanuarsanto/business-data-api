package domain

import (
	"context"

	"go.internal/business-data-api/internal/dto"
)

type ResellerRepository interface {
	ListForDropdown(ctx context.Context, tenant string) ([]dto.ResellerDropdown, error)
}
