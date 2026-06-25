package propertyimage

import "time"

type PropertyImage struct {
	ID        int64     `json:"id"`
	ListingID int64     `json:"listingId"`
	SourceURL string    `json:"sourceUrl"`
	Position  int       `json:"position"`
	IsPrimary bool      `json:"isPrimary"`
	CreatedAt time.Time `json:"createdAt"`
}