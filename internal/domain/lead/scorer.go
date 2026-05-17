package lead

import (
	"strings"
	"time"

	"github.com/nath070707/estatelink-lead-engine/internal/domain/listing"
)

func CalculateScore(l listing.Listing) Score {
	reasons := make([]Reason, 0)
	total := 0

	yieldPoints, yieldReason := scoreYield(l)
	total += yieldPoints
	if yieldReason != nil {
		reasons = append(reasons, *yieldReason)
	}

	marketPoints, marketReason := scoreBelowMarket(l)
	total += marketPoints
	if marketReason != nil {
		reasons = append(reasons, *marketReason)
	}

	motivationPoints, motivationReason := scoreSellerMotivation(l)
	total += motivationPoints
	if motivationReason != nil {
		reasons = append(reasons, *motivationReason)
	}

	stalePoints, staleReason := scoreStaleness(l)
	total += stalePoints
	if staleReason != nil {
		reasons = append(reasons, *staleReason)
	}

	dataPoints, dataReason := scoreDataQuality(l)
	total += dataPoints
	if dataReason != nil {
		reasons = append(reasons, *dataReason)
	}

	if total > 100 {
		total = 100
	}

	return Score{
		ListingID: l.ID,
		Value:     total,
		Grade:     gradeFromScore(total),
		Reasons:   reasons,
		CreatedAt: time.Now(),
	}
}

func scoreYield(l listing.Listing) (int, *Reason) {
	if l.Price <= 0 || l.RentalEstimate <= 0 {
		return 0, nil
	}

	annualRent := l.RentalEstimate * 12
	yieldPercentage := (float64(annualRent) / float64(l.Price)) * 100

	if yieldPercentage >= 8 {
		return 25, &Reason{
			Code:    "VERY_HIGH_YIELD",
			Message: "Estimated rental yield is 8% or higher",
			Points:  25,
		}
	}

	if yieldPercentage >= 6 {
		return 20, &Reason{
			Code:    "HIGH_YIELD",
			Message: "Estimated rental yield is 6% or higher",
			Points:  20,
		}
	}

	if yieldPercentage >= 4 {
		return 10, &Reason{
			Code:    "MODERATE_YIELD",
			Message: "Estimated rental yield is 4% or higher",
			Points:  10,
		}
	}

	return 0, nil
}

func scoreBelowMarket(l listing.Listing) (int, *Reason) {
	if l.Price <= 0 || l.MarketPriceEstimate <= 0 {
		return 0, nil
	}

	difference := float64(l.MarketPriceEstimate-l.Price) / float64(l.MarketPriceEstimate) * 100

	if difference >= 15 {
		return 25, &Reason{
			Code:    "SIGNIFICANTLY_BELOW_MARKET",
			Message: "Listing appears 15% or more below market estimate",
			Points:  25,
		}
	}

	if difference >= 8 {
		return 18, &Reason{
			Code:    "BELOW_MARKET",
			Message: "Listing appears 8% or more below market estimate",
			Points:  18,
		}
	}

	if difference >= 3 {
		return 8, &Reason{
			Code:    "SLIGHTLY_BELOW_MARKET",
			Message: "Listing appears slightly below market estimate",
			Points:  8,
		}
	}

	return 0, nil
}

func scoreSellerMotivation(l listing.Listing) (int, *Reason) {
	text := strings.ToLower(l.Title + " " + l.Description)

	keywords := []string{
		"motivated seller",
		"quick sale",
		"cash buyer",
		"auction",
		"reduced",
		"chain free",
		"no onward chain",
		"priced to sell",
	}

	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return 20, &Reason{
				Code:    "SELLER_MOTIVATION_SIGNAL",
				Message: "Listing contains seller motivation language",
				Points:  20,
			}
		}
	}

	return 0, nil
}

func scoreStaleness(l listing.Listing) (int, *Reason) {
	if l.DaysOnMarket >= 90 {
		return 15, &Reason{
			Code:    "VERY_STALE_LISTING",
			Message: "Listing has been on the market for 90 days or more",
			Points:  15,
		}
	}

	if l.DaysOnMarket >= 45 {
		return 10, &Reason{
			Code:    "STALE_LISTING",
			Message: "Listing has been on the market for 45 days or more",
			Points:  10,
		}
	}

	return 0, nil
}

func scoreDataQuality(l listing.Listing) (int, *Reason) {
	score := 0

	if l.Title != "" {
		score += 3
	}
	if l.Description != "" {
		score += 3
	}
	if l.Price > 0 {
		score += 3
	}
	if l.PostcodeArea != "" {
		score += 3
	}
	if l.Bedrooms > 0 {
		score += 3
	}

	if score == 15 {
		return 15, &Reason{
			Code:    "HIGH_DATA_QUALITY",
			Message: "Listing has strong data completeness",
			Points:  15,
		}
	}

	if score >= 9 {
		return score, &Reason{
			Code:    "PARTIAL_DATA_QUALITY",
			Message: "Listing has acceptable data completeness",
			Points:  score,
		}
	}

	return score, nil
}

func gradeFromScore(score int) Grade {
	switch {
	case score >= 80:
		return GradeA
	case score >= 60:
		return GradeB
	case score >= 40:
		return GradeC
	default:
		return GradeD
	}
}






