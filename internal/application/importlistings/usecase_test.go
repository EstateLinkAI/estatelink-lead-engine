package importlistings

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/ingestlisting"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/importjob"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/listing"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/rawlisting"
)

type fakeRawListingRepo struct {
	mu        sync.Mutex
	saveCalls int
	nextID    int
}

func (r *fakeRawListingRepo) Save(ctx context.Context, l rawlisting.RawListing) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saveCalls++
	r.nextID++
	return string(rune('a' + r.nextID)), nil
}

func (r *fakeRawListingRepo) MarkProcessed(ctx context.Context, id string) error { return nil }

func (r *fakeRawListingRepo) MarkFailed(ctx context.Context, id string, reason string) error {
	return nil
}

func (r *fakeRawListingRepo) SaveCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveCalls
}

type fakeImportJobRepo struct {
	mu          sync.Mutex
	createCalls int
	job         importjob.ImportJob
	done        chan struct{}
}

func newFakeImportJobRepo() *fakeImportJobRepo {
	return &fakeImportJobRepo{done: make(chan struct{})}
}

func (r *fakeImportJobRepo) Create(ctx context.Context, totalCount int) (importjob.ImportJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.createCalls++
	r.job = importjob.ImportJob{
		ID:         "job-1",
		Status:     importjob.StatusQueued,
		TotalCount: totalCount,
	}
	return r.job, nil
}

func (r *fakeImportJobRepo) GetByID(ctx context.Context, id string) (importjob.ImportJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.job, nil
}

func (r *fakeImportJobRepo) List(ctx context.Context, limit int) ([]importjob.ImportJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return []importjob.ImportJob{r.job}, nil
}

func (r *fakeImportJobRepo) MarkProcessing(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.job.Status = importjob.StatusProcessing
	return nil
}

func (r *fakeImportJobRepo) IncrementProcessed(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.job.ProcessedCount++
	return nil
}

func (r *fakeImportJobRepo) IncrementFailed(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.job.FailedCount++
	return nil
}

func (r *fakeImportJobRepo) MarkCompleted(ctx context.Context, id string) error {
	r.mu.Lock()
	r.job.Status = importjob.StatusCompleted
	r.mu.Unlock()
	close(r.done)
	return nil
}

func (r *fakeImportJobRepo) MarkFailed(ctx context.Context, id string, reason string) error {
	r.mu.Lock()
	r.job.Status = importjob.StatusFailed
	msg := reason
	r.job.ErrorMessage = &msg
	r.mu.Unlock()
	close(r.done)
	return nil
}

func (r *fakeImportJobRepo) MarkCancelled(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.job.Status = importjob.StatusCancelled
	return nil
}

func (r *fakeImportJobRepo) CreateCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.createCalls
}

func (r *fakeImportJobRepo) waitDone(t *testing.T) {
	t.Helper()
	select {
	case <-r.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for import job to finish")
	}
}

type fakeIngester struct{}

func (f *fakeIngester) Execute(ctx context.Context, input listing.Listing) (ingestlisting.Result, error) {
	return ingestlisting.Result{Listing: input}, nil
}

func validCleanListingPayload(n int) []json.RawMessage {
	payload := make([]json.RawMessage, 0, n)
	for i := 0; i < n; i++ {
		raw, _ := json.Marshal(map[string]any{
			"source":      "rightmove",
			"property_id": "p" + string(rune('0'+i%10)) + string(rune('a'+i)),
			"title":       "2 Bed Flat",
			"price_val":   200000,
		})
		payload = append(payload, raw)
	}
	return payload
}

func TestStartCleanListingsImportSucceedsForSmallImport(t *testing.T) {
	rawRepo := &fakeRawListingRepo{}
	jobRepo := newFakeImportJobRepo()

	uc := NewUseCase(rawRepo, jobRepo, &fakeIngester{}, nil, 10, 4)

	payload := validCleanListingPayload(3)

	result, err := uc.StartCleanListingsImport(context.Background(), payload, ActivityContext{ActorUserID: "user-1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Total != 3 {
		t.Fatalf("expected total 3, got %d", result.Total)
	}

	jobRepo.waitDone(t)

	if jobRepo.CreateCalls() != 1 {
		t.Fatalf("expected exactly one import job to be created, got %d", jobRepo.CreateCalls())
	}

	if rawRepo.SaveCalls() != 3 {
		t.Fatalf("expected 3 raw listings saved, got %d", rawRepo.SaveCalls())
	}

	job, _ := jobRepo.GetByID(context.Background(), "job-1")
	if job.Status != importjob.StatusCompleted {
		t.Fatalf("expected job status completed, got %s", job.Status)
	}
}

func TestStartCleanListingsImportRejectsOversizedImport(t *testing.T) {
	rawRepo := &fakeRawListingRepo{}
	jobRepo := newFakeImportJobRepo()

	uc := NewUseCase(rawRepo, jobRepo, &fakeIngester{}, nil, 2, 4)

	payload := validCleanListingPayload(5)

	_, err := uc.StartCleanListingsImport(context.Background(), payload, ActivityContext{ActorUserID: "user-1"})
	if err == nil {
		t.Fatal("expected error for oversized import, got nil")
	}

	if !errors.Is(err, ErrImportTooLarge) {
		t.Fatalf("expected ErrImportTooLarge, got %v", err)
	}

	var tooLarge *ImportTooLargeError
	if !errors.As(err, &tooLarge) {
		t.Fatalf("expected *ImportTooLargeError, got %T", err)
	}

	if tooLarge.MaxRows != 2 || tooLarge.ReceivedRows != 5 {
		t.Fatalf("expected maxRows=2 receivedRows=5, got maxRows=%d receivedRows=%d", tooLarge.MaxRows, tooLarge.ReceivedRows)
	}

	if jobRepo.CreateCalls() != 0 {
		t.Fatalf("expected no import job to be created for an oversized import, got %d", jobRepo.CreateCalls())
	}

	if rawRepo.SaveCalls() != 0 {
		t.Fatalf("expected no raw listings written for an oversized import, got %d", rawRepo.SaveCalls())
	}
}

func TestListImportJobsReturnsJobsFromRepository(t *testing.T) {
	jobRepo := newFakeImportJobRepo()
	jobRepo.job = importjob.ImportJob{ID: "job-1", Status: importjob.StatusCompleted}

	uc := NewUseCase(&fakeRawListingRepo{}, jobRepo, &fakeIngester{}, nil, 10, 4)

	jobs, err := uc.ListImportJobs(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(jobs) != 1 || jobs[0].ID != "job-1" {
		t.Fatalf("expected to get back job-1, got %+v", jobs)
	}
}

func TestCancelImportJobMarksJobCancelled(t *testing.T) {
	jobRepo := newFakeImportJobRepo()
	jobRepo.job = importjob.ImportJob{ID: "job-1", Status: importjob.StatusProcessing}

	uc := NewUseCase(&fakeRawListingRepo{}, jobRepo, &fakeIngester{}, nil, 10, 4)

	job, err := uc.CancelImportJob(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if job.Status != importjob.StatusCancelled {
		t.Fatalf("expected status cancelled, got %s", job.Status)
	}
}
