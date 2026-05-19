package lead

type ReadModel struct {
	ID             int64
	ListingID      int64
	Title          string
	City           string
	Postcode       string
	PostcodeArea   string
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
	PostcodeArea   string
	PropertyType   string
	SourcePlatform string
	MinScore       *int
	Limit          int
	Offset         int
}
