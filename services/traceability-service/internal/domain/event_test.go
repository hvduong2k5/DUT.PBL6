package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDistributedEventRejectsPrivateOrContradictoryPayload(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	profile := "22222222-2222-4222-8222-222222222222"
	trace, _ := NewTraceParent()
	for _, scenario := range []struct{ event, status string }{{"profile.published", "ACTIVE"}, {"profile.suspended", "SUSPENDED"}, {"profile.archived", "ARCHIVED"}, {"certification.status.changed", "ACTIVE"}} {
		t.Run(scenario.event, func(t *testing.T) {
			data := EventData{ProfileID: profile, AggregateVersion: 4, Revision: 1, Mode: "sandbox", Status: scenario.status}
			if strings.HasPrefix(scenario.event, "certification.") {
				eligible := false
				data.Eligible = &eligible
				data.ReasonCode = "CERTIFICATE_EXPIRED"
			}
			event := CloudEvent{SpecVersion: "1.0", ID: id, Source: "https://omama.vn/services/traceability-service", Type: "vn.omama.traceability." + scenario.event + ".v1", Subject: "profile:" + profile, Time: time.Now().UTC(), DataContentType: "application/json", TraceParent: trace, Data: data}
			body, _ := json.Marshal(event)
			message := OutboxMessage{ID: id, ProfileID: profile, Version: 4, Type: event.Type, Topic: "traceability.events.v1", Key: profile, Payload: body}
			if err := ValidateEvent(message); err != nil {
				t.Fatal("valid event rejected", err)
			}
			var raw map[string]any
			_ = json.Unmarshal(body, &raw)
			raw["data"].(map[string]any)["internal_evidence"] = "private/document"
			message.Payload, _ = json.Marshal(raw)
			if err := ValidateEvent(message); err == nil {
				t.Fatal("private field would be sent to broker")
			}
			raw["data"].(map[string]any)["status"] = "WRONG"
			delete(raw["data"].(map[string]any), "internal_evidence")
			message.Payload, _ = json.Marshal(raw)
			if err := ValidateEvent(message); err == nil {
				t.Fatal("contradictory status accepted")
			}
		})
	}
}
