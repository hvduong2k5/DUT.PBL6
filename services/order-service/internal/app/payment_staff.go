package app

import (
	"encoding/base64"
	"encoding/json"
	"github.com/omamx/order-service/internal/domain"
	"github.com/omamx/order-service/internal/persistence"
	"strings"
	"time"
)

func (s *Service) PaymentStaff(bearer string) (persistence.PaymentActor, error) {
	principal, e := s.Authenticate(bearer)
	if e != nil {
		return persistence.PaymentActor{}, e
	}
	if principal.IAM {
		return s.paymentStaffIAM(bearer, principal)
	}
	// The legacy payment staff protocol (connection/receiver claims on a
	// sandbox token) stays available for non-IAM tokens only.
	if !principal.Staff() {
		return persistence.PaymentActor{}, persistence.ErrForbidden
	}
	// Authenticate already validated this exact JWT's MAC/issuer/audience/expiry.
	parts := strings.Split(strings.TrimPrefix(bearer, "Bearer "), ".")
	raw, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		return persistence.PaymentActor{}, e
	}
	var claims Claims
	if json.Unmarshal(raw, &claims) != nil {
		return persistence.PaymentActor{}, persistence.ErrForbidden
	}
	actor := persistence.PaymentActor{ID: principal.ID, Connection: claims.PaymentConnection, Receivers: claims.PaymentReceivers, Permissions: map[string]bool{}}
	for _, permission := range claims.Permissions {
		actor.Permissions[permission] = true
	}
	mfa, pwd, otp := false, false, false
	for _, method := range claims.AMR {
		switch method {
		case "mfa":
			mfa = true
		case "pwd":
			pwd = true
		case "otp", "hwk":
			otp = true
		}
	}
	if claims.MFAAt > 0 && (mfa || pwd && otp) {
		actor.MFA = time.Unix(claims.MFAAt, 0)
	}
	return actor, nil
}

// paymentStaffPermissions is the explicit, fail-closed translation from IAM
// payment grants (the identity permission universe, F-42) to this service's
// internal payment permission codes. Only GLOBAL grants map. PAYMENT_VIEW is
// intentionally absent: it only opens the admin web page and must never
// authorize an API read. PAYMENT_SENSITIVE_VIEW maps to nothing by itself;
// together with PAYMENT_REFUND_VIEW it unlocks the refund queue, whose views
// carry beneficiary bank data. PAYMENT_CASE_DECIDE joins the write tier
// under the same auth_time freshness rule: it records the resolution
// decision that a different holder then consumes (F-55 maker-checker).
// The refund tiers (F-42 slice 2) join the same write tier: approve and
// accept for PAYMENT_REFUND_APPROVE, execute plus evidence attachment for
// PAYMENT_REFUND_EXECUTE, verify plus evidence issuance for
// PAYMENT_REFUND_VERIFY. No tier implies another; the persistence layer
// separates the duties per person (approver never executes, verifier is
// neither approver nor executor, issuer is neither approver nor executor).
func paymentStaffPermissions(grants []domain.PermissionGrant, writeFresh bool) map[string]bool {
	iam := map[string]bool{}
	for _, g := range grants {
		if g.ScopeType == "GLOBAL" {
			iam[g.PermissionCode] = true
		}
	}
	perms := map[string]bool{}
	if iam["PAYMENT_RECONCILIATION_VIEW"] {
		perms["payment.reconciliation.read"] = true
	}
	if iam["PAYMENT_REFUND_VIEW"] && iam["PAYMENT_SENSITIVE_VIEW"] {
		perms["payment.refund.read"] = true
	}
	if writeFresh && iam["PAYMENT_CASE_RESOLVE"] {
		perms["payment.reconciliation.manage"] = true
	}
	if writeFresh && iam["PAYMENT_CASE_DECIDE"] {
		perms["payment.case.decide"] = true
	}
	if writeFresh && iam["PAYMENT_REFUND_APPROVE"] {
		perms["payment.refund.approve"] = true
		perms["payment.refund.accept"] = true
	}
	if writeFresh && iam["PAYMENT_REFUND_EXECUTE"] {
		perms["payment.refund.execute"] = true
		perms["payment.evidence.write"] = true
	}
	if writeFresh && iam["PAYMENT_REFUND_VERIFY"] {
		perms["payment.refund.verify"] = true
		perms["payment.evidence.issue"] = true
	}
	return perms
}

// paymentAuthTime extracts the auth_time claim from an access token whose
// signature Authenticate has already verified. A zero result means the token
// carries no usable auth_time; the write tier then stays unmapped (fail
// closed) instead of falling back to the token's issue time, which a refresh
// would silently renew.
func paymentAuthTime(bearer string) time.Time {
	parts := strings.Split(strings.TrimPrefix(bearer, "Bearer "), ".")
	if len(parts) != 3 {
		return time.Time{}
	}
	raw, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		return time.Time{}
	}
	var claims Claims
	if json.Unmarshal(raw, &claims) != nil || claims.MFAAt <= 0 {
		return time.Time{}
	}
	return time.Unix(claims.MFAAt, 0)
}

// paymentStaffIAM builds the payment actor for a Keycloak (IAM) staff
// principal from its signed permission grants (F-42, option c). The actor is
// bound to this service's single configured SePay connection and receiver
// profile — the same values the payment store enforces on every query — and
// never to a wildcard. The write tier (case manage) additionally requires a
// fresh auth_time (<= 15 minutes, the legacy protocol's window); this is a
// recorded, time-boxed exception, not an MFA substitute: the realm has no
// MFA enrolled yet, and a token without auth_time gets no write tier.
func (s *Service) paymentStaffIAM(bearer string, principal domain.Principal) (persistence.PaymentActor, error) {
	if !principal.Staff() || s.Payment == nil || s.Payment.Connection == "" || s.Payment.Receiver == "" {
		return persistence.PaymentActor{}, persistence.ErrForbidden
	}
	authTime := paymentAuthTime(bearer)
	now := time.Now()
	writeFresh := !authTime.IsZero() && !authTime.After(now.Add(time.Minute)) && now.Sub(authTime) <= 15*time.Minute
	perms := paymentStaffPermissions(principal.Grants, writeFresh)
	if len(perms) == 0 {
		return persistence.PaymentActor{}, persistence.ErrForbidden
	}
	return persistence.PaymentActor{
		ID:          principal.ID,
		Connection:  s.Payment.Connection,
		Receivers:   []string{s.Payment.Receiver},
		Permissions: perms,
		MFA:         authTime,
	}, nil
}
