package persistence

import (
	"testing"
	"time"
)

func TestReconcileDisposition(t *testing.T) {
	cases := []struct {
		name, opStatus, opError, proofState string
		proofExpired, stageExpired          bool
		jobAttempts                         int
		want                                string
	}{
		{"succeeded is stale", "SUCCEEDED", "", "RESERVED", false, false, 3, reconcileStale},
		{"failed is stale", "FAILED", "", "COMMITTED", false, false, 3, reconcileStale},
		{"recovered by admin is stale", "WAITING_RETRY", "", "RELEASED", false, false, 3, reconcileStale},
		{"reserved valid with budget resumes", "MANUAL_REVIEW", "CHECKOUT_DEPENDENCY_RETRY", "RESERVED", false, false, 3, reconcileResume},
		{"reserved valid but budget spent stays for admin", "MANUAL_REVIEW", "CHECKOUT_DEPENDENCY_RETRY", "RESERVED", false, false, 20, reconcileReview},
		{"reserved lapsed must release first", "MANUAL_REVIEW", "CHECKOUT_DEPENDENCY_RETRY", "RESERVED", true, false, 3, reconcileRelease},
		{"reserved but stage window closed stays for admin", "MANUAL_REVIEW", "CHECKOUT_DEPENDENCY_RETRY", "RESERVED", false, true, 3, reconcileReview},
		{"deadline park never auto resumes", "MANUAL_REVIEW", "STAGE_DEADLINE_EXCEEDED", "RESERVED", false, false, 3, reconcileReview},
		{"released compensates", "MANUAL_REVIEW", "CHECKOUT_DEPENDENCY_RETRY", "RELEASED", false, false, 20, reconcileCompensate},
		{"expired compensates", "MANUAL_REVIEW", "CHECKOUT_DEPENDENCY_RETRY", "EXPIRED", false, false, 20, reconcileCompensate},
		{"committed stays with a human", "MANUAL_REVIEW", "CHECKOUT_DEPENDENCY_RETRY", "COMMITTED", false, false, 3, reconcileReview},
		{"unknown state stays with a human", "MANUAL_REVIEW", "CHECKOUT_DEPENDENCY_RETRY", "SOMETHING_ELSE", false, false, 3, reconcileReview},
	}
	for _, c := range cases {
		if got := reconcileDisposition(c.opStatus, c.opError, c.proofState, c.proofExpired, c.stageExpired, c.jobAttempts); got != c.want {
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
