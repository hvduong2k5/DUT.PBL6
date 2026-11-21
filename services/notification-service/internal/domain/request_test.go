package domain

import "testing"

func TestIntentAndFingerprint(t *testing.T) {
	a := Command{SourceEvent: SourceEvent{Source: "order-service", ID: "event-1"}, Purpose: "ORDER_CONFIRMATION", Audience: "BUYER", RecipientRef: "recipient-1", ContextRef: "context-v1"}
	b := a
	b.ContextRef = "context-v2"
	if a.Intent() != b.Intent() || a.Fingerprint() == b.Fingerprint() {
		t.Fatal("changed context must conflict with original intent")
	}
	b = a
	b.Audience = "GIFT_RECIPIENT"
	if a.Intent() == b.Intent() {
		t.Fatal("audience isolation lost")
	}
}
