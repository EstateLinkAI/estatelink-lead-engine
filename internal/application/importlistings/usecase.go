package importlistings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/ingestlisting"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/activitylog"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/importjob"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/listing"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/rawlisting"
)

type RawListingRepository interface {
	Save(ctx context.Context, listing rawlisting.RawListing) (string, error)
	MarkProcessed(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, reason string) error
}

type ImportJobRepository interface {
	Create(ctx context.Context, totalCount int) (importjob.ImportJob, error)
	GetByID(ctx context.Context, id string) (importjob.ImportJob, error)
	List(ctx context.Context, limit int) ([]importjob.ImportJob, error)
	MarkProcessing(ctx context.Context, id string) error
	IncrementProcessed(ctx context.Context, id string) error
	IncrementFailed(ctx context.Context, id string) error
	MarkCompleted(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, reason string) error
	MarkCancelled(ctx context.Context, id string) error
}

type ListingIngester interface {
	Execute(ctx context.Context, input listing.Listing) (ingestlisting.Result, error)
}

type UseCase struct {
	rawListings    RawListingRepository
	importJobs     ImportJobRepository
	ingester       ListingIngester
	activityLogger ActivityLogger
	maxImportRows  int
	importWorkers  int

	cancelFnsMu sync.Mutex
	cancelFns   map[string]context.CancelFunc
}

// ErrImportTooLarge is the sentinel wrapped by ImportTooLargeError, for
// callers that only need errors.Is. It is checked before the import job is
// created, so an oversized import never creates an import_jobs row or
// writes any raw listings.
var ErrImportTooLarge = errors.New("import payload exceeds maximum allowed rows")

// ImportTooLargeError carries the row counts needed to build a clear 400
// response (see ImportHandler.ImportCleanListings), while still satisfying
// errors.Is(err, ErrImportTooLarge) via Unwrap.
type ImportTooLargeError struct {
	MaxRows      int
	ReceivedRows int
}

func (e *ImportTooLargeError) Error() string {
	return fmt.Sprintf("%s: got %d rows, limit is %d", ErrImportTooLarge, e.ReceivedRows, e.MaxRows)
}

func (e *ImportTooLargeError) Unwrap() error {
	return ErrImportTooLarge
}

// defaultImportWorkers and defaultMaxImportRows are used when NewUseCase is
// given a non-positive value, so callers (and existing tests) that don't
// care about these limits still get sane behaviour.
const (
	defaultImportWorkers = 4
	defaultMaxImportRows = 5000
)

type ActivityLogger interface {
	Log(ctx context.Context, log activitylog.ActivityLog) error
}

type ActivityContext struct {
	ActorUserID string
	IPAddress   string
	UserAgent   string
	Filename    string
}

func NewUseCase(
	rawListings RawListingRepository,
	importJobs ImportJobRepository,
	ingester ListingIngester,
	activityLogger ActivityLogger,
	maxImportRows int,
	importWorkers int,
) *UseCase {
	if maxImportRows <= 0 {
		maxImportRows = defaultMaxImportRows
	}

	if importWorkers <= 0 {
		importWorkers = defaultImportWorkers
	}

	return &UseCase{
		rawListings:    rawListings,
		importJobs:     importJobs,
		ingester:       ingester,
		activityLogger: activityLogger,
		maxImportRows:  maxImportRows,
		importWorkers:  importWorkers,
		cancelFns:      make(map[string]context.CancelFunc),
	}
}

type StartImportResult struct {
	JobID  string `json:"jobId"`
	Status string `json:"status"`
	Total  int    `json:"total"`
}

type cleanListingInput struct {
	Source              string `json:"source"`
	PropertyID          string `json:"property_id"`
	ScrapedAt           string `json:"scraped_at"`
	URL                 string `json:"url"`
	Title               string `json:"title"`
	Description         string `json:"description"`
	PriceVal            int    `json:"price_val"`
	DisplayAddress      string `json:"display_address"`
	Postcode            string `json:"postcode"`
	PropertyType        string `json:"property_type"`
	Bedrooms            int    `json:"bedrooms"`
	Bathrooms           int    `json:"bathrooms"`
	RentalEstimate      int    `json:"rental_estimate"`
	MarketPriceEstimate int    `json:"market_price_estimate"`
	DateAdded           string `json:"date_added"`
}

func (u *UseCase) StartCleanListingsImport(ctx context.Context, payload []json.RawMessage, activityCtx ActivityContext) (StartImportResult, error) {
	if len(payload) == 0 {
		return StartImportResult{}, errors.New("import payload cannot be empty")
	}

	if len(payload) > u.maxImportRows {
		return StartImportResult{}, &ImportTooLargeError{MaxRows: u.maxImportRows, ReceivedRows: len(payload)}
	}

	job, err := u.importJobs.Create(ctx, len(payload))
	if err != nil {
		return StartImportResult{}, err
	}

	payloadCopy := make([]json.RawMessage, len(payload))
	copy(payloadCopy, payload)

	u.logActivityBestEffort(ctx, activitylog.ActivityLog{
		ActorUserID: activityCtx.ActorUserID,
		Action:      "import.started",
		EntityType:  "import_job",
		Metadata: buildImportMetadata(
			activityCtx.Filename,
			map[string]interface{}{
				"import_type": "clean_listings",
				"job_id":      job.ID,
			},
		),
		IPAddress: activityCtx.IPAddress,
		UserAgent: activityCtx.UserAgent,
	})

	jobCtx, cancel := context.WithCancel(context.Background())
	u.setCancelFunc(job.ID, cancel)

	go u.processImportJob(jobCtx, job.ID, payloadCopy, activityCtx)

	return StartImportResult{
		JobID:  job.ID,
		Status: job.Status,
		Total:  job.TotalCount,
	}, nil
}

func (u *UseCase) GetImportJob(ctx context.Context, id string) (importjob.ImportJob, error) {
	return u.importJobs.GetByID(ctx, id)
}

func (u *UseCase) ListImportJobs(ctx context.Context, limit int) ([]importjob.ImportJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	return u.importJobs.List(ctx, limit)
}

// CancelImportJob signals the running goroutine for jobID (if this server
// process is the one running it) to stop processing further listings, and
// marks the job cancelled unless it has already reached a terminal state.
func (u *UseCase) CancelImportJob(ctx context.Context, id string) (importjob.ImportJob, error) {
	if cancel := u.takeCancelFunc(id); cancel != nil {
		cancel()
	}

	if err := u.importJobs.MarkCancelled(ctx, id); err != nil {
		return importjob.ImportJob{}, err
	}

	return u.importJobs.GetByID(ctx, id)
}

func (u *UseCase) setCancelFunc(jobID string, cancel context.CancelFunc) {
	u.cancelFnsMu.Lock()
	defer u.cancelFnsMu.Unlock()
	u.cancelFns[jobID] = cancel
}

func (u *UseCase) takeCancelFunc(jobID string) context.CancelFunc {
	u.cancelFnsMu.Lock()
	defer u.cancelFnsMu.Unlock()
	cancel := u.cancelFns[jobID]
	return cancel
}

func (u *UseCase) clearCancelFunc(jobID string) {
	u.cancelFnsMu.Lock()
	defer u.cancelFnsMu.Unlock()
	delete(u.cancelFns, jobID)
}

