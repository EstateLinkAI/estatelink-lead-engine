package readleads

import (
	"context"
	"errors"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/lead"
)

var ErrLeadNotFound = errors.New("lead not found")

const (
	defaultLimit = 20
	maxLimit     = 100
)

type Repository interface {
	List(ctx context.Context, filters lead.ListFilters) (lead.ListResult, error)
	GetByID(ctx context.Context, id string) (*lead.ReadModel, error)
}

type UseCase struct {
	repo Repository
}

func NewUseCase(repo Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) List(ctx context.Context, filters lead.ListFilters) ([]lead.ReadModel, lead.Pagination, error) {
	if filters.Limit <= 0 {
		filters.Limit = defaultLimit
	}

	if filters.Limit > maxLimit {
		filters.Limit = maxLimit
	}

	if filters.Offset < 0 {
		filters.Offset = 0
	}

	result, err := uc.repo.List(ctx, filters)
	if err != nil {
		return nil, lead.Pagination{}, err
	}

	pagination := lead.Pagination{
		Limit:       filters.Limit,
		Offset:      filters.Offset,
		Total:       result.Total,
		Returned:    len(result.Items),
		HasNext:     filters.Offset+len(result.Items) < result.Total,
		HasPrevious: filters.Offset > 0,
	}

	return result.Items, pagination, nil
}

func (uc *UseCase) GetByID(ctx context.Context, id string) (*lead.ReadModel, error) {
	if id == "" {
		return nil, ErrLeadNotFound
	}

	return uc.repo.GetByID(ctx, id)
}
