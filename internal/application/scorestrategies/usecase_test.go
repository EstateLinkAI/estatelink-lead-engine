package scorestrategies

import (
	"testing"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/strategy"
)

func TestCalculateStrategyScoresReturnsAllStrategies(t *testing.T) {
	input := ListingInput{
		ListingID:       1,
		Price:           250000,
		RentalEstimate:  1600,
		Bedrooms:        4,
		PropertyType:    "Terraced house",
		City:            "Manchester",
		PostcodeArea:    "M14",
		DaysOnMarket:    75,
	}

	scores := CalculateStrategyScores(input)

	if len(scores) != len(strategy.AllStrategies()) {
		t.Fatalf("expected %d scores, got %d", len(strategy.AllStrategies()), len(scores))
	}

	seen := map[strategy.Strategy]bool{}

	for _, score := range scores {
		seen[score.Strategy] = true

		if score.ListingID != input.ListingID {
			t.Fatalf("expected listing ID %d, got %d", input.ListingID, score.ListingID)
		}

		if score.Score < 0 || score.Score > 100 {
			t.Fatalf("expected score between 0 and 100, got %d", score.Score)
		}

		if score.Grade == "" {
			t.Fatal("expected grade to be set")
		}

		if len(score.Reasons) == 0 {
			t.Fatal("expected reasons to be set")
		}
	}

	for _, strategyName := range strategy.AllStrategies() {
		if !seen[strategyName] {
			t.Fatalf("expected strategy %s to be present", strategyName)
		}
	}
}

func TestGrossYield(t *testing.T) {
	got := grossYield(240000, 1200)
	want := 6.0

	if got != want {
		t.Fatalf("expected %.2f, got %.2f", want, got)
	}
}

func TestGrossYieldReturnsZeroWhenMissingData(t *testing.T) {
	tests := []struct {
		name           string
		price          int
		rentalEstimate int
	}{
		{name: "missing price", price: 0, rentalEstimate: 1200},
		{name: "missing rental estimate", price: 250000, rentalEstimate: 0},
		{name: "negative price", price: -1, rentalEstimate: 1200},
		{name: "negative rental estimate", price: 250000, rentalEstimate: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := grossYield(tt.price, tt.rentalEstimate)
			if got != 0 {
				t.Fatalf("expected 0, got %.2f", got)
			}
		})
	}
}

func TestHMOScoreRewardsLargerHouses(t *testing.T) {
	input := ListingInput{
		ListingID:       1,
		Price:           300000,
		RentalEstimate:  2200,
		Bedrooms:        5,
		PropertyType:    "Semi-detached house",
	}

	scores := CalculateStrategyScores(input)

	var hmoScore int

	for _, score := range scores {
		if score.Strategy == strategy.StrategyHMO {
			hmoScore = score.Score
		}
	}

	if hmoScore < 70 {
		t.Fatalf("expected strong HMO score, got %d", hmoScore)
	}
}

func TestBTLScoreHandlesMissingRentalEstimate(t *testing.T) {
	input := ListingInput{
		ListingID:       1,
		Price:           250000,
		RentalEstimate:  0,
		Bedrooms:        3,
		PropertyType:    "Terraced house",
		City:            "Birmingham",
		PostcodeArea:    "B12",
		DaysOnMarket:    20,
	}

	scores := CalculateStrategyScores(input)

	var btlScore strategy.StrategyScore

	for _, score := range scores {
		if score.Strategy == strategy.StrategyBuyToLet {
			btlScore = score
		}
	}

	if btlScore.Strategy == "" {
		t.Fatal("expected buy-to-let score to be present")
	}

	if len(btlScore.Reasons) == 0 {
		t.Fatal("expected reasons to explain missing rental estimate")
	}
}