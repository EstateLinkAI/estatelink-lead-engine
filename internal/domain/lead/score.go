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
	Code    string
	Message string
	Points  int
}

type Score struct {
	ListingID int64
	Value     int
	Grade     Grade
	Reasons   []Reason
	CreatedAt time.Time
}
