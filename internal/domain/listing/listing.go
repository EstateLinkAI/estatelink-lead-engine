package listing

import "time"

type Listing struct {
	ID                  int64     `json:"id"`
	Title               string    `json:"title"`
	Description         string    `json:"description"`
	Price               int       `json:"price"`
	City                string    `json:"city"`
	Postcode            string    `json:"postcode"`
	PostcodeArea        string    `json:"postcodeArea"`
	PropertyType        string    `json:"propertyType"`
	Bedrooms            int       `json:"bedrooms"`
	Bathrooms           int       `json:"bathrooms"`
	RentalEstimate      int       `json:"rentalEstimate"`
	MarketPriceEstimate int       `json:"marketPriceEstimate"`
	DaysOnMarket        int       `json:"daysOnMarket"`
	SourcePlatform      string    `json:"sourcePlatform"`
	SourceURL           string    `json:"sourceUrl"`
	ExternalPropertyID  string    `json:"externalPropertyId"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}
