package domain

import (
	"time"

	"github.com/google/uuid"
)

// GuestOrderClaimStatus là các trạng thái của Guest Order Claim FSM
type GuestOrderClaimStatus string

const (
	ClaimStatusInitiated   GuestOrderClaimStatus = "INITIATED"
	ClaimStatusOTPVerified GuestOrderClaimStatus = "OTP_VERIFIED"
	ClaimStatusClaimed     GuestOrderClaimStatus = "CLAIMED"
	ClaimStatusExpired     GuestOrderClaimStatus = "EXPIRED"
	ClaimStatusRejected    GuestOrderClaimStatus = "REJECTED"
	ClaimStatusVerified    GuestOrderClaimStatus = "VERIFIED"
	ClaimStatusRevoked     GuestOrderClaimStatus = "REVOKED"
)

// GuestOrderClaim là Entity đại diện cho yêu cầu liên kết đơn hàng vãng lai
type GuestOrderClaim struct {
	ID               uuid.UUID  `json:"id"`
	CustomerID       uuid.UUID  `json:"customer_id"`
	OrderID          string     `json:"order_id"`
	PhoneNumber      string     `json:"phone_number"`
	ClaimStatus      string     `json:"claim_status"` // 'INITIATED', 'OTP_VERIFIED', 'CLAIMED', 'VERIFIED', 'REVOKED'
	VerificationCode string     `json:"-"`
	VerifiedAt       *time.Time `json:"verified_at,omitempty"`
	ExpiresAt        time.Time  `json:"expires_at"`
	ClaimedAt        time.Time  `json:"claimed_at"`
}
