package listing

import "strings"

func Normalise(l Listing) Listing {
	l.City = strings.TrimSpace(l.City)
	l.City = strings.Title(strings.ToLower(l.City))

	l.PropertyType = strings.TrimSpace(l.PropertyType)
	l.PropertyType = strings.ToLower(l.PropertyType)

	l.Postcode = strings.TrimSpace(l.Postcode)
	l.Postcode = strings.ToUpper(l.Postcode)

	l.PostcodeArea = ExtractPostcodeArea(l.Postcode)

	l.SourcePlatform = strings.TrimSpace(l.SourcePlatform)
	l.ExternalPropertyID = strings.TrimSpace(l.ExternalPropertyID)

	return l
}

func ExtractPostcodeArea(postcode string) string {
	postcode = strings.TrimSpace(strings.ToUpper(postcode))

	if postcode == "" {
		return ""
	}

	parts := strings.Fields(postcode)
	if len(parts) == 0 {
		return ""
	}

	return parts[0]
}