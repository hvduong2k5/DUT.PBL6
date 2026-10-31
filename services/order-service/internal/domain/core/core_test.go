package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fixture struct {
	Name         string          `json:"name"`
	Snapshot     json.RawMessage `json:"snapshot"`
	ExpectedHash PayloadHash     `json:"expected_logical_hash"`
}

func fixtures(t *testing.T) []fixture {
	t.Helper()
	p := filepath.Join("..", "..", "..", "..", "..", "docs", "03_api_specs", "order-snapshot.v1.examples.json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Cases []fixture `json:"cases"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Cases) != 2 {
		t.Fatal("missing contract fixtures")
	}
	return f.Cases
}

func snapshotFixture(t *testing.T, index int) Snapshot {
	t.Helper()
	s, err := DecodeSnapshot(fixtures(t)[index].Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func placed(t *testing.T, index int) (Aggregate, ResourceProof) {
	t.Helper()
	s := snapshotFixture(t, index)
	d := s.Data()
	p := ResourceProof{OrderID: d.OrderID, ReservationID: "reservation-fixture", WarehouseID: d.WarehouseID,
		ItemsDigest: d.Provenance.ItemsDigest, State: "RESERVED", ResourceVersion: 1, DecisionID: "reserve-decision",
		DecisionAt: d.PlacedAt.Add(-time.Second), ExpiresAt: d.PlacedAt.Add(15 * time.Minute)}
	if d.Payment.Method == "COD" {
		p.State = "COMMITTED"
	}
	a, err := Place(s, d.OrderID, p, d.PlacedAt)
	if err != nil {
		t.Fatal(err)
	}
	return a, p
}

func contextFor(a Aggregate, at time.Time) DecisionContext {
	return DecisionContext{ExpectedVersion: a.Version(), At: at}
}

func allocationFor(a Aggregate) AllocationEvidence {
	s := a.Snapshot().Data()
	return AllocationEvidence{OrderID: a.ID(), AllocationID: "70000000-0000-4000-8000-000000000001",
		ReceiptID: "80000000-0000-4000-8000-000000000001", ReceiverProfileID: *s.Payment.ReceiverProfileID,
		Currency: "VND", Amount: s.Money.FinalAmount}
}

func allocated(t *testing.T) (Aggregate, ResourceProof) {
	t.Helper()
	a, proof := placed(t, 0)
	a, err := a.Allocate(contextFor(a, a.Snapshot().Data().PlacedAt.Add(time.Second)), allocationFor(a))
	if err != nil {
		t.Fatal(err)
	}
	proof.State, proof.ResourceVersion, proof.DecisionID = "COMMITTED", 2, "commit-decision"
	proof.DecisionAt = proof.ExpiresAt.Add(-time.Second)
	return a, proof
}

func TestMoneyCheckedArithmetic(t *testing.T) {
	for _, tc := range []struct {
		currency string
		amount   int64
	}{{"USD", 1}, {"VND", -1}, {"VND", MaxOrderAmount + 1}, {"VND", math.MaxInt64}} {
		if _, err := NewMoney(tc.currency, tc.amount); !errors.Is(err, ErrMoney) {
			t.Fatalf("missing amount guard: %v", err)
		}
	}
	max, _ := NewMoney("VND", MaxOrderAmount)
	one, _ := NewMoney("VND", 1)
	if _, err := max.Add(one); !errors.Is(err, ErrOverflow) {
		t.Fatal(err)
	}
	if _, err := max.Multiply(math.MaxInt64); !errors.Is(err, ErrOverflow) {
		t.Fatal(err)
	}
	if _, err := one.Multiply(-1); !errors.Is(err, ErrMoney) {
		t.Fatal(err)
	}
	zero, err := NewMoney("VND", 0)
	if err != nil || zero.Currency() != "VND" {
		t.Fatal("zero shipping fee must be valid")
	}
	price, _ := NewMoney("VND", 75000)
	line, err := price.Multiply(2)
	if err != nil || line.Amount() != 150000 || price.Amount() != 75000 {
		t.Fatal("checked immutable multiplication")
	}
	if _, err := max.Add(zero); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotGoldenAndDefensiveCopies(t *testing.T) {
	for _, f := range fixtures(t) {
		t.Run(f.Name, func(t *testing.T) {
			s, err := DecodeSnapshot(f.Snapshot)
			if err != nil || s.Hash() != f.ExpectedHash {
				t.Fatalf("golden mismatch: %v %s", err, s.Hash())
			}
			input := s.Data()
			other, err := NewSnapshot(input)
			if err != nil || other.Hash() != s.Hash() {
				t.Fatal("typed round trip", err)
			}
			input.Items[0].Name = "changed"
			if input.Owner.CustomerID != nil {
				*input.Owner.CustomerID = "changed"
			} else {
				*input.Owner.GuestSessionID = "changed"
			}
			data := other.Data()
			data.Items[0].Quantity++
			raw := other.Bytes()
			raw[0] = '!'
			if other.Hash() != s.Hash() || !bytes.Equal(other.Bytes(), s.Bytes()) || other.Data().Items[0].Name == "changed" {
				t.Fatal("snapshot aliased mutable input/output")
			}
			input = s.Data()
			input.Items[0], input.Items[1] = input.Items[1], input.Items[0]
			reordered, err := NewSnapshot(input)
			if err != nil || reordered.Hash() != s.Hash() {
				t.Fatal("SKU ordering must be canonical", err)
			}
		})
	}
}

func TestSnapshotSemanticRejections(t *testing.T) {
	mutations := map[string]func(*SnapshotInput){
		"version":                  func(s *SnapshotInput) { s.SchemaVersion = "2.0" },
		"owner discriminator":      func(s *SnapshotInput) { s.Owner.GuestSessionID = s.Owner.CustomerID },
		"nil UUID":                 func(s *SnapshotInput) { s.OrderID = "00000000-0000-0000-0000-000000000000" },
		"line arithmetic":          func(s *SnapshotInput) { s.Items[0].LineSubtotal++ },
		"subtotal":                 func(s *SnapshotInput) { s.Money.Subtotal++ },
		"final":                    func(s *SnapshotInput) { s.Money.FinalAmount++ },
		"fee authority":            func(s *SnapshotInput) { s.Shipping.QuotedFee++ },
		"currency":                 func(s *SnapshotInput) { s.Money.Currency = "USD" },
		"overflow":                 func(s *SnapshotInput) { s.Items[0].Quantity = math.MaxInt64 },
		"duplicate SKU":            func(s *SnapshotInput) { s.Items[1].SKU = s.Items[0].SKU },
		"duplicate line":           func(s *SnapshotInput) { s.Items[1].LineID = s.Items[0].LineID },
		"catalog coherence":        func(s *SnapshotInput) { s.Items[0].CatalogVersion = "other" },
		"missing packaging":        func(s *SnapshotInput) { s.Items[0].Packaging.Version = "" },
		"zero weight":              func(s *SnapshotInput) { s.Items[0].WeightGrams = 0 },
		"underweight parcel":       func(s *SnapshotInput) { s.Shipping.PackageWeightGrams = 1 },
		"promotion disabled":       func(s *SnapshotInput) { v := "enabled"; s.Policies.Promotion = &v },
		"discount disabled":        func(s *SnapshotInput) { s.Money.MerchandiseDiscount = 1 },
		"selection binding":        func(s *SnapshotInput) { s.Provenance.SelectionDigest = SelectionHash(strings.Repeat("a", 64)) },
		"manifest binding":         func(s *SnapshotInput) { s.Provenance.ItemsDigest = ManifestHash(strings.Repeat("a", 64)) },
		"quote hash":               func(s *SnapshotInput) { s.Provenance.QuoteHash = "bad" },
		"chronology":               func(s *SnapshotInput) { s.Provenance.CanonicalizedAt = s.PlacedAt.Add(time.Second) },
		"catalog expiry equality":  func(s *SnapshotInput) { s.Catalog.ValidUntil = s.PlacedAt },
		"shipping expiry equality": func(s *SnapshotInput) { s.Shipping.ValidUntil = s.PlacedAt },
		"payment expiry equality":  func(s *SnapshotInput) { s.Payment.ExpiresAt = &s.PlacedAt },
		"address ownership source": func(s *SnapshotInput) { s.Address.Source.AddressVersion = nil },
		"local phone":              func(s *SnapshotInput) { s.Address.PhoneE164 = "0905123456" },
		"blank receiver":           func(s *SnapshotInput) { s.Payment.ReceiverProfileID = nil },
		"non NFC":                  func(s *SnapshotInput) { s.Items[0].Name = "Me\u0300 xung" },
		"invalid UTF8":             func(s *SnapshotInput) { s.Items[0].Name = string([]byte{0xff}) },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			s := snapshotFixture(t, 0).Data()
			mutate(&s)
			if _, err := NewSnapshot(s); !errors.Is(err, ErrSnapshot) {
				t.Fatalf("accepted invalid snapshot: %v", err)
			}
		})
	}
	s := snapshotFixture(t, 1).Data()
	*s.Payment.CollectAmount++
	if _, err := NewSnapshot(s); err == nil {
		t.Fatal("COD collection must equal final amount")
	}
}

func TestSnapshotStrictJSON(t *testing.T) {
	raw := fixtures(t)[0].Snapshot
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	delete(obj, "snapshot_id")
	missing, _ := json.Marshal(obj)
	for _, b := range [][]byte{missing, append(append([]byte(nil), raw...), []byte(` {}`)...), bytes.Replace(raw, []byte(`"snapshot_schema_version": "1.0"`), []byte(`"snapshot_schema_version":"1.0","snapshot_schema_version":"1.0"`), 1)} {
		if _, err := DecodeSnapshot(b); err == nil {
			t.Fatal("accepted ambiguous/incomplete JSON")
		}
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	obj["status"] = json.RawMessage(`"PAID"`)
	extra, _ := json.Marshal(obj)
	if _, err := DecodeSnapshot(extra); err == nil {
		t.Fatal("snapshot cannot contain mutable status")
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	delete(obj, "status")
	var money map[string]json.RawMessage
	_ = json.Unmarshal(obj["money"], &money)
	delete(money, "merchandise_discount")
	obj["money"], _ = json.Marshal(money)
	missingZero, _ := json.Marshal(obj)
	if _, err := DecodeSnapshot(missingZero); err == nil {
		t.Fatal("missing zero differs from explicit zero")
	}
}

func TestCanonicalProfileRejectsAmbiguityAndPreservesIntegers(t *testing.T) {
	for _, raw := range []string{`{"a":1,"a":2}`, `{"n":1.0}`, `{"n":1e0}`, `{"n":-0}`, `{"n":9223372036854775808}`, `{"s":"\ud800"}`, `{"s":"\udc00"}`, `{"s":"e\u0301"}`, `{} []`} {
		if _, err := parseJSON([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	v, err := parseJSON([]byte(`{"b":"Mè <xửng>\u2028🧾","a":9007199254740993}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := canonicalValue(v)
	if err != nil || string(b) != "{\"a\":9007199254740993,\"b\":\"Mè <xửng>\u2028🧾\"}" {
		t.Fatalf("canonical bytes %s: %v", b, err)
	}
	if _, err := parseJSON([]byte(`{"s":"\ud83e\uddfe"}`)); err != nil {
		t.Fatal("valid surrogate pair", err)
	}
}

func TestManifestAndCommandIdentity(t *testing.T) {
	s := snapshotFixture(t, 0).Data()
	lines := []SelectionLine{{1, s.Items[1].SKU}, {2, s.Items[0].SKU}}
	h, err := HashManifest(lines)
	if err != nil || h != s.Provenance.ItemsDigest || lines[0].SKU != s.Items[1].SKU {
		t.Fatal("wire manifest profile or input mutation", err)
	}
	if _, err := HashManifest([]SelectionLine{{1, "A"}, {1, "A"}}); err == nil {
		t.Fatal("unconsolidated manifest")
	}
	first, err := NewCommandIdentity("customer:demo", "cancel", s.OrderID, "command-key-0001", []byte(`{"reason":"changed","version":1}`))
	if err != nil {
		t.Fatal(err)
	}
	reordered, err := NewCommandIdentity("customer:demo", "cancel", s.OrderID, "command-key-0001", []byte(`{"version":1,"reason":"changed"}`))
	if err != nil || first.CheckReplay(reordered) != nil {
		t.Fatal("same request must replay", err)
	}
	changed, _ := NewCommandIdentity("customer:demo", "cancel", s.OrderID, "command-key-0001", []byte(`{"reason":"changed","version":2}`))
	if first.CheckReplay(changed) == nil {
		t.Fatal("key reuse with changed payload")
	}
	for _, key := range []string{"short", "command key 0001", "command-key-000🧾", strings.Repeat("a", 129)} {
		if _, err := NewCommandIdentity("scope", "cancel", s.OrderID, key, []byte(`{}`)); err == nil {
			t.Fatal("invalid key")
		}
	}
	one, _ := NewCommandIdentity("scope", "cancel", s.OrderID, "command-key-0001", []byte(`{"n":9007199254740992}`))
	two, _ := NewCommandIdentity("scope", "cancel", s.OrderID, "command-key-0001", []byte(`{"n":9007199254740993}`))
	if one.RequestHash() == two.RequestHash() {
		t.Fatal("integer hashes rounded")
	}
}

func TestPaymentDecisionDeadlinesAndMilestone(t *testing.T) {
	for _, delta := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
		a, _ := placed(t, 0)
		at := a.Snapshot().Data().Payment.ExpiresAt.Add(delta)
		result, err := a.Allocate(contextFor(a, at), allocationFor(a))
		if delta < 0 {
			if err != nil || result.State() != PaymentFinalizing {
				t.Fatal("before deadline", err)
			}
		} else if !errors.Is(err, ErrDeadline) || result.State() != PendingPayment {
			t.Fatal("equality must be expired", err)
		}
	}
	a, proof := allocated(t)
	late := proof.ExpiresAt.Add(time.Minute)
	paid, err := a.ConfirmPaid(contextFor(a, late), proof)
	if err != nil || paid.State() != Paid || !paid.HasMilestone(FirstPaid) || !paid.ReadyEligible() {
		t.Fatal("late query of pre-expiry commit", err)
	}
	if a.HasMilestone(FirstPaid) || a.State() != PaymentFinalizing {
		t.Fatal("mutated source aggregate")
	}
	if _, err := paid.ConfirmPaid(contextFor(paid, late), proof); !errors.Is(err, ErrState) {
		t.Fatal("second FIRST_PAID", err)
	}
	proof.DecisionAt = proof.ExpiresAt
	if _, err := a.ConfirmPaid(contextFor(a, late), proof); !errors.Is(err, ErrProof) {
		t.Fatal("commit at expiry invalid", err)
	}
}

func TestPaymentGuardsAndResourceAssociation(t *testing.T) {
	for _, block := range []Blocker{HoldBlock, CancellationBlock, RefundBlock} {
		a, proof := allocated(t)
		blocked, err := a.Block(contextFor(a, a.decidedAt), block)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := blocked.ConfirmPaid(contextFor(blocked, proof.ExpiresAt), proof); !errors.Is(err, ErrBlocked) {
			t.Fatal("late effect must not override control", err)
		}
		if blocked.ReadyEligible() {
			t.Fatal("blocked order ready")
		}
	}
	for name, mutate := range map[string]func(*ResourceProof){
		"order": func(p *ResourceProof) { p.OrderID = "other" }, "warehouse": func(p *ResourceProof) { p.WarehouseID = "other" },
		"digest": func(p *ResourceProof) { p.ItemsDigest = "wrong" }, "reservation": func(p *ResourceProof) { p.ReservationID = "other" },
		"stale version": func(p *ResourceProof) { p.ResourceVersion = 0 }, "same-version conflict": func(p *ResourceProof) { p.ResourceVersion = 1 },
		"no decision": func(p *ResourceProof) { p.DecisionID = "" }, "unknown": func(p *ResourceProof) { p.State = "UNKNOWN" },
	} {
		t.Run(name, func(t *testing.T) {
			a, p := allocated(t)
			mutate(&p)
			if _, err := a.ConfirmPaid(contextFor(a, p.ExpiresAt.Add(time.Second)), p); !errors.Is(err, ErrProof) {
				t.Fatal(err)
			}
		})
	}
	a, p := allocated(t)
	p.ExpiresAt = p.ExpiresAt.UTC()
	p.DecisionAt = p.DecisionAt.UTC()
	if _, err := a.ConfirmPaid(contextFor(a, p.ExpiresAt.Add(time.Second)), p); err != nil {
		t.Fatal("same instant in different timezone", err)
	}
	b, _ := placed(t, 0)
	e := allocationFor(b)
	e.Amount--
	if _, err := b.Allocate(contextFor(b, b.decidedAt), e); !errors.Is(err, ErrPayment) {
		t.Fatal("underpayment", err)
	}
}

func TestCancelTimeoutVersionOwnerAndNoRevival(t *testing.T) {
	a, proof := placed(t, 0)
	if _, err := a.Timeout(contextFor(a, a.decidedAt)); !errors.Is(err, ErrDeadline) {
		t.Fatal(err)
	}
	if _, err := a.Allocate(DecisionContext{ExpectedVersion: 0, At: a.decidedAt}, allocationFor(a)); !errors.Is(err, ErrVersion) {
		t.Fatal(err)
	}
	owner := a.Owner()
	other := a.Owner()
	*other.CustomerID = "90000000-0000-4000-8000-000000000001"
	if _, err := a.CancelPending(contextFor(a, a.decidedAt), other); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	cancelled, err := a.CancelPending(contextFor(a, a.decidedAt), owner)
	if err != nil || cancelled.State() != CancelledByUser {
		t.Fatal(err)
	}
	if _, err := cancelled.Allocate(contextFor(cancelled, proof.ExpiresAt), allocationFor(a)); !errors.Is(err, ErrState) {
		t.Fatal("late receipt revived order", err)
	}
	held, err := a.Block(contextFor(a, a.decidedAt), HoldBlock)
	if err != nil {
		t.Fatal(err)
	}
	expired, err := held.Timeout(contextFor(held, *a.Snapshot().Data().Payment.ExpiresAt))
	if err != nil || expired.State() != CancelledTimeout {
		t.Fatal("hold must not extend deadline", err)
	}
	atDeadline, err := a.CancelPending(contextFor(a, *a.Snapshot().Data().Payment.ExpiresAt), owner)
	if err != nil || atDeadline.State() != CancelledTimeout {
		t.Fatal("expired owner cancel disposition", err)
	}
	finalizing, p := allocated(t)
	if _, err := finalizing.Timeout(contextFor(finalizing, p.ExpiresAt)); !errors.Is(err, ErrState) {
		t.Fatal("blind timeout of finalizing", err)
	}
	if _, err := finalizing.CancelUncommittedFinalization(contextFor(finalizing, p.ExpiresAt), p); !errors.Is(err, ErrProof) {
		t.Fatal("committed must not release", err)
	}
	p.State = "EXPIRED"
	p.DecisionAt = p.ExpiresAt
	failed, err := finalizing.CancelUncommittedFinalization(contextFor(finalizing, p.ExpiresAt), p)
	if err != nil || failed.State() != CancelledTimeout || failed.allocation.ReceiptID == "" || failed.ReadyEligible() {
		t.Fatal("preserve allocated money", err)
	}
	if _, err := finalizing.ConfirmPaid(contextFor(finalizing, finalizing.decidedAt.Add(-time.Nanosecond)), p); !errors.Is(err, ErrDeadline) {
		t.Fatal("decision time regressed", err)
	}
}

func TestCODPlacementAndSafetyMargin(t *testing.T) {
	cod, proof := placed(t, 1)
	if cod.State() != ConfirmedCOD || cod.HasMilestone(FirstPaid) || !cod.ReadyEligible() {
		t.Fatal("COD is unpaid but fulfillment eligible")
	}
	if _, err := cod.Timeout(contextFor(cod, proof.ExpiresAt)); !errors.Is(err, ErrState) {
		t.Fatal("COD has no online payment timeout", err)
	}
	a, p := placed(t, 0)
	s := a.Snapshot().Data()
	expiry := s.PlacedAt.Add(30 * time.Second)
	s.Payment.ExpiresAt = &expiry
	snapshot, err := NewSnapshot(s)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Place(snapshot, s.OrderID, p, s.PlacedAt); !errors.Is(err, ErrDeadline) {
		t.Fatal("placement requires more than30s", err)
	}
}

func TestLifecycleGraphAndCompletionGuard(t *testing.T) {
	states := []State{PendingPayment, PaymentFinalizing, Paid, ConfirmedCOD, Processing, Packed, Shipped, Delivered, DeliveryFailed, Completed, CancelledTimeout, CancelledByUser, CancelledByAdmin}
	for _, from := range states {
		for _, to := range states {
			if from.Terminal() && StateAllowed(from, to) {
				t.Fatalf("terminal revival %s->%s", from, to)
			}
		}
	}
	if !StateAllowed(PendingPayment, CancelledByAdmin) || StateAllowed(ConfirmedCOD, Paid) || StateAllowed(Shipped, Packed) {
		t.Fatal("target lifecycle graph")
	}
	a, _ := placed(t, 0)
	now := a.decidedAt
	e := CompletionEligibility{State: Delivered, At: now.Add(24 * time.Hour), DeliveredAt: now, WindowEndsAt: now.Add(24 * time.Hour),
		CareOrderID: a.ID(), OrderID: a.ID(), CareDecisionID: "care-decision", CareAllowed: true, SettlementVerified: true}
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*CompletionEligibility){func(e *CompletionEligibility) { e.UnresolvedRefund = true }, func(e *CompletionEligibility) { e.ActiveCase = true }, func(e *CompletionEligibility) { e.UnresolvedStock = true }, func(e *CompletionEligibility) { e.BlockingHold = true }, func(e *CompletionEligibility) { e.CancellationPending = true }, func(e *CompletionEligibility) { e.CareOrderID = "other" }, func(e *CompletionEligibility) { e.SettlementVerified = false }, func(e *CompletionEligibility) { e.At = e.WindowEndsAt.Add(-time.Nanosecond) }} {
		bad := e
		mutate(&bad)
		if err := bad.Validate(); err == nil {
			t.Fatal("unsafe completion")
		}
	}
}

func TestEffectIdentityCannotChangeAcrossRetries(t *testing.T) {
	orderID := snapshotFixture(t, 0).Data().OrderID
	key := "checkout:operation-fixture:reserve"
	hash := strings.Repeat("a", 64)
	first, err := NewEffectIdentity("inventory-service", "order-service", "ReserveStock", orderID, key, hash)
	if err != nil {
		t.Fatal(err)
	}
	retry, _ := NewEffectIdentity("inventory-service", "order-service", "ReserveStock", orderID, key, hash)
	if first.CheckReplay(retry) != nil || retry.Key() != key || retry.PlannedOrderID() != orderID {
		t.Fatal("retry changed remote identity")
	}
	changed, _ := NewEffectIdentity("inventory-service", "order-service", "ReserveStock", orderID, key, strings.Repeat("b", 64))
	if first.CheckReplay(changed) == nil {
		t.Fatal("same key with altered remote request")
	}
	if _, err := NewEffectIdentity("inventory-service", "order-service", "ReserveStock", orderID, key, "bad"); err == nil {
		t.Fatal("invalid hash")
	}
	if _, err := NewCommandIdentity("bad\nscope", "cancel", orderID, "command-key-0001", []byte(`{}`)); err == nil {
		t.Fatal("control character scope")
	}
}

func TestApprovedPendingCancellationAndSerializedDecisionGuard(t *testing.T) {
	a, _ := placed(t, 0)
	ctx := contextFor(a, a.decidedAt)
	approval := CancellationApproval{OrderID: a.ID(), OrderVersion: a.Version(), DecisionID: "care-decision",
		PolicyVersion: a.Snapshot().Data().Policies.Order, Scope: "ORDER_CANCEL_PENDING", Allowed: true}
	for _, mutate := range []func(*CancellationApproval){
		func(p *CancellationApproval) { p.OrderID = "other" }, func(p *CancellationApproval) { p.OrderVersion++ },
		func(p *CancellationApproval) { p.PolicyVersion = "other" }, func(p *CancellationApproval) { p.Scope = "REFUND" },
		func(p *CancellationApproval) { p.Allowed = false }, func(p *CancellationApproval) { p.DecisionID = "" },
	} {
		bad := approval
		mutate(&bad)
		if _, err := a.CancelApprovedPending(ctx, bad); !errors.Is(err, ErrProof) {
			t.Fatal("unbound approval", err)
		}
	}
	cancelled, err := a.CancelApprovedPending(ctx, approval)
	if err != nil || cancelled.State() != CancelledByAdmin {
		t.Fatal(err)
	}
	finalizing, err := a.Allocate(ctx, allocationFor(a))
	if err != nil {
		t.Fatal(err)
	}
	// SQL will serialize these requests. Re-evaluation on the newer root must
	// reject the version read by the loser, not silently release held stock.
	if _, err := finalizing.CancelPending(ctx, a.Owner()); !errors.Is(err, ErrVersion) {
		t.Fatal("stale competing decision", err)
	}
	if _, err := cancelled.Allocate(ctx, allocationFor(a)); !errors.Is(err, ErrVersion) {
		t.Fatal("stale receipt decision", err)
	}
}

func TestConcurrentSnapshotAndOwnerViews(t *testing.T) {
	a, _ := placed(t, 0)
	original := a.Snapshot().Hash()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				data := a.Snapshot().Data()
				data.Items[0].Name = "local mutation"
				owner := a.Owner()
				*owner.CustomerID = "local mutation"
				raw := a.Snapshot().Bytes()
				raw[0] = '!'
				if a.Snapshot().Hash() != original || !validID(*a.Owner().CustomerID) || a.Snapshot().Data().Items[0].Name == "local mutation" {
					t.Error("concurrent view mutated aggregate")
					return
				}
			}
		}()
	}
	wg.Wait()
}

func FuzzDecodeSnapshot(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"snapshot_schema_version":"1.0","snapshot_schema_version":"1.0"}`))
	f.Add([]byte(`{"x":"\ud800"}`))
	p := filepath.Join("..", "..", "..", "..", "..", "docs", "03_api_specs", "order-snapshot.v1.examples.json")
	b, err := os.ReadFile(p)
	if err != nil {
		f.Fatal(err)
	}
	var data struct {
		Cases []fixture `json:"cases"`
	}
	if err := json.Unmarshal(b, &data); err != nil {
		f.Fatal(err)
	}
	for _, c := range data.Cases {
		f.Add([]byte(c.Snapshot))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		s, err := DecodeSnapshot(b)
		if err != nil {
			return
		}
		again, err := DecodeSnapshot(s.Bytes())
		if err != nil || again.Hash() != s.Hash() {
			t.Fatal("accepted snapshot is not canonical/idempotent")
		}
	})
}
