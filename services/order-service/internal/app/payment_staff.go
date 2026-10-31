package app

import (
	"encoding/base64"
	"encoding/json"
	"github.com/omamx/order-service/internal/persistence"
	"strings"
	"time"
)

func (s *Service) PaymentStaff(bearer string) (persistence.PaymentActor, error) {
	principal, e := s.Authenticate(bearer)
	if e != nil {
		return persistence.PaymentActor{}, e
	}
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
