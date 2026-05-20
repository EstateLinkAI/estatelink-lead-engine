package importlistings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/ingestlisting"
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
	MarkProcessing(ctx context.Context, id string) error
	IncrementProcessed(ctx context.Context, id string) error
	IncrementFailed(ctx context.Context, id string) error
	MarkCompleted(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, reason string) error
}

type ListingIngester interface {
	Execute(ctx context.Context, input listing.Listing) (ingestlisting.Result, error)
}

type UseCase struct {
	rawListings RawListingRepository
	importJobs  ImportJobRepository
	ingester    ListingIngester
}

func NewUseCase(
	rawListings RawListingRepository,
	importJobs ImportJobRepository,
	ingester ListingIngester,
) *UseCase {
	return &UseCase{
		rawListings: rawListings,
		importJobs:  importJobs,
		ingester:    ingester,
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

func (u *UseCase) StartCleanListingsImport(ctx context.Context, payload []json.RawMessage) (StartImportResult, error) {
	if len(payload) == 0 {
		return StartImportResult{}, errors.New("import payload cannot be empty")
	}

	job, err := u.importJobs.Create(ctx, len(payload))
	if err != nil {
		return StartImportResult{}, err
	}

	payloadCopy := make([]json.RawMessage, len(payload))
	copy(payloadCopy, payload)

	go u.processImportJob(context.Background(), job.ID, payloadCopy)

	return StartImportResult{
		JobID:  job.ID,
		Status: job.Status,
		Total:  job.TotalCount,
	}, nil
}

func (u *UseCase) GetImportJob(ctx context.Context, id string) (importjob.ImportJob, error) {
	return u.importJobs.GetByID(ctx, id)
}

func (u *UseCase) processImportJob(ctx context.Context, jobID string, payload []json.RawMessage) {
	if err := u.importJobs.MarkProcessing(ctx, jobID); err != nil {
		_ = u.importJobs.MarkFailed(ctx, jobID, err.Error())
		return
	}

	for _, raw := range payload {
		if err := u.processOne(ctx, jobID, raw); err != nil {
			_ = u.importJobs.IncrementFailed(ctx, jobID)
			continue
		}

		_ = u.importJobs.IncrementProcessed(ctx, jobID)
	}

	job, err := u.importJobs.GetByID(ctx, jobID)
	if err != nil {
		_ = u.importJobs.MarkFailed(ctx, jobID, err.Error())
		return
	}

	if job.FailedCount > 0 && job.ProcessedCount == 0 {
		_ = u.importJobs.MarkFailed(ctx, jobID, "all listings failed to import")
		return
	}

	_ = u.importJobs.MarkCompleted(ctx, jobID)
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