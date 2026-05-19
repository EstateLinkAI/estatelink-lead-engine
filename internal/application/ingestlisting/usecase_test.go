package ingestlisting

import (
	"context"
	"testing"
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/lead"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/listing"
)

type fakeListingRepo struct{}

func (r *fakeListingRepo) Create(ctx context.Context, l listing.Listing) (listing.Listing, error) {
	l.ID = 1
	l.CreatedAt = time.Now()
	l.UpdatedAt = time.Now()
	return l, nil
}

type fakeLeadScoreRepo struct{}

func (r *fakeLeadScoreRepo) Create(ctx context.Context, score lead.Score) (lead.Score, error) {
	score.CreatedAt = time.Now()
	return score, nil
}

func TestExecuteNormalisesListingCalculatesScoreAndPersists(t *testing.T) {
	uc := NewUseCase(&fakeListingRepo{}, &fakeLeadScoreRepo{})

	input := listing.Listing{
		Title:               "2 Bed Flat in Manchester",
		Description:         "Vendor motivated, quick sale preferred",
		Price:               240000,
		City:                " manchester ",
		Postcode:            "m1 4ab",
		PropertyType:        " Flat ",
		Bedrooms:            2,
		Bathrooms:           1,
		RentalEstimate:      1600,
		MarketPriceEstimate: 280000,
		DaysOnMarket:        65,
		SourcePlatform:      "Rightmove",
	}

	result, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Listing.ID == 0 {
		t.Fatal("expected saved listing to have an ID")
	}

	if result.Listing.City != "Manchester" {
		t.Fatalf("expected city Manchester, got %s", result.Listing.City)
	}

	if result.Listing.PostcodeArea != "M1" {
		t.Fatalf("expected postcode area M1, got %s", result.Listing.PostcodeArea)
	}

	if result.Score.ListingID != result.Listing.ID {
		t.Fatalf("expected score listing ID %d, got %d", result.Listing.ID, result.Score.ListingID)
	}

	if result.Score.Value < 80 {
		t.Fatalf("expected strong lead score, got %d", result.Score.Value)
	}
}
