package listing

import "time"

type Listing struct {
	ID                  int64
	Title               string
	Description         string
	Price               int
	City                string
	Postcode            string
	PostcodeArea        string
	PropertyType        string
	Bedrooms            int
	Bathrooms           int
	RentalEstimate      int
	MarketPriceEstimate int
	DaysOnMarket        int
	SourcePlatform      string
	SourceURL           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
