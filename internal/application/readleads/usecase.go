package readleads

import (
	"context"
	"errors"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/lead"
)

var ErrLeadNotFound = errors.New("lead not found")

type Repository interface {
	List(ctx context.Context, filters lead.ListFilters) ([]lead.ReadModel, error)
	GetByID(ctx context.Context, id string) (*lead.ReadModel, error)
}

type UseCase struct {
	repo Repository
}

func NewUseCase(repo Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) List(ctx context.Context, filters lead.ListFilters) ([]lead.ReadModel, error) {
	if filters.Limit <= 0 {
		filters.Limit = 20
	}

	if filters.Limit > 100 {
		filters.Limit = 100
	}

	if filters.Offset < 0 {
		filters.Offset = 0
	}

	return uc.repo.List(ctx, filters)
}

func (uc *UseCase) GetByID(ctx context.Context, id string) (*lead.ReadModel, error) {
	if id == "" {
		return nil, ErrLeadNotFound
	}

	return uc.repo.GetByID(ctx, id)
}
