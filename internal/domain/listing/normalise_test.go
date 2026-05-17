package listing

import "testing"

func TestNormaliseCleansListingFields(t *testing.T) {
	l := Listing{
		City:         " manchester ",
		PropertyType: " Flat ",
		Postcode:     "m1 4ab",
	}

	normalised := Normalise(l)

	if normalised.City != "Manchester" {
		t.Fatalf("expected city Manchester, got %s", normalised.City)
	}

	if normalised.PropertyType != "flat" {
		t.Fatalf("expected property type flat, got %s", normalised.PropertyType)
	}

	if normalised.Postcode != "M1 4AB" {
		t.Fatalf("expected postcode M1 4AB, got %s", normalised.Postcode)
	}

	if normalised.PostcodeArea != "M1" {
		t.Fatalf("expected postcode area M1, got %s", normalised.PostcodeArea)
	}
}

func TestExtractPostcodeAreaHandlesEmptyPostcode(t *testing.T) {
	area := ExtractPostcodeArea("")

	if area != "" {
		t.Fatalf("expected empty postcode area, got %s", area)
	}
}