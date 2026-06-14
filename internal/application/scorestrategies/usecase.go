package scorestrategies

import (
	"context"
	"strings"
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/strategy"
)

type ListingInput struct {
	ListingID     int64
	Price        float64
	MonthlyRent  float64
	Bedrooms     int
	PropertyType string
	City         string
	PostcodeArea string
	DaysOnMarket int
}

type StrategyScoreRepository interface {
	SaveMany(ctx context.Context, scores []strategy.StrategyScore) error
}

type UseCase struct {
	repo StrategyScoreRepository
}

func NewUseCase(repo StrategyScoreRepository) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) Execute(ctx context.Context, input ListingInput) ([]strategy.StrategyScore, error) {
	scores := CalculateStrategyScores(input)

	if uc.repo == nil {
		return scores, nil
	}

	if err := uc.repo.SaveMany(ctx, scores); err != nil {
		return nil, err
	}

	return scores, nil
}

func CalculateStrategyScores(input ListingInput) []strategy.StrategyScore {
	now := time.Now().UTC()

	return []strategy.StrategyScore{
		buildScore(input.ListingID, strategy.StrategyBuyToLet, scoreBuyToLet(input), now),
		buildScore(input.ListingID, strategy.StrategyBRRRR, scoreBRRRR(input), now),
		buildScore(input.ListingID, strategy.StrategyFlip, scoreFlip(input), now),
		buildScore(input.ListingID, strategy.StrategyBuyAndHold, scoreBuyAndHold(input), now),
		buildScore(input.ListingID, strategy.StrategyHMO, scoreHMO(input), now),
		buildScore(input.ListingID, strategy.StrategyDevelopment, scoreDevelopment(input), now),
	}
}

type scoreResult struct {
	score   int
	reasons []string
}

func buildScore(listingID int64, strategyName strategy.Strategy, result scoreResult, createdAt time.Time) strategy.StrategyScore {
	score := clampScore(result.score)

	return strategy.StrategyScore{
		ListingID: listingID,
		Strategy:  strategyName,
		Score:     score,
		Grade:     strategy.GradeFromScore(score),
		Reasons:   result.reasons,
		CreatedAt: createdAt,
	}
}

func scoreBuyToLet(input ListingInput) scoreResult {
	score := 45
	reasons := []string{}

	yield := grossYield(input.Price, input.MonthlyRent)

	switch {
	case yield >= 8:
		score += 35
		reasons = append(reasons, "Gross yield is very strong")
	case yield >= 6:
		score += 25
		reasons = append(reasons, "Gross yield is above average")
	case yield >= 4.5:
		score += 10
		reasons = append(reasons, "Gross yield is acceptable")
	case yield > 0:
		score -= 15
		reasons = append(reasons, "Gross yield appears weak")
	default:
		reasons = append(reasons, "Rental income data is missing")
	}

	if input.Bedrooms >= 2 && input.Bedrooms <= 4 {
		score += 10
		reasons = append(reasons, "Bedroom count suits mainstream rental demand")
	}

	if isFlatOrHouse(input.PropertyType) {
		score += 5
		reasons = append(reasons, "Property type is suitable for standard rental demand")
	}

	return scoreResult{score: score, reasons: reasons}
}

func scoreBRRRR(input ListingInput) scoreResult {
	score := 40
	reasons := []string{}

	yield := grossYield(input.Price, input.MonthlyRent)

	if yield >= 7 {
		score += 20
		reasons = append(reasons, "Yield may support refinancing potential")
	}

	if input.DaysOnMarket >= 60 {
		score += 15
		reasons = append(reasons, "Long days on market may indicate negotiation room")
	}

	if input.Bedrooms >= 3 {
		score += 10
		reasons = append(reasons, "Bedroom count may support value-add rental strategy")
	}

	if input.Price > 0 && input.MonthlyRent == 0 {
		score -= 10
		reasons = append(reasons, "Rental income estimate is missing")
	}

	return scoreResult{score: score, reasons: reasons}
}

