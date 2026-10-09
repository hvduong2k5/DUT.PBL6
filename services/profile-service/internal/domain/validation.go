package domain

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var e164 = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
var vnMobile = regexp.MustCompile(`^0[35789][0-9]{8}$`)

func NormalizePhone(raw string) (string, error) {
	p := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(strings.TrimSpace(raw))
	if vnMobile.MatchString(p) {
		p = "+84" + p[1:]
	}
	if !e164.MatchString(p) {
		return "", &ValidationError{"phone_number", "must be an E.164 number or Vietnamese mobile number"}
	}
	return p, nil
}
func ValidateText(field, value string, max int) error {
	if strings.TrimSpace(value) == "" || utf8.RuneCountInString(value) > max {
		return &ValidationError{field, "must be non-empty and within the maximum length"}
	}
	return nil
}
func ValidateAddress(a *ShippingAddress) error {
	a.RecipientName = strings.TrimSpace(a.RecipientName)
	a.StreetAddress = strings.TrimSpace(a.StreetAddress)
	for _, f := range []struct {
		name, value string
		max         int
	}{
		{"recipient_name", a.RecipientName, 150}, {"street_address", a.StreetAddress, 255},
		{"ward_code", a.WardCode, 20}, {"province_code", a.ProvinceCode, 20},
	} {
		if err := ValidateText(f.name, f.value, f.max); err != nil {
			return err
		}
	}
	p, err := NormalizePhone(a.PhoneNumber)
	if err != nil {
		return err
	}
	a.PhoneNumber = p
	label, err := NormalizeAddressLabel(a.Label)
	if err != nil {
		return &ValidationError{"label", "invalid address label"}
	}
	a.Label = label
	if (a.Latitude == nil) != (a.Longitude == nil) {
		return &ValidationError{"coordinates", "latitude and longitude must be supplied together"}
	}
	if a.Latitude != nil && !(AddressCoordinates{Latitude: *a.Latitude, Longitude: *a.Longitude}).IsValid() {
		return &ValidationError{"coordinates", "outside supported bounds"}
	}
	return nil
}
