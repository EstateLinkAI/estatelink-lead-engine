package logactivity

import (
	"context"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/activitylog"
)

type Service struct {
	Repo ActivityLogRepositoryInterface
}

type ActivityLogRepositoryInterface interface {
	Insert(ctx context.Context, log activitylog.ActivityLog) error
	List(ctx context.Context, limit, offset int) ([]activitylog.ActivityLog, error)
	GetByID(ctx context.Context, id int64) (activitylog.ActivityLog, error)
}

func (s *Service) Log(ctx context.Context, log activitylog.ActivityLog) error {
	return s.Repo.Insert(ctx, log)
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]activitylog.ActivityLog, error) {
	return s.Repo.List(ctx, limit, offset)
}

func (s *Service) GetByID(ctx context.Context, id int64) (activitylog.ActivityLog, error) {
	return s.Repo.GetByID(ctx, id)
}