func scoreFlip(input ListingInput) scoreResult {
	score := 40
	reasons := []string{}

	if input.DaysOnMarket >= 60 {
		score += 20
		reasons = append(reasons, "Long days on market may create negotiation opportunity")
	}

	if input.DaysOnMarket >= 120 {
		score += 10
		reasons = append(reasons, "Very stale listing may indicate seller motivation")
	}

	if strings.Contains(normalize(input.PropertyType), "house") {
		score += 10
		reasons = append(reasons, "House type may allow stronger resale repositioning")
	}

	if input.Bedrooms >= 3 {
		score += 10
		reasons = append(reasons, "Larger layout may support resale demand")
	}

	return scoreResult{score: score, reasons: reasons}
}

func scoreBuyAndHold(input ListingInput) scoreResult {
	score := 50
	reasons := []string{}

	if input.Price > 0 {
		score += 5
		reasons = append(reasons, "Property has usable price data")
	}

	if input.Bedrooms >= 2 {
		score += 10
		reasons = append(reasons, "Bedroom count supports long-term family or tenant demand")
	}

	if input.City != "" || input.PostcodeArea != "" {
		score += 10
		reasons = append(reasons, "Location fields are available for future area intelligence")
	}

	yield := grossYield(input.Price, input.MonthlyRent)
	if yield >= 5 {
		score += 10
		reasons = append(reasons, "Yield provides some income support for long-term hold")
	}

	return scoreResult{score: score, reasons: reasons}
}

func scoreHMO(input ListingInput) scoreResult {
	score := 35
	reasons := []string{}

	if input.Bedrooms >= 4 {
		score += 35
		reasons = append(reasons, "Bedroom count is suitable for potential HMO strategy")
	} else if input.Bedrooms == 3 {
		score += 15
		reasons = append(reasons, "Bedroom count may support small shared rental strategy")
	} else {
		score -= 10
		reasons = append(reasons, "Bedroom count is likely too low for HMO strategy")
	}

	if strings.Contains(normalize(input.PropertyType), "house") {
		score += 15
		reasons = append(reasons, "House type is more suitable for HMO layout")
	}

	yield := grossYield(input.Price, input.MonthlyRent)
	if yield >= 7 {
		score += 10
		reasons = append(reasons, "Yield may support shared rental economics")
	}

	return scoreResult{score: score, reasons: reasons}
}

func scoreDevelopment(input ListingInput) scoreResult {
	score := 30
	reasons := []string{}

	propertyType := normalize(input.PropertyType)

	if strings.Contains(propertyType, "detached") || strings.Contains(propertyType, "semi") {
		score += 20
		reasons = append(reasons, "House type may offer extension or redevelopment potential")
	}

	if input.Bedrooms >= 3 {
		score += 10
		reasons = append(reasons, "Larger property may offer layout reconfiguration potential")
	}

	if input.DaysOnMarket >= 90 {
		score += 10
		reasons = append(reasons, "Long market exposure may support negotiation")
	}

	reasons = append(reasons, "Development score is limited until plot size and planning data are added")

	return scoreResult{score: score, reasons: reasons}
}

func grossYield(price float64, monthlyRent float64) float64 {
	if price <= 0 || monthlyRent <= 0 {
		return 0
	}

	return (monthlyRent * 12 / price) * 100
}

func clampScore(score int) int {
	if score < 0 {
		return 0
	}

	if score > 100 {
		return 100
	}

	return score
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isFlatOrHouse(propertyType string) bool {
	normalized := normalize(propertyType)

	return strings.Contains(normalized, "flat") ||
		strings.Contains(normalized, "apartment") ||
		strings.Contains(normalized, "house") ||
		strings.Contains(normalized, "terraced") ||
		strings.Contains(normalized, "semi") ||
		strings.Contains(normalized, "detached")
}