// processPayloadConcurrently runs processOne for each raw listing using a
// bounded worker pool (sized by u.importWorkers, see IMPORT_WORKERS), stopping
// new dispatches once ctx is cancelled. Work already in flight is allowed to
// finish so counts stay accurate. Each listing does a handful of sequential
// round trips (save raw listing, upsert listing, upsert score, upsert
// strategy scores), so the worker count is tuned to keep the pool busy
// without exhausting it alongside normal API traffic - see pgxpool.MaxConns
// in cmd/api/main.go.
func (u *UseCase) processPayloadConcurrently(ctx context.Context, jobID string, payload []json.RawMessage) {
	sem := make(chan struct{}, u.importWorkers)
	var wg sync.WaitGroup

	for _, raw := range payload {
		if ctx.Err() != nil {
			break
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(raw json.RawMessage) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := u.processOne(ctx, jobID, raw); err != nil {
				_ = u.importJobs.IncrementFailed(ctx, jobID)
				return
			}

			_ = u.importJobs.IncrementProcessed(ctx, jobID)
		}(raw)
	}

	wg.Wait()
}

func (u *UseCase) processImportJob(ctx context.Context, jobID string, payload []json.RawMessage, activityCtx ActivityContext) {
	defer u.clearCancelFunc(jobID)

	if err := u.importJobs.MarkProcessing(ctx, jobID); err != nil {
		_ = u.importJobs.MarkFailed(ctx, jobID, err.Error())
		u.logImportFailure(ctx, jobID, activityCtx, err.Error())
		return
	}

	u.processPayloadConcurrently(ctx, jobID, payload)

	if ctx.Err() != nil {
		u.logActivityBestEffort(context.Background(), activitylog.ActivityLog{
			ActorUserID: activityCtx.ActorUserID,
			Action:      "import.cancelled",
			EntityType:  "import_job",
			Metadata: buildImportMetadata(
				activityCtx.Filename,
				map[string]interface{}{
					"import_type": "clean_listings",
					"job_id":      jobID,
				},
			),
			IPAddress: activityCtx.IPAddress,
			UserAgent: activityCtx.UserAgent,
		})
		return
	}

	job, err := u.importJobs.GetByID(ctx, jobID)
	if err != nil {
		_ = u.importJobs.MarkFailed(ctx, jobID, err.Error())
		u.logImportFailure(ctx, jobID, activityCtx, err.Error())
		return
	}

	if job.FailedCount > 0 && job.ProcessedCount == 0 {
		reason := "all listings failed to import"
		_ = u.importJobs.MarkFailed(ctx, jobID, reason)
		u.logImportFailure(ctx, jobID, activityCtx, reason)
		return
	}

	if err := u.importJobs.MarkCompleted(ctx, jobID); err != nil {
		u.logImportFailure(ctx, jobID, activityCtx, err.Error())
		return
	}

	completedJob, err := u.importJobs.GetByID(ctx, jobID)
	if err != nil {
		u.logImportFailure(ctx, jobID, activityCtx, err.Error())
		return
	}

	u.logActivityBestEffort(ctx, activitylog.ActivityLog{
		ActorUserID: activityCtx.ActorUserID,
		Action:      "import.completed",
		EntityType:  "import_job",
		Metadata: buildImportMetadata(
			activityCtx.Filename,
			map[string]interface{}{
				"import_type":    "clean_listings",
				"job_id":         jobID,
				"total_rows":     completedJob.TotalCount,
				"processed_rows": completedJob.ProcessedCount,
				"failed_rows":    completedJob.FailedCount,
			},
		),
		IPAddress: activityCtx.IPAddress,
		UserAgent: activityCtx.UserAgent,
	})
}

