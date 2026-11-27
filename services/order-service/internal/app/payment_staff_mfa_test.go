package app

import (
	"testing"
	"time"
)

// F-42 MFA follow-up: the write tier opens only for a token that proves
// a real step-up (Keycloak LoA 2 = password + TOTP) completed inside the
// freshness window. A fresh password-only login — accepted under the
// original time-boxed exception — no longer counts.

func TestPaymentStepUpFresh(t *testing.T) {
	now := time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Minute)
	cases := []struct {
		name string
		acr  string
		amr  []string
		at   time.Time
		want bool
	}{
		{"acr and otp fresh", "2", []string{"pwd", "otp"}, fresh, true},
		{"acr alone fresh", "2", []string{"pwd"}, fresh, true},
		{"otp alone fresh", "", []string{"pwd", "otp"}, fresh, true},
		{"hardware key fresh", "", []string{"pwd", "hwk"}, fresh, true},
		{"password only fresh", "1", []string{"pwd"}, fresh, false},
		{"no claims fresh time", "", []string{"pwd"}, fresh, false},
		{"step-up gone stale", "2", []string{"pwd", "otp"}, now.Add(-16 * time.Minute), false},
		{"step-up at the window edge", "2", []string{"pwd", "otp"}, now.Add(-15 * time.Minute), true},
		{"no auth time", "2", []string{"pwd", "otp"}, time.Time{}, false},
		{"auth time in the future", "2", []string{"pwd", "otp"}, now.Add(2 * time.Minute), false},
		{"named acr without otp", "gold", []string{"pwd"}, fresh, false},
		{"named acr with otp", "gold", []string{"otp"}, fresh, true},
	}
	for _, c := range cases {
		if got := paymentStepUpFresh(c.acr, c.amr, c.at, now); got != c.want {
			t.Errorf("%s: paymentStepUpFresh(%q, %v) = %v, want %v", c.name, c.acr, c.amr, got, c.want)
		}
	}
}
