package importlistings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/ingestlisting"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/importjob"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/listing"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/rawlisting"
)

type fakeRawListingRepo struct {
	mu            sync.Mutex
	saveCalls     int
	nextID        int
	failedReasons []string
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
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failedReasons = append(r.failedReasons, reason)
	return nil
}

func (r *fakeRawListingRepo) SaveCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveCalls
}

func (r *fakeRawListingRepo) FailedReasons() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.failedReasons))
	copy(out, r.failedReasons)
	return out
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

type alwaysFailingIngester struct{}

func (f *alwaysFailingIngester) Execute(ctx context.Context, input listing.Listing) (ingestlisting.Result, error) {
	return ingestlisting.Result{}, errors.New("listing insert failed: price out of range")
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

// invalidPriceTypePayload mirrors a real staging incident: price_val sent as
// a non-integer JSON number (e.g. 3.62) fails to unmarshal into the int
// field, but source/property_id still parse fine since they're independent
// JSON keys.
func invalidPriceTypePayload(n int) []json.RawMessage {
	payload := make([]json.RawMessage, 0, n)
	for i := 0; i < n; i++ {
		raw := fmt.Sprintf(`{"source":"zillow","property_id":"p%d","title":"House","price_val":3.62}`, i)
		payload = append(payload, json.RawMessage(raw))
	}
	return payload
}

func TestProcessOneClassifiesInvalidFieldTypeAndStoresReason(t *testing.T) {
	rawRepo := &fakeRawListingRepo{}
	jobRepo := newFakeImportJobRepo()

	uc := NewUseCase(rawRepo, jobRepo, &fakeIngester{}, nil, 10, 4)

	raw := json.RawMessage(`{"source":"zillow","property_id":"123","price_val":3.62}`)

	err := uc.processOne(context.Background(), "job-1", raw)
	if err == nil {
		t.Fatal("expected an error for a row with a malformed price_val field")
	}

	var rowErr *RowImportError
	if !errors.As(err, &rowErr) {
		t.Fatalf("expected *RowImportError, got %T: %v", err, err)
	}

	if rowErr.Category != invalidFieldCategory("price_val") {
		t.Fatalf("expected category %q, got %q", invalidFieldCategory("price_val"), rowErr.Category)
	}

	if !strings.Contains(err.Error(), "price_val") {
		t.Fatalf("expected error message to mention price_val, got %q", err.Error())
	}

	reasons := rawRepo.FailedReasons()
	if len(reasons) != 1 {
		t.Fatalf("expected raw_listings to record exactly 1 failure reason, got %d: %v", len(reasons), reasons)
	}

	if !strings.Contains(reasons[0], "price_val") {
		t.Fatalf("expected stored raw_listings failure reason to mention price_val, got %q", reasons[0])
	}
}

func TestStartCleanListingsImportAllRowsFailedReachesFailedStatusWithUsefulSummary(t *testing.T) {
	rawRepo := &fakeRawListingRepo{}
	jobRepo := newFakeImportJobRepo()

	uc := NewUseCase(rawRepo, jobRepo, &fakeIngester{}, nil, 50, 4)

	payload := invalidPriceTypePayload(20)

	_, err := uc.StartCleanListingsImport(context.Background(), payload, ActivityContext{ActorUserID: "user-1"})
	if err != nil {
		t.Fatalf("expected no error starting the import, got %v", err)
	}

	jobRepo.waitDone(t)

	job, _ := jobRepo.GetByID(context.Background(), "job-1")

	if job.Status != importjob.StatusFailed {
		t.Fatalf("expected job status failed, got %s", job.Status)
	}

	if job.FailedCount != 20 || job.ProcessedCount != 0 {
		t.Fatalf("expected failedCount=20 processedCount=0, got failed=%d processed=%d", job.FailedCount, job.ProcessedCount)
	}

	if job.ErrorMessage == nil || *job.ErrorMessage == "" {
		t.Fatal("expected a non-empty error_message explaining the failure")
	}

	msg := *job.ErrorMessage
	if !strings.Contains(msg, "20 rows failed") {
		t.Fatalf("expected error_message to include the failure count, got %q", msg)
	}

	if !strings.Contains(msg, "price_val") {
		t.Fatalf("expected error_message to name the dominant failing field, got %q", msg)
	}

	reasons := rawRepo.FailedReasons()
	if len(reasons) != 20 {
		t.Fatalf("expected all 20 rows to have a stored raw_listings failure reason, got %d", len(reasons))
	}
}

func TestStartCleanListingsImportAllRowsFailViaIngestKeepsTerminalFailedStatus(t *testing.T) {
	rawRepo := &fakeRawListingRepo{}
	jobRepo := newFakeImportJobRepo()

	uc := NewUseCase(rawRepo, jobRepo, &alwaysFailingIngester{}, nil, 10, 4)

	payload := validCleanListingPayload(5)

	_, err := uc.StartCleanListingsImport(context.Background(), payload, ActivityContext{ActorUserID: "user-1"})
	if err != nil {
		t.Fatalf("expected no error starting the import, got %v", err)
	}

	jobRepo.waitDone(t)

	job, _ := jobRepo.GetByID(context.Background(), "job-1")

	if job.Status != importjob.StatusFailed {
		t.Fatalf("expected job to reach terminal failed status, got %s", job.Status)
	}

	if job.ErrorMessage == nil || !strings.Contains(*job.ErrorMessage, "5 rows failed") {
		t.Fatalf("expected error_message to summarize the failure count, got %v", job.ErrorMessage)
	}
}

func TestStartCleanListingsImportPartialFailureStillCompletes(t *testing.T) {
	rawRepo := &fakeRawListingRepo{}
	jobRepo := newFakeImportJobRepo()

	uc := NewUseCase(rawRepo, jobRepo, &fakeIngester{}, nil, 10, 4)

	payload := append(validCleanListingPayload(3), invalidPriceTypePayload(2)...)

	_, err := uc.StartCleanListingsImport(context.Background(), payload, ActivityContext{ActorUserID: "user-1"})
	if err != nil {
		t.Fatalf("expected no error starting the import, got %v", err)
	}

	jobRepo.waitDone(t)

	job, _ := jobRepo.GetByID(context.Background(), "job-1")

	if job.Status != importjob.StatusCompleted {
		t.Fatalf("expected job status completed when some rows succeed, got %s", job.Status)
	}

	if job.ProcessedCount != 3 || job.FailedCount != 2 {
		t.Fatalf("expected processed=3 failed=2, got processed=%d failed=%d", job.ProcessedCount, job.FailedCount)
	}
}
