package lead

import "time"

type Grade string

const (
	GradeA Grade = "A"
	GradeB Grade = "B"
	GradeC Grade = "C"
	GradeD Grade = "D"
)

type Reason struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Points  int    `json:"points"`
}

type Score struct {
	ListingID int64     `json:"listingId"`
	Value     int       `json:"value"`
	Grade     Grade     `json:"grade"`
	Reasons   []Reason  `json:"reasons"`
	CreatedAt time.Time `json:"createdAt"`
}
