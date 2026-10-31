package domain

import "testing"

func TestTwoTierDestinationDoesNotInventDistrict(t *testing.T) {
	r := Recipient{Name: "Synthetic", Phone: "0905123456", Street: "1 Test Street", Ward: "Test ward", Province: "Test province", CountryCode: "VN"}
	if err := r.Validate(); err != nil {
		t.Fatal("province and ward address should not require a fabricated district", err)
	}
	r.Ward = ""
	if r.Validate() == nil {
		t.Fatal("missing ward accepted")
	}
}
