package simulator

import "testing"

func TestAdministrativeValidatorAcceptsEveryCataloguePair(t *testing.T) {
	v := defaultAdministrativeValidator
	if len(v.provinces) != 34 || len(v.wards) != 3321 {
		t.Fatalf("dataset shape changed: %d provinces, %d wards", len(v.provinces), len(v.wards))
	}
	for _, w := range v.wards {
		if e := v.Validate(*w.ParentCode, w.Code); e != nil {
			t.Errorf("valid pair province=%s ward=%s rejected: %v", *w.ParentCode, w.Code, e)
		}
	}
}

func TestAdministrativeValidatorRejectsCrossProvincePairs(t *testing.T) {
	v := defaultAdministrativeValidator
	rejected := 0
	for _, w := range v.wards {
		for code := range v.provinces {
			if code == *w.ParentCode {
				continue
			}
			if e := v.Validate(code, w.Code); e == nil {
				t.Errorf("cross pair province=%s ward=%s accepted", code, w.Code)
			} else {
				rejected++
			}
		}
	}
	if rejected != 3321*33 {
		t.Fatalf("expected %d rejected cross pairs, got %d", 3321*33, rejected)
	}
}

func TestAdministrativeValidatorRejectsUnknownAndLegacyCodes(t *testing.T) {
	v := defaultAdministrativeValidator
	cases := [][2]string{
		{"99", "19753"},       // unknown province
		{"46", "99999"},       // unknown ward
		{"75", "19753"},       // ward of Hue (46) filed under Dong Nai (75): cross pair
		{"75", "WARD-TH-001"}, // both codes retired with dataset v1
		{"46", "WARD-TH-001"}, // retired v1 ward under its old province's successor
		{"46", "HUE-01"},      // legacy synthetic ward of the old hardcode stays rejected
		{"19753", "46"},       // swapped codes
		{"", ""},              // empty
		{"46", "46"},          // province code used as ward
	}
	for _, c := range cases {
		if e := v.Validate(c[0], c[1]); e == nil {
			t.Errorf("pair %v accepted, want rejection", c)
		}
	}
}

func TestAdministrativeValidatorNames(t *testing.T) {
	v := defaultAdministrativeValidator
	if name, ok := v.ProvinceName("46"); !ok || name != "Thành phố Huế" {
		t.Errorf("province 46 name = %q, %v", name, ok)
	}
	if name, ok := v.ProvinceName("48"); !ok || name != "Thành phố Đà Nẵng" {
		t.Errorf("province 48 name = %q, %v", name, ok)
	}
	if name, ok := v.WardName("19753"); !ok || name != "Phường Phú Xuân" {
		t.Errorf("ward 19753 name = %q, %v", name, ok)
	}
	if name, ok := v.WardName("20242"); !ok || name != "Phường Hải Châu" {
		t.Errorf("ward 20242 name = %q, %v", name, ok)
	}
	if name, ok := v.ProvinceName("75"); !ok || name != "Tỉnh Đồng Nai" {
		t.Errorf("province 75 name = %q, %v (75 is Dong Nai in the official coding; Hue is 46)", name, ok)
	}
	if _, ok := v.WardName("WARD-TH-001"); ok {
		t.Error("retired ward WARD-TH-001 must not resolve")
	}
	if _, ok := v.WardName("HUE-01"); ok {
		t.Error("legacy ward HUE-01 must not resolve")
	}
}

func TestAdministrativeDatasetVersion(t *testing.T) {
	if got := defaultAdministrativeValidator.Version(); got != "2" {
		t.Fatalf("dataset version = %q, want 2", got)
	}
}

func TestAdministrativeChecksumStable(t *testing.T) {
	a, e := loadAdministrativeValidator(administrativeUnitsJSON)
	if e != nil {
		t.Fatal(e)
	}
	b, e := loadAdministrativeValidator(administrativeUnitsJSON)
	if e != nil {
		t.Fatal(e)
	}
	if a.Checksum() == "" || a.Checksum() != b.Checksum() {
		t.Fatalf("checksum unstable: %q vs %q", a.Checksum(), b.Checksum())
	}
	if a.Version() != defaultAdministrativeValidator.Version() {
		t.Fatal("version mismatch")
	}
}
