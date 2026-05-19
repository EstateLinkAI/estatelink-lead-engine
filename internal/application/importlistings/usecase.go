package importlistings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/ingestlisting"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/listing"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/rawlisting"
)

type RawListingRepository interface {
	Save(ctx context.Context, listing rawlisting.RawListing) (string, error)
	MarkProcessed(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, reason string) error
}

type ListingIngester interface {
	Execute(ctx context.Context, input listing.Listing) (ingestlisting.Result, error)
}

type UseCase struct {
	rawListings RawListingRepository
	ingester    ListingIngester
}

func NewUseCase(rawListings RawListingRepository, ingester ListingIngester) *UseCase {
	return &UseCase{
		rawListings: rawListings,
		ingester:    ingester,
	}
}

type ImportResult struct {
	Imported int              `json:"imported"`
	Failed   int              `json:"failed"`
	Items    []ImportItemInfo `json:"items"`
}

type ImportItemInfo struct {
	RawListingID       string `json:"rawListingId,omitempty"`
	ListingID          int64  `json:"listingId,omitempty"`
	// LeadScoreID        int64  `json:"leadScoreId,omitempty"`
	Source             string `json:"source,omitempty"`
	ExternalPropertyID string `json:"externalPropertyId,omitempty"`
	Status             string `json:"status"`
	Error              string `json:"error,omitempty"`
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

func (u *UseCase) ImportCleanListings(ctx context.Context, payload []json.RawMessage) (ImportResult, error) {
	if len(payload) == 0 {
		return ImportResult{}, errors.New("import payload cannot be empty")
	}

	result := ImportResult{
		Items: make([]ImportItemInfo, 0, len(payload)),
	}

	for _, raw := range payload {
		item, err := u.importOne(ctx, raw)
		if err != nil {
			result.Failed++
			result.Items = append(result.Items, ImportItemInfo{
				Status: rawlisting.StatusFailed,
				Error:  err.Error(),
			})
			continue
		}

		result.Imported++
		result.Items = append(result.Items, item)
	}

	return result, nil
}

func (u *UseCase) importOne(ctx context.Context, raw json.RawMessage) (ImportItemInfo, error) {
	var input cleanListingInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return ImportItemInfo{}, fmt.Errorf("invalid listing JSON: %w", err)
	}

	if input.Source == "" {
		return ImportItemInfo{}, errors.New("source is required")
	}

	if input.PropertyID == "" {
		return ImportItemInfo{}, errors.New("property_id is required")
	}

	var scrapedAt *time.Time
	if input.ScrapedAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, input.ScrapedAt)
		if err != nil {
			return ImportItemInfo{}, fmt.Errorf("invalid scraped_at value: %w", err)
		}
		scrapedAt = &parsed
	}

	rawListing := rawlisting.RawListing{
		Source:             input.Source,
		ExternalPropertyID: input.PropertyID,
		RawPayload:         raw,
		ScrapedAt:          scrapedAt,
		ProcessingStatus:   rawlisting.StatusPending,
	}

	rawListingID, err := u.rawListings.Save(ctx, rawListing)
	if err != nil {
		return ImportItemInfo{}, fmt.Errorf("save raw listing: %w", err)
	}

	mappedListing := mapCleanListingToDomain(input)

	ingestResult, err := u.ingester.Execute(ctx, mappedListing)
	if err != nil {
		_ = u.rawListings.MarkFailed(ctx, rawListingID, err.Error())

		return ImportItemInfo{}, fmt.Errorf("ingest listing from raw listing %s: %w", rawListingID, err)
	}

	if err := u.rawListings.MarkProcessed(ctx, rawListingID); err != nil {
		return ImportItemInfo{}, fmt.Errorf("mark raw listing processed: %w", err)
	}

	return ImportItemInfo{
		RawListingID:       rawListingID,
		ListingID:          ingestResult.Listing.ID,
		Source:             input.Source,
		ExternalPropertyID: input.PropertyID,
		Status:             rawlisting.StatusProcessed,
	}, nil
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