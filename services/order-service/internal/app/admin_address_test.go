package app

import "testing"

func TestAdminAddressUsesSnapshotNames(t *testing.T) {
	for _, tc := range []struct {
		name, ward, province, want string
	}{
		{"snapshot names", "Phường Phú Nhuận", "Thành phố Huế", "1 Lê Lợi, Phường Phú Nhuận, Thành phố Huế"},
		{"historical duplicated name", "Phường Phú Nhuận, Thành phố Huế", "Thành phố Huế", "1 Lê Lợi, Phường Phú Nhuận, Thành phố Huế"},
		{"historical codes", "", "", "1 Lê Lợi, 30109, 75"},
		{"partial snapshot", "Phường Phú Nhuận", "", "1 Lê Lợi, Phường Phú Nhuận, 75"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := formatAdminAddress("1 Lê Lợi", tc.ward, "30109", tc.province, "75")
			if got != tc.want {
				t.Fatalf("address = %q, want %q", got, tc.want)
			}
		})
	}
}
