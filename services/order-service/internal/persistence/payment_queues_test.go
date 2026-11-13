package persistence

import "testing"

// F-29 regression: SweepManualRefunds joins refunds with receipts, and both
// tables carry a version column. The sweep's UPDATE once incremented an
// unqualified `version`, which PostgreSQL rejects as ambiguous
// (SQLSTATE 42702) on every run — the retry loop then logged only its label,
// so the failure repeated every 30s with no diagnosable message. Running the
// sweep against the real schema, even with no refunds present, proves the
// statement resolves.
func TestSweepManualRefundsResolvesAgainstSchema(t *testing.T) {
	h, p, _, _ := paymentFixture(t)
	if e := p.SweepManualRefunds(h.ctx); e != nil {
		t.Fatal(e)
	}
}
