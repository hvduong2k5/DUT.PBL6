package persistence

import (
	"context"
	"strings"
	"sync"
	"testing"
)

type packingCleanup struct{}

func (packingCleanup) Cleanup(context.Context, string, string, int64) (string, error) {
	return "REMOVED", nil
}
func TestPackingPermitAuthorityReplayAndExpiry(t *testing.T) {
	h, p, a, tr := paymentFixture(t)
	digest := string(a.Snapshot().Data().Provenance.ItemsDigest)
	if _, _, e := h.store.PackingInputs(h.ctx, a.ID(), 1, digest, "permit-test-key-01", "packing-test-v1", nil); e == nil {
		t.Fatal("unpaid order received packing permit")
	}
	if e := p.Ingest(h.ctx, tr); e != nil {
		t.Fatal(e)
	}
	if _, e := p.WorkFinalize(h.ctx, inventoryProof{}); e != nil {
		t.Fatal(e)
	}
	runtime := &CheckoutRuntime{Store: h.store}
	for i := 0; i < 3; i++ {
		if _, e := runtime.WorkFollowup(h.ctx, packingCleanup{}); e != nil {
			t.Fatal(e)
		}
	}
	permits := make(chan PackingPermit, 6)
	errs := make(chan error, 6)
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, permit, e := h.store.PackingInputs(h.ctx, a.ID(), 1, digest, "permit-test-key-01", "packing-test-v1", nil)
			permits <- permit
			errs <- e
		}()
	}
	wg.Wait()
	close(permits)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	var first PackingPermit
	for permit := range permits {
		if first.ID == "" {
			first = permit
		}
		if permit.ID != first.ID || !permit.Expires.Equal(first.Expires) {
			t.Fatal("concurrent retry changed permit identity/expiry")
		}
	}
	if h.count(t, "SELECT count(*) FROM fulfillment_permits") != 1 {
		t.Fatal("duplicate permit")
	}
	if _, _, e := h.store.PackingInputs(h.ctx, a.ID(), 1, digest, first.Key, first.Policy, &first); e != nil {
		t.Fatal("original permit no longer valid", e)
	}
	wrong := first
	wrong.Warehouse = "different-warehouse"
	if _, _, e := h.store.PackingInputs(h.ctx, a.ID(), 1, digest, first.Key, first.Policy, &wrong); e == nil {
		t.Fatal("changed permit accepted")
	}
	if _, _, e := h.store.PackingInputs(h.ctx, a.ID(), 2, digest, first.Key, first.Policy, nil); e == nil {
		t.Fatal("wrong generation accepted")
	}
	if _, _, e := h.store.PackingInputs(h.ctx, a.ID(), 1, strings.Repeat("b", 64), first.Key, first.Policy, nil); e == nil {
		t.Fatal("wrong digest accepted")
	}
	if _, e := h.db.Exec(h.ctx, "UPDATE orders SET has_active_case=true WHERE id=$1", a.ID()); e != nil {
		t.Fatal(e)
	}
	if _, _, e := h.store.PackingInputs(h.ctx, a.ID(), 1, digest, first.Key, first.Policy, &first); e == nil {
		t.Fatal("blocking case ignored")
	}
	if _, e := h.db.Exec(h.ctx, "UPDATE orders SET has_active_case=false WHERE id=$1", a.ID()); e != nil {
		t.Fatal(e)
	}
	if _, e := h.db.Exec(h.ctx, "UPDATE fulfillment_permits SET expires_at=clock_timestamp()-interval '1 second'"); e != nil {
		t.Fatal(e)
	}
	if _, _, e := h.store.PackingInputs(h.ctx, a.ID(), 1, digest, first.Key, first.Policy, nil); e == nil {
		t.Fatal("replay renewed expired permit")
	}
}
