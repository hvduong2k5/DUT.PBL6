package domain

import "testing"

func TestMoneyBoundaries(t *testing.T) {
	cases := []struct {
		name  string
		items []Item
		want  int64
		valid bool
	}{
		{"canonical", []Item{{SKU: "A", Quantity: 2, UnitPrice: 110000}}, 220000, true},
		{"zero quantity", []Item{{SKU: "A", Quantity: 0, UnitPrice: 110000}}, 0, false},
		{"overflow", []Item{{SKU: "A", Quantity: 999, UnitPrice: MaxMoney}}, 0, false},
		{"duplicate SKU", []Item{{SKU: "A", Quantity: 1, UnitPrice: 1}, {SKU: "A", Quantity: 1, UnitPrice: 1}}, 0, false},
		{"sum overflow", []Item{{SKU: "A", Quantity: 1, UnitPrice: MaxMoney}, {SKU: "B", Quantity: 1, UnitPrice: 1}}, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, e := Total(c.items)
			if (e == nil) != c.valid || got != c.want {
				t.Fatalf("got %d valid=%v", got, e == nil)
			}
		})
	}
}
func TestLifecycleGuards(t *testing.T) {
	for _, pair := range [][2]string{{"PAID", "PENDING_PAYMENT"}, {"CANCELLED_TIMEOUT", "PAID"}, {"SHIPPED", "PACKED"}, {"COMPLETED", "PROCESSING"}, {"CONFIRMED_COD", "PAID"}} {
		if StateAllowed(pair[0], pair[1]) {
			t.Fatalf("unsafe transition %v", pair)
		}
	}
	for _, pair := range [][2]string{{"PENDING_PAYMENT", "PAYMENT_FINALIZING"}, {"PAYMENT_FINALIZING", "PAID"}, {"CONFIRMED_COD", "PROCESSING"}, {"SHIPPED", "DELIVERED"}} {
		if !StateAllowed(pair[0], pair[1]) {
			t.Fatalf("expected transition %v", pair)
		}
	}
}
