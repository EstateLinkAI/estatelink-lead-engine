package http

import (
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/readleads"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/lead"
)

type fakeLeadRepository struct {
	receivedFilters lead.ListFilters
	items           []lead.ReadModel
	total           int
}

func (f *fakeLeadRepository) List(ctx context.Context, filters lead.ListFilters) (lead.ListResult, error) {
	f.receivedFilters = filters

	items := f.items
	if items == nil {
		items = []lead.ReadModel{}
	}

	return lead.ListResult{Items: items, Total: f.total}, nil
}

func (f *fakeLeadRepository) GetByID(ctx context.Context, id string) (*lead.ReadModel, error) {
	return &lead.ReadModel{ID: 1}, nil
}

func decodeLeadListResponse(t *testing.T, rec *httptest.ResponseRecorder) leadListResponse {
	t.Helper()

	var body leadListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	return body
}

func TestLeadHandlerListDefaultPagination(t *testing.T) {
	repo := &fakeLeadRepository{total: 368000}
	handler := NewLeadHandler(readleads.NewUseCase(repo))

	req := httptest.NewRequest(nethttp.MethodGet, "/api/leads", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if repo.receivedFilters.Limit != 20 {
		t.Fatalf("expected default limit 20, got %d", repo.receivedFilters.Limit)
	}

	if repo.receivedFilters.Offset != 0 {
		t.Fatalf("expected default offset 0, got %d", repo.receivedFilters.Offset)
	}

	body := decodeLeadListResponse(t, rec)

	if body.Pagination.Limit != 20 || body.Pagination.Offset != 0 {
		t.Fatalf("expected pagination limit=20 offset=0, got %+v", body.Pagination)
	}
}

func TestLeadHandlerListCustomLimitAndOffset(t *testing.T) {
	repo := &fakeLeadRepository{total: 368000}
	handler := NewLeadHandler(readleads.NewUseCase(repo))

	req := httptest.NewRequest(nethttp.MethodGet, "/api/leads?limit=20&offset=20", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if repo.receivedFilters.Limit != 20 {
		t.Fatalf("expected limit 20, got %d", repo.receivedFilters.Limit)
	}

	if repo.receivedFilters.Offset != 20 {
		t.Fatalf("expected offset 20, got %d", repo.receivedFilters.Offset)
	}
}

func TestLeadHandlerListClampsLimitAboveMax(t *testing.T) {
	repo := &fakeLeadRepository{total: 5}
	handler := NewLeadHandler(readleads.NewUseCase(repo))

	req := httptest.NewRequest(nethttp.MethodGet, "/api/leads?limit=5000", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if repo.receivedFilters.Limit != 100 {
		t.Fatalf("expected limit clamped to 100, got %d", repo.receivedFilters.Limit)
	}

	body := decodeLeadListResponse(t, rec)
	if body.Pagination.Limit != 100 {
		t.Fatalf("expected pagination limit clamped to 100, got %d", body.Pagination.Limit)
	}
}

func TestLeadHandlerListReturnsPaginationMetadata(t *testing.T) {
	repo := &fakeLeadRepository{
		items: []lead.ReadModel{
			{ID: 1, Title: "3 Bed Terraced House in Manchester", Score: 83, Grade: "A"},
		},
		total: 368000,
	}
	handler := NewLeadHandler(readleads.NewUseCase(repo))

	req := httptest.NewRequest(nethttp.MethodGet, "/api/leads?limit=20&offset=0", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	body := decodeLeadListResponse(t, rec)

	if len(body.Data) != 1 {
		t.Fatalf("expected 1 lead in data, got %d", len(body.Data))
	}

	if body.Pagination.Total != 368000 {
		t.Fatalf("expected total 368000, got %d", body.Pagination.Total)
	}

	if body.Pagination.Returned != 1 {
		t.Fatalf("expected returned 1, got %d", body.Pagination.Returned)
	}
}

func TestLeadHandlerListHasNextAndHasPrevious(t *testing.T) {
	repo := &fakeLeadRepository{
		items: make([]lead.ReadModel, 20),
		total: 368000,
	}
	handler := NewLeadHandler(readleads.NewUseCase(repo))

	req := httptest.NewRequest(nethttp.MethodGet, "/api/leads?limit=20&offset=20", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	body := decodeLeadListResponse(t, rec)

	if !body.Pagination.HasNext {
		t.Fatalf("expected hasNext true, got %+v", body.Pagination)
	}

	if !body.Pagination.HasPrevious {
		t.Fatalf("expected hasPrevious true, got %+v", body.Pagination)
	}
}

func TestLeadHandlerListPreservesFiltersWithPagination(t *testing.T) {
	repo := &fakeLeadRepository{total: 12}
	handler := NewLeadHandler(readleads.NewUseCase(repo))

	req := httptest.NewRequest(nethttp.MethodGet, "/api/leads?city=Manchester&minScore=80&limit=20&offset=0", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if repo.receivedFilters.City != "Manchester" {
		t.Fatalf("expected city filter Manchester, got %q", repo.receivedFilters.City)
	}

	if repo.receivedFilters.MinScore == nil || *repo.receivedFilters.MinScore != 80 {
		t.Fatalf("expected minScore filter 80, got %v", repo.receivedFilters.MinScore)
	}

	body := decodeLeadListResponse(t, rec)
	if body.Pagination.Total != 12 {
		t.Fatalf("expected filtered total 12, got %d", body.Pagination.Total)
	}
}

func TestLeadHandlerListRejectsInvalidMinScore(t *testing.T) {
	repo := &fakeLeadRepository{}
	handler := NewLeadHandler(readleads.NewUseCase(repo))

	req := httptest.NewRequest(nethttp.MethodGet, "/api/leads?minScore=not-a-number", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != nethttp.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
