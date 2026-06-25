package lead

import "github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/strategy"

type ReadModel struct {
	ID             int64                    `json:"id"`
	ListingID      int64                    `json:"listingId"`
	Title          string                   `json:"title"`
	City           string                   `json:"city"`
	Postcode       string                   `json:"postcode"`
	PostcodeArea   string                   `json:"postcodeArea"`
	PropertyType   string                   `json:"propertyType"`
	Price          int                      `json:"price"`
	Bedrooms       int                      `json:"bedrooms"`
	SourcePlatform string                   `json:"sourcePlatform"`
	Score          int                      `json:"score"`
	Grade          string                   `json:"grade"`
	Reasons        []map[string]any         `json:"reasons"`
	StrategyScores []strategy.StrategyScore `json:"strategyScores"`
}

type ListFilters struct {
	City           string
	PostcodeArea   string
	PropertyType   string
	SourcePlatform string
	MinScore       *int
	Limit          int
	Offset         int
}
