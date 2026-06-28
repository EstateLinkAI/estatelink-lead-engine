package readleads

import (
	"context"
	"testing"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/lead"
)

type fakeRepository struct {
	receivedFilters lead.ListFilters
	items           []lead.ReadModel
	total           int
}

func (f *fakeRepository) List(ctx context.Context, filters lead.ListFilters) (lead.ListResult, error) {
	f.receivedFilters = filters

	items := f.items
	if items == nil {
		items = []lead.ReadModel{}
	}

	return lead.ListResult{Items: items, Total: f.total}, nil
}

func (f *fakeRepository) GetByID(ctx context.Context, id string) (*lead.ReadModel, error) {
	return &lead.ReadModel{ID: 1}, nil
}

func TestListAppliesDefaultPagination(t *testing.T) {
	repo := &fakeRepository{}
	uc := NewUseCase(repo)

	_, _, err := uc.List(context.Background(), lead.ListFilters{})
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

func TestListAppliesCustomLimitAndOffset(t *testing.T) {
	repo := &fakeRepository{}
	uc := NewUseCase(repo)

	_, _, err := uc.List(context.Background(), lead.ListFilters{Limit: 10, Offset: 30})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.receivedFilters.Limit != 10 {
		t.Fatalf("expected limit 10, got %d", repo.receivedFilters.Limit)
	}

	if repo.receivedFilters.Offset != 30 {
		t.Fatalf("expected offset 30, got %d", repo.receivedFilters.Offset)
	}
}

func TestListCapsLimitAt100(t *testing.T) {
	repo := &fakeRepository{}
	uc := NewUseCase(repo)

	_, pagination, err := uc.List(context.Background(), lead.ListFilters{
		Limit: 250,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.receivedFilters.Limit != 100 {
		t.Fatalf("expected limit capped at 100, got %d", repo.receivedFilters.Limit)
	}

	if pagination.Limit != 100 {
		t.Fatalf("expected pagination limit capped at 100, got %d", pagination.Limit)
	}
}

func TestListNormalizesNegativeOffset(t *testing.T) {
	repo := &fakeRepository{}
	uc := NewUseCase(repo)

	_, _, err := uc.List(context.Background(), lead.ListFilters{Offset: -5})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.receivedFilters.Offset != 0 {
		t.Fatalf("expected offset normalized to 0, got %d", repo.receivedFilters.Offset)
	}
}

func TestListReturnsPaginationMetadata(t *testing.T) {
	repo := &fakeRepository{
		items: []lead.ReadModel{{ID: 1}, {ID: 2}},
		total: 368000,
	}
	uc := NewUseCase(repo)

	items, pagination, err := uc.List(context.Background(), lead.ListFilters{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if pagination.Total != 368000 {
		t.Fatalf("expected total 368000, got %d", pagination.Total)
	}

	if pagination.Returned != 2 {
		t.Fatalf("expected returned 2, got %d", pagination.Returned)
	}

	if pagination.Limit != 20 || pagination.Offset != 0 {
		t.Fatalf("expected limit/offset to be echoed back, got limit=%d offset=%d", pagination.Limit, pagination.Offset)
	}
}

func TestListHasNextTrueWhenMoreRecordsRemain(t *testing.T) {
	repo := &fakeRepository{
		items: make([]lead.ReadModel, 20),
		total: 100,
	}
	uc := NewUseCase(repo)

	_, pagination, err := uc.List(context.Background(), lead.ListFilters{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !pagination.HasNext {
		t.Fatalf("expected hasNext true when offset+returned < total")
	}

	if pagination.HasPrevious {
		t.Fatalf("expected hasPrevious false at offset 0")
	}
}

func TestListHasNextFalseOnLastPage(t *testing.T) {
	repo := &fakeRepository{
		items: make([]lead.ReadModel, 10),
		total: 100,
	}
	uc := NewUseCase(repo)

	_, pagination, err := uc.List(context.Background(), lead.ListFilters{Limit: 20, Offset: 90})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if pagination.HasNext {
		t.Fatalf("expected hasNext false on last page")
	}

	if !pagination.HasPrevious {
		t.Fatalf("expected hasPrevious true when offset > 0")
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
