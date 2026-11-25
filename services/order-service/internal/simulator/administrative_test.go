package simulator

import "testing"

func TestAdministrativeValidatorAcceptsEveryCataloguePair(t *testing.T) {
	v := defaultAdministrativeValidator
	if len(v.provinces) != 4 || len(v.wards) != 35 {
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
	if rejected != 105 {
		t.Fatalf("expected 105 rejected cross pairs, got %d", rejected)
	}
}

func TestAdministrativeValidatorRejectsUnknownAndLegacyCodes(t *testing.T) {
	v := defaultAdministrativeValidator
	cases := [][2]string{
		{"99", "WARD-TH-001"}, // unknown province
		{"75", "WARD-XX-999"}, // unknown ward
		{"75", "HUE-01"},      // legacy synthetic ward of the old hardcode stays rejected
		{"WARD-TH-001", "75"}, // swapped codes
		{"", ""},              // empty
		{"75", "75"},          // province code used as ward
	}
	for _, c := range cases {
		if e := v.Validate(c[0], c[1]); e == nil {
			t.Errorf("pair %v accepted, want rejection", c)
		}
	}
}

func TestAdministrativeValidatorNames(t *testing.T) {
	v := defaultAdministrativeValidator
	if name, ok := v.ProvinceName("75"); !ok || name != "Thành phố Huế" {
		t.Errorf("province 75 name = %q, %v", name, ok)
	}
	if name, ok := v.WardName("WARD-TH-001"); !ok || name != "Thuận Hòa" {
		t.Errorf("ward WARD-TH-001 name = %q, %v", name, ok)
	}
	if name, ok := v.WardName("WARD-DN-002"); !ok || name != "Hải Châu 1" {
		t.Errorf("ward WARD-DN-002 name = %q, %v", name, ok)
	}
	if _, ok := v.WardName("HUE-01"); ok {
		t.Error("legacy ward HUE-01 must not resolve")
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
