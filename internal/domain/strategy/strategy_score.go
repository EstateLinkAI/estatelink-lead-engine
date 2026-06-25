package strategy

import "time"

type Strategy string

const (
	StrategyBuyToLet    Strategy = "buy_to_let"
	StrategyBRRRR       Strategy = "brrrr"
	StrategyFlip        Strategy = "flip"
	StrategyBuyAndHold  Strategy = "buy_and_hold"
	StrategyHMO         Strategy = "hmo"
	StrategyDevelopment Strategy = "development"
)

type StrategyScore struct {
	ID        int64     `json:"id"`
	ListingID int64     `json:"listingId"`
	Strategy  Strategy  `json:"strategy"`
	Score     int       `json:"score"`
	Grade     string    `json:"grade"`
	Reasons   []string  `json:"reasons"`
	CreatedAt time.Time `json:"createdAt"`
}

func AllStrategies() []Strategy {
	return []Strategy{
		StrategyBuyToLet,
		StrategyBRRRR,
		StrategyFlip,
		StrategyBuyAndHold,
		StrategyHMO,
		StrategyDevelopment,
	}
}