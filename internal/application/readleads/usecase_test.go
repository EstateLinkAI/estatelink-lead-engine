package readleads

import (
	"context"
	"testing"

	"github.com/nath070707/estatelink-lead-engine/internal/domain/lead"
)

type fakeRepository struct {
	receivedFilters lead.ListFilters
}

func (f *fakeRepository) List(ctx context.Context, filters lead.ListFilters) ([]lead.ReadModel, error) {
	f.receivedFilters = filters
	return []lead.ReadModel{}, nil
}

func (f *fakeRepository) GetByID(ctx context.Context, id string) (*lead.ReadModel, error) {
	return &lead.ReadModel{ID: 1}, nil
}

func TestListAppliesDefaultPagination(t *testing.T) {
	repo := &fakeRepository{}
	uc := NewUseCase(repo)

	_, err := uc.List(context.Background(), lead.ListFilters{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.receivedFilters.Limit != 20 {
		t.Fatalf("expected default limit 20, got %d", repo.receivedFilters.Limit)
	}

	if repo.receivedFilters.Offset != 0 {
		t.Fatalf("expected default offset 0, got %d", repo.receivedFilters.Offset)
	}
}

func TestListCapsLimitAt100(t *testing.T) {
	repo := &fakeRepository{}
	uc := NewUseCase(repo)

	_, err := uc.List(context.Background(), lead.ListFilters{
		Limit: 250,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.receivedFilters.Limit != 100 {
		t.Fatalf("expected limit capped at 100, got %d", repo.receivedFilters.Limit)
	}
}

func TestGetByIDReturnsNotFoundForEmptyID(t *testing.T) {
	repo := &fakeRepository{}
	uc := NewUseCase(repo)

	_, err := uc.GetByID(context.Background(), "")
	if err != ErrLeadNotFound {
		t.Fatalf("expected ErrLeadNotFound, got %v", err)
	}
}