func (u *UseCase) processOne(ctx context.Context, jobID string, raw json.RawMessage) error {
	var input cleanListingInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return fmt.Errorf("invalid listing JSON: %w", err)
	}

	if input.Source == "" {
		return errors.New("source is required")
	}

	if input.PropertyID == "" {
		return errors.New("property_id is required")
	}

	var scrapedAt *time.Time
	if input.ScrapedAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, input.ScrapedAt)
		if err != nil {
			return fmt.Errorf("invalid scraped_at value: %w", err)
		}
		scrapedAt = &parsed
	}

	rawListing := rawlisting.RawListing{
		ImportJobID:        &jobID,
		Source:             input.Source,
		ExternalPropertyID: input.PropertyID,
		RawPayload:         raw,
		ScrapedAt:          scrapedAt,
		ProcessingStatus:   rawlisting.StatusPending,
	}

	rawListingID, err := u.rawListings.Save(ctx, rawListing)
	if err != nil {
		return fmt.Errorf("save raw listing: %w", err)
	}

	mappedListing := mapCleanListingToDomain(input)

	if _, err := u.ingester.Execute(ctx, mappedListing); err != nil {
		_ = u.rawListings.MarkFailed(ctx, rawListingID, err.Error())
		return fmt.Errorf("ingest listing from raw listing %s: %w", rawListingID, err)
	}

	if err := u.rawListings.MarkProcessed(ctx, rawListingID); err != nil {
		return fmt.Errorf("mark raw listing processed: %w", err)
	}

	return nil
}

func mapCleanListingToDomain(input cleanListingInput) listing.Listing {
	return listing.Listing{
		Title:               input.Title,
		Description:         input.Description,
		Price:               input.PriceVal,
		City:                inferCity(input.DisplayAddress, input.Postcode),
		Postcode:            input.Postcode,
		PostcodeArea:        inferPostcodeArea(input.Postcode),
		PropertyType:        input.PropertyType,
		Bedrooms:            input.Bedrooms,
		Bathrooms:           input.Bathrooms,
		RentalEstimate:      input.RentalEstimate,
		MarketPriceEstimate: input.MarketPriceEstimate,
		DaysOnMarket:        calculateDaysOnMarket(input.DateAdded),
		SourcePlatform:      input.Source,
		SourceURL:           input.URL,
		ExternalPropertyID:  input.PropertyID,
	}
}

func inferPostcodeArea(postcode string) string {
	postcode = strings.TrimSpace(strings.ToUpper(postcode))
	if postcode == "" {
		return ""
	}

	parts := strings.Fields(postcode)
	if len(parts) == 0 {
		return ""
	}

	outward := parts[0]

	var area strings.Builder
	for _, r := range outward {
		if r >= 'A' && r <= 'Z' {
			area.WriteRune(r)
			continue
		}
		break
	}

	return area.String()
}

func inferCity(displayAddress string, postcode string) string {
	address := strings.ToLower(displayAddress)
	postcode = strings.ToUpper(strings.TrimSpace(postcode))

	if strings.Contains(address, "aberdeen") || strings.HasPrefix(postcode, "AB") {
		return "Aberdeen"
	}

	return ""
}

func calculateDaysOnMarket(dateAdded string) int {
	if dateAdded == "" {
		return 0
	}

	parsed, err := time.Parse("20060102", dateAdded)
	if err != nil {
		return 0
	}

	days := int(time.Since(parsed).Hours() / 24)
	if days < 0 {
		return 0
	}

	return days
}

func (u *UseCase) logImportFailure(ctx context.Context, jobID string, activityCtx ActivityContext, message string) {
	u.logActivityBestEffort(ctx, activitylog.ActivityLog{
		ActorUserID: activityCtx.ActorUserID,
		Action:      "import.failed",
		EntityType:  "import_job",
		Metadata: buildImportMetadata(
			activityCtx.Filename,
			map[string]interface{}{
				"import_type": "clean_listings",
				"job_id":      jobID,
				"error":       message,
			},
		),
		IPAddress: activityCtx.IPAddress,
		UserAgent: activityCtx.UserAgent,
	})
}

func (u *UseCase) logActivityBestEffort(ctx context.Context, entry activitylog.ActivityLog) {
	if u.activityLogger == nil {
		return
	}

	if err := u.activityLogger.Log(ctx, entry); err != nil {
		log.Printf("activity log failed for action %q: %v", entry.Action, err)
	}
}

func buildImportMetadata(filename string, fields map[string]interface{}) map[string]interface{} {
	if filename != "" {
		fields["filename"] = filename
	}

	return fields
}
