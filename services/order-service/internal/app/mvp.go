package app

import (
	"errors"
	"github.com/omamx/order-service/internal/domain"
)

func validateRelease(c Config) error {
	if c.ReleaseProfile != "" && c.ReleaseProfile != "sandbox3.1" && c.ReleaseProfile != "mvp1" {
		return errors.New("unknown ORDER_RELEASE_PROFILE")
	}
	if c.ReleaseProfile == "mvp1" {
		if c.AutoComplete {
			return errors.New("mvp1 does not enable auto completion")
		}
		return c.SePay.Validate()
	}
	return nil
}
func (s *Service) MVP() bool { return s.C.ReleaseProfile == "mvp1" }
func (s *Service) checkoutPolicy(p domain.Principal, method string) error {
	if s.MVP() {
		if p.Role != "CUSTOMER" {
			return domain.E("CUSTOMER_LOGIN_REQUIRED", 403)
		}
		if method != "VIETQR" {
			return domain.E("PAYMENT_METHOD_DISABLED", 422)
		}
	}
	return nil
}
