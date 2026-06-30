package importlistings

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// RowFailureCategory groups row-level import failures into buckets that can
// be counted and summarized, without losing the per-row detail kept in the
// wrapped error message.
type RowFailureCategory string

const (
	CategoryMissingSource     RowFailureCategory = "missing_source"
	CategoryMissingPropertyID RowFailureCategory = "missing_property_id"
	CategoryInvalidJSON       RowFailureCategory = "invalid_json"
	CategoryInvalidScrapedAt  RowFailureCategory = "invalid_scraped_at"
	CategorySaveFailed        RowFailureCategory = "save_failed"
	CategoryIngestFailed      RowFailureCategory = "ingest_failed"
	CategoryUnknown           RowFailureCategory = "unknown_error"

	invalidFieldPrefix = "invalid_field:"
)

// RowImportError carries a classification alongside the detailed, row-specific
// message, so failures can be aggregated into a job-level summary (see
// failureTracker.Summary) while individual rows still get a precise reason.
type RowImportError struct {
	Category RowFailureCategory
	Err      error
}

func (e *RowImportError) Error() string {
	return e.Err.Error()
}

func (e *RowImportError) Unwrap() error {
	return e.Err
}

// invalidFieldCategory builds a category for a specific malformed JSON field,
// e.g. "invalid_field:price_val", so "most common error" summaries can name
// the actual field instead of a generic "malformed JSON".
func invalidFieldCategory(field string) RowFailureCategory {
	return RowFailureCategory(invalidFieldPrefix + field)
}

// describeCategory turns a category into the human-readable text used in
// job error_message summaries and frontend-facing messages.
func describeCategory(category RowFailureCategory) string {
	if field, ok := strings.CutPrefix(string(category), invalidFieldPrefix); ok {
		return fmt.Sprintf("invalid %s field", field)
	}

	switch category {
	case CategoryMissingSource:
		return "missing source field"
	case CategoryMissingPropertyID:
		return "missing property_id field"
	case CategoryInvalidScrapedAt:
		return "invalid scraped_at field"
	case CategoryInvalidJSON:
		return "malformed row JSON"
	case CategorySaveFailed:
		return "failed to save raw listing"
	case CategoryIngestFailed:
		return "listing ingestion failed"
	default:
		return "unknown error"
	}
}

func asRowImportError(err error) *RowImportError {
	var rowErr *RowImportError
	if errors.As(err, &rowErr) {
		return rowErr
	}

	return &RowImportError{Category: CategoryUnknown, Err: err}
}

const maxFailureSamples = 10

// failureTracker collects row-level failures for a single import job run:
// enough detailed samples to log for debugging, plus counts per category so
// a "most common error" summary can be built for the job's error_message.
type failureTracker struct {
	mu      sync.Mutex
	counts  map[RowFailureCategory]int
	samples []string
}

func newFailureTracker() *failureTracker {
	return &failureTracker{counts: make(map[RowFailureCategory]int)}
}

func (t *failureTracker) Record(err error) {
	rowErr := asRowImportError(err)

	t.mu.Lock()
	defer t.mu.Unlock()

	t.counts[rowErr.Category]++

	if len(t.samples) < maxFailureSamples {
		t.samples = append(t.samples, rowErr.Error())
	}
}

// Samples returns up to maxFailureSamples detailed failure messages, for
// server-side logging.
func (t *failureTracker) Samples() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := make([]string, len(t.samples))
	copy(out, t.samples)
	return out
}

func (t *failureTracker) topCategory() (RowFailureCategory, int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var top RowFailureCategory
	var topCount int

	for category, count := range t.counts {
		if count > topCount {
			top, topCount = category, count
		}
	}

	return top, topCount
}

// Summary builds a frontend-displayable explanation of why an import job
// failed, e.g. "4826 rows failed. Most common error: invalid price_val field."
func (t *failureTracker) Summary(failedCount int) string {
	top, topCount := t.topCategory()
	if topCount == 0 {
		return fmt.Sprintf("%d rows failed.", failedCount)
	}

	return fmt.Sprintf("%d rows failed. Most common error: %s (%d/%d rows).", failedCount, describeCategory(top), topCount, failedCount)
}
