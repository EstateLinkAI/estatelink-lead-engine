package lead

type ReadModel struct {
	ID             string
	ListingID      string
	Title          string
	Address        string
	City           string
	Postcode       string
	PropertyType   string
	Price          int
	Bedrooms       int
	SourcePlatform string
	Score          int
	Grade          string
	Reasons        []string
}

type ListFilters struct {
	City           string
	Postcode      string
	PropertyType  string
	SourcePlatform string
	MinScore       *int
	Limit          int
	Offset         int
}