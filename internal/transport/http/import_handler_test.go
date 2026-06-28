package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/importlistings"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/ingestlisting"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/importjob"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/listing"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/rawlisting"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/user"
)

type stubRawListingRepo struct {
	mu        sync.Mutex
	saveCalls int
}

func (r *stubRawListingRepo) Save(ctx context.Context, l rawlisting.RawListing) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saveCalls++
	return "raw-1", nil
}

func (r *stubRawListingRepo) MarkProcessed(ctx context.Context, id string) error { return nil }

func (r *stubRawListingRepo) MarkFailed(ctx context.Context, id string, reason string) error {
	return nil
}

func (r *stubRawListingRepo) SaveCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveCalls
}

type stubImportJobRepo struct {
	mu          sync.Mutex
	createCalls int
	job         importjob.ImportJob
	done        chan struct{}
}

func newStubImportJobRepo() *stubImportJobRepo {
	return &stubImportJobRepo{done: make(chan struct{})}
}

func (r *stubImportJobRepo) waitDone(t *testing.T) {
	t.Helper()
	select {
	case <-r.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for import job to finish")
	}
}

func (r *stubImportJobRepo) Create(ctx context.Context, totalCount int) (importjob.ImportJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.createCalls++
	r.job = importjob.ImportJob{ID: "job-1", Status: importjob.StatusQueued, TotalCount: totalCount}
	return r.job, nil
}

func (r *stubImportJobRepo) GetByID(ctx context.Context, id string) (importjob.ImportJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.job, nil
}

func (r *stubImportJobRepo) List(ctx context.Context, limit int) ([]importjob.ImportJob, error) {
	return nil, nil
}

func (r *stubImportJobRepo) MarkProcessing(ctx context.Context, id string) error { return nil }

func (r *stubImportJobRepo) IncrementProcessed(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.job.ProcessedCount++
	return nil
}

func (r *stubImportJobRepo) IncrementFailed(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.job.FailedCount++
	return nil
}

func (r *stubImportJobRepo) MarkCompleted(ctx context.Context, id string) error {
	r.mu.Lock()
	r.job.Status = importjob.StatusCompleted
	r.mu.Unlock()
	close(r.done)
	return nil
}

func (r *stubImportJobRepo) MarkFailed(ctx context.Context, id string, reason string) error {
	r.mu.Lock()
	r.job.Status = importjob.StatusFailed
	msg := reason
	r.job.ErrorMessage = &msg
	r.mu.Unlock()
	close(r.done)
	return nil
}

func (r *stubImportJobRepo) MarkCancelled(ctx context.Context, id string) error { return nil }

func (r *stubImportJobRepo) CreateCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.createCalls
}

type stubIngester struct{}

func (s *stubIngester) Execute(ctx context.Context, input listing.Listing) (ingestlisting.Result, error) {
	return ingestlisting.Result{Listing: input}, nil
}

func newTestImportHandler(maxRows int, maxBodyBytes int64) (*ImportHandler, *stubRawListingRepo, *stubImportJobRepo) {
	rawRepo := &stubRawListingRepo{}
	jobRepo := newStubImportJobRepo()

	uc := importlistings.NewUseCase(rawRepo, jobRepo, &stubIngester{}, nil, maxRows, 4)

	return NewImportHandler(uc, maxBodyBytes), rawRepo, jobRepo
}

func authedImportRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/imports/clean-listings", strings.NewReader(body))
	ctx := contextWithCurrentUser(req.Context(), CurrentUser{ID: "user-1", Email: "admin@estatelink.dev", Role: user.RoleAdmin})
	return req.WithContext(ctx)
}

