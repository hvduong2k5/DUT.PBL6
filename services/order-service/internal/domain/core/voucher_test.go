package core

import (
	"encoding/json"
	"testing"
	"time"
)

func voucherFixture(t *testing.T) SnapshotInput {
	s := snapshotFixture(t, 0).Data()
	s.SchemaVersion = "1.1"
	policy, id := "promotion-voucher-1", "10000000-0000-4000-8000-000000000001"
	expires := s.PlacedAt.Add(20 * time.Minute)
	s.Policies.Promotion = &policy
	s.Voucher = &VoucherTerms{Code: "WELCOME", Kind: "FIXED_AMOUNT", Amount: 1000, Policy: policy, ValidUntil: expires, HoldID: &id, HoldExpiresAt: &expires}
	s.Money.MerchandiseDiscount = 1000
	s.Money.FinalAmount -= 1000
	return s
}

func TestVoucherSnapshotsPreserveLegacyAndBindDiscounts(t *testing.T) {
	// The existing golden fixtures still exercise exact legacy bytes and hashes.
	s := voucherFixture(t)
	if _, err := NewSnapshot(s); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*SnapshotInput){
		"old schema with voucher":   func(s *SnapshotInput) { s.SchemaVersion = "1.0" },
		"wrong discount":            func(s *SnapshotInput) { s.Voucher.Amount++ },
		"wrong policy":              func(s *SnapshotInput) { s.Voucher.Policy = "unknown" },
		"expired quote":             func(s *SnapshotInput) { s.Voucher.ValidUntil = s.PlacedAt },
		"hold shorter than payment": func(s *SnapshotInput) { v := s.PlacedAt.Add(time.Minute); s.Voucher.HoldExpiresAt = &v },
		"noncanonical code":         func(s *SnapshotInput) { s.Voucher.Code = "welcome" },
		"wrong discount bucket":     func(s *SnapshotInput) { s.Voucher.Kind = "FREE_SHIPPING" },
	} {
		t.Run(name, func(t *testing.T) {
			s := voucherFixture(t)
			mutate(&s)
			if _, err := NewSnapshot(s); err == nil {
				t.Fatal("accepted incoherent voucher terms")
			}
		})
	}
	s = snapshotFixture(t, 0).Data()
	raw, _ := json.Marshal(s)
	var obj map[string]any
	_ = json.Unmarshal(raw, &obj)
	obj["unrecognized_voucher"] = nil
	raw, _ = json.Marshal(obj)
	if _, err := DecodeSnapshot(raw); err == nil {
		t.Fatal("unknown key must not replace omitted voucher")
	}
}

func TestVoucherPlacementRequiresHoldEvidence(t *testing.T) {
	s := voucherFixture(t)
	s.Voucher.HoldID = nil
	s.Voucher.HoldExpiresAt = nil
	snapshot, err := NewSnapshot(s)
	if err != nil {
		t.Fatal("quote snapshot may omit hold evidence", err)
	}
	_, proof := placed(t, 0)
	if _, err = Place(snapshot, s.OrderID, proof, s.PlacedAt); err == nil {
		t.Fatal("placement accepted an unheld voucher")
	}
}
