package persistence

import (
	"testing"
	"time"
)

func TestReconcileDisposition(t *testing.T) {
	cases := []struct {
		name, opStatus, proofState string
		proofExpired               bool
		want                       string
	}{
		{"terminal succeeded is stale", "SUCCEEDED", "RESERVED", false, reconcileStale},
		{"terminal failed is stale", "FAILED", "COMMITTED", false, reconcileStale},
		{"reserved and valid resumes", "MANUAL_REVIEW", "RESERVED", false, reconcileResume},
		{"reserved but lapsed compensates", "MANUAL_REVIEW", "RESERVED", true, reconcileCompensate},
		{"released compensates", "MANUAL_REVIEW", "RELEASED", false, reconcileCompensate},
		{"expired compensates", "MANUAL_REVIEW", "EXPIRED", false, reconcileCompensate},
		{"committed stays with a human", "MANUAL_REVIEW", "COMMITTED", false, reconcileReview},
		{"unknown state stays with a human", "MANUAL_REVIEW", "SOMETHING_ELSE", false, reconcileReview},
	}
	for _, c := range cases {
		if got := reconcileDisposition(c.opStatus, c.proofState, c.proofExpired); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

func TestStageDeadlineExceeded(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	if stageDeadlineExceeded(nil, now) {
		t.Error("nil deadline must never expire")
	}
	future := now.Add(time.Minute)
	if stageDeadlineExceeded(&future, now) {
		t.Error("future deadline must not expire")
	}
	past := now.Add(-time.Minute)
	if !stageDeadlineExceeded(&past, now) {
		t.Error("past deadline must expire")
	}
	if !stageDeadlineExceeded(&now, now) {
		t.Error("deadline equal to now counts as exceeded (budget fully consumed)")
	}
}
