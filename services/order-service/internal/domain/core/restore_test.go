package core

import "testing"

func TestPersistenceRecordRoundTrip(t *testing.T) {
	for _, index := range []int{0, 1} {
		a, _ := placed(t, index)
		record := a.Record()
		b, err := Restore(a.Snapshot(), record)
		if err != nil || b.State() != a.State() || b.Version() != a.Version() || b.Snapshot().Hash() != a.Snapshot().Hash() {
			t.Fatal("restore", err)
		}
		if record.Owner.CustomerID != nil {
			*record.Owner.CustomerID = "changed"
		} else {
			*record.Owner.GuestSessionID = "changed"
		}
		if a.Owner().Validate() != nil || b.Owner().Validate() != nil {
			t.Fatal("record owner aliased aggregate")
		}
	}
	a, proof := allocated(t)
	a, err := a.ConfirmPaid(contextFor(a, proof.ExpiresAt), proof)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Restore(a.Snapshot(), a.Record())
	if err != nil || !b.HasMilestone(FirstPaid) {
		t.Fatal("lost paid milestone", err)
	}
	for _, change := range []func(*StateRecord){
		func(r *StateRecord) { r.SchemaVersion = 2 }, func(r *StateRecord) { r.OrderID = "other" }, func(r *StateRecord) { r.FirstPaid = false },
		func(r *StateRecord) { r.Version = 0 }, func(r *StateRecord) { r.Allocation.Amount++ }, func(r *StateRecord) { r.Reservation.State = "UNKNOWN" },
		func(r *StateRecord) { r.State = Shipped }, func(r *StateRecord) { r.Blocks = 128 },
	} {
		r := a.Record()
		change(&r)
		if _, err := Restore(a.Snapshot(), r); err == nil {
			t.Fatal("corrupt persistence record accepted")
		}
	}
}