func TestImportCleanListingsAcceptsSmallImport(t *testing.T) {
	handler, rawRepo, jobRepo := newTestImportHandler(10, 1024*1024)

	body := `[{"source":"rightmove","property_id":"p1"},{"source":"rightmove","property_id":"p2"}]`
	req := authedImportRequest(body)
	rec := httptest.NewRecorder()

	handler.ImportCleanListings(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	if !strings.Contains(rec.Body.String(), "job-1") {
		t.Fatalf("expected response to include job id, got %s", rec.Body.String())
	}

	jobRepo.waitDone(t)

	if jobRepo.CreateCalls() != 1 {
		t.Fatalf("expected exactly one import job created, got %d", jobRepo.CreateCalls())
	}

	if rawRepo.SaveCalls() != 2 {
		t.Fatalf("expected 2 raw listings saved, got %d", rawRepo.SaveCalls())
	}
}

func TestImportCleanListingsRejectsTooManyRows(t *testing.T) {
	handler, rawRepo, jobRepo := newTestImportHandler(1, 1024*1024)

	body := `[{"source":"rightmove","property_id":"p1"},{"source":"rightmove","property_id":"p2"}]`
	req := authedImportRequest(body)
	rec := httptest.NewRecorder()

	handler.ImportCleanListings(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	if jobRepo.CreateCalls() != 0 {
		t.Fatalf("expected no import job created for oversized import, got %d", jobRepo.CreateCalls())
	}

	if rawRepo.SaveCalls() != 0 {
		t.Fatalf("expected no raw listings written for oversized import, got %d", rawRepo.SaveCalls())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected valid JSON response, got %v: %s", err, rec.Body.String())
	}

	if resp["error"] != "import too large" {
		t.Fatalf("expected error %q, got %v", "import too large", resp["error"])
	}

	if resp["maxRows"] != float64(1) {
		t.Fatalf("expected maxRows 1, got %v", resp["maxRows"])
	}

	if resp["receivedRows"] != float64(2) {
		t.Fatalf("expected receivedRows 2, got %v", resp["receivedRows"])
	}

	if resp["message"] == nil || resp["message"] == "" {
		t.Fatal("expected a non-empty message field")
	}
}

func TestImportCleanListingsRejectsOversizedBody(t *testing.T) {
	handler, rawRepo, jobRepo := newTestImportHandler(100, 16)

	var sb bytes.Buffer
	sb.WriteString(`[{"source":"rightmove","property_id":"p1"}]`)

	req := authedImportRequest(sb.String())
	rec := httptest.NewRecorder()

	handler.ImportCleanListings(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", rec.Code, rec.Body.String())
	}

	if jobRepo.CreateCalls() != 0 {
		t.Fatalf("expected no import job created for oversized body, got %d", jobRepo.CreateCalls())
	}

	if rawRepo.SaveCalls() != 0 {
		t.Fatalf("expected no raw listings written for oversized body, got %d", rawRepo.SaveCalls())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected valid JSON response, got %v: %s", err, rec.Body.String())
	}

	if resp["error"] != "request too large" {
		t.Fatalf("expected error %q, got %v", "request too large", resp["error"])
	}

	if resp["maxBytes"] != float64(16) {
		t.Fatalf("expected maxBytes 16, got %v", resp["maxBytes"])
	}
}

func TestImportJobDetailExposesUsefulFailureSummaryWhenAllRowsFail(t *testing.T) {
	handler, _, jobRepo := newTestImportHandler(10, 1024*1024)

	body := `[
		{"source":"zillow","property_id":"p1","price_val":3.62},
		{"source":"zillow","property_id":"p2","price_val":1.1}
	]`
	req := authedImportRequest(body)
	rec := httptest.NewRecorder()

	handler.ImportCleanListings(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	jobRepo.waitDone(t)

	detailReq := httptest.NewRequest(http.MethodGet, "/api/imports/job-1", nil)
	detailRec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobId", "job-1")
	detailReq = detailReq.WithContext(context.WithValue(detailReq.Context(), chi.RouteCtxKey, rctx))

	handler.GetImportJob(detailRec, detailReq)

	if detailRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", detailRec.Code, detailRec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(detailRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected valid JSON response, got %v: %s", err, detailRec.Body.String())
	}

	if resp["status"] != "failed" {
		t.Fatalf("expected status failed, got %v", resp["status"])
	}

	errorMessage, _ := resp["errorMessage"].(string)
	if errorMessage == "" {
		t.Fatal("expected a non-empty errorMessage for the frontend to display")
	}

	if !strings.Contains(errorMessage, "2 rows failed") {
		t.Fatalf("expected errorMessage to include the failure count, got %q", errorMessage)
	}

	if !strings.Contains(errorMessage, "price_val") {
		t.Fatalf("expected errorMessage to name the dominant failing field, got %q", errorMessage)
	}
}
