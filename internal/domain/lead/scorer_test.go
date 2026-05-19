package lead

import (
	"testing"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/listing"
)

func TestCalculateScoreReturnsHighScoreForStrongLead(t *testing.T) {
	l := listing.Listing{
		ID:                  1,
		Title:               "2 Bed Flat in Manchester",
		Description:         "Vendor motivated, quick sale preferred. Chain free.",
		Price:               240000,
		City:                "Manchester",
		Postcode:            "M1 4AB",
		PostcodeArea:        "M1",
		PropertyType:        "flat",
		Bedrooms:            2,
		Bathrooms:           1,
		RentalEstimate:      1600,
		MarketPriceEstimate: 280000,
		DaysOnMarket:        65,
		SourcePlatform:      "Rightmove",
	}

	score := CalculateScore(l)

	if score.Value < 80 {
		t.Fatalf("expected score to be at least 80, got %d", score.Value)
	}

	if score.Grade != GradeA {
		t.Fatalf("expected grade A, got %s", score.Grade)
	}

	if len(score.Reasons) == 0 {
		t.Fatal("expected scoring reasons, got none")
	}
}

func TestCalculateScoreReturnsLowScoreForWeakLead(t *testing.T) {
	l := listing.Listing{
		ID:                  2,
		Title:               "Studio Flat",
		Description:         "",
		Price:               300000,
		City:                "London",
		Postcode:            "SW1A 1AA",
		PostcodeArea:        "SW1A",
		PropertyType:        "studio",
		Bedrooms:            1,
		Bathrooms:           1,
		RentalEstimate:      900,
		MarketPriceEstimate: 280000,
		DaysOnMarket:        10,
		SourcePlatform:      "Zoopla",
	}

	score := CalculateScore(l)

	if score.Value >= 60 {
		t.Fatalf("expected score below 60, got %d", score.Value)
	}

	if score.Grade == GradeA {
		t.Fatalf("expected non-A grade, got %s", score.Grade)
	}
}
