package ingestlisting

import (
	"context"
	"testing"
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/scorestrategies"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/lead"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/listing"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/strategy"
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
	strategyScorer := scorestrategies.NewUseCase(nil)

	uc := NewUseCase(
		&fakeListingRepo{},
		&fakeLeadScoreRepo{},
		strategyScorer,
	)

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

	if len(result.StrategyScores) != len(strategy.AllStrategies()) {
		t.Fatalf("expected %d strategy scores, got %d", len(strategy.AllStrategies()), len(result.StrategyScores))
	}
}

func TestExecuteCanRunWithoutStrategyScorer(t *testing.T) {
	uc := NewUseCase(
		&fakeListingRepo{},
		&fakeLeadScoreRepo{},
		nil,
	)

	input := listing.Listing{
		Title:               "3 Bed House in Birmingham",
		Description:         "Good rental demand",
		Price:               300000,
		City:                "birmingham",
		Postcode:            "b12 8aa",
		PropertyType:        "Terraced house",
		Bedrooms:            3,
		Bathrooms:           1,
		RentalEstimate:      1800,
		MarketPriceEstimate: 330000,
		DaysOnMarket:        30,
		SourcePlatform:      "Rightmove",
	}

	result, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Listing.ID == 0 {
		t.Fatal("expected saved listing to have an ID")
	}

	if len(result.StrategyScores) != 0 {
		t.Fatalf("expected no strategy scores when strategy scorer is nil, got %d", len(result.StrategyScores))
	}
}