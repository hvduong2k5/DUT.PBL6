package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/repository"
)

type ClaimOrderRequest struct {
	OrderID           string `json:"order_id"`
	PhoneNumber       string `json:"phone_number"`
	VerificationToken string `json:"verification_token"`
}

type ClaimOrderResponse struct {
	Message    string    `json:"message"`
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	ClaimedAt  time.Time `json:"claimed_at"`
}

type ClaimUsecase struct {
	claimRepo *repository.ClaimRepository
	pool      *pgxpool.Pool
	verifier  GuestClaimVerifier
}

func NewClaimUsecase(claimRepo *repository.ClaimRepository, pool *pgxpool.Pool) *ClaimUsecase {
	return &ClaimUsecase{
		claimRepo: claimRepo,
		pool:      pool,
	}
}

// ClaimGuestOrder executes Use Case 5: Link Guest Order & Emit Kafka Outbox Event.
func (u *ClaimUsecase) ClaimGuestOrder(ctx context.Context, customerID uuid.UUID, req ClaimOrderRequest) (*ClaimOrderResponse, error) {
	if req.OrderID == "" || req.PhoneNumber == "" {
		return nil, &domain.ValidationError{Field: "order_id", Message: "order_id and phone_number are required"}
	}
	if err := domain.ValidateText("order_id", req.OrderID, 50); err != nil {
		return nil, err
	}

	if u.verifier == nil {
		return nil, domain.ErrClaimVerificationUnavailable
	}
	phone, err := domain.NormalizePhone(req.PhoneNumber)
	if err != nil {
		return nil, err
	}
	if req.VerificationToken == "" {
		return nil, &domain.ValidationError{Field: "verification_token", Message: "required"}
	}
	if err = u.verifier.VerifyGuestOrderOwnership(ctx, customerID, req.OrderID, phone, req.VerificationToken); err != nil {
		return nil, err
	}
	if err = u.claimRepo.CreateClaim(ctx, customerID, req.OrderID, phone); err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	return &ClaimOrderResponse{
		Message:    "Guest order claimed successfully",
		OrderID:    req.OrderID,
		CustomerID: customerID.String(),
		ClaimedAt:  now,
	}, nil
}

// GetActiveClaim tra cứu thông tin claim hiện tại của 1 đơn hàng
func (u *ClaimUsecase) GetActiveClaim(ctx context.Context, orderID string) (*domain.GuestOrderClaim, error) {
	return u.claimRepo.GetActiveClaim(ctx, orderID)
}

// RevokeClaim hủy quyền claim (dành cho Admin giải quyết tranh chấp)
func (u *ClaimUsecase) RevokeClaim(ctx context.Context, orderID string) error {
	return u.claimRepo.RevokeClaim(ctx, orderID)
}

// Implementations must validate OTP expiry/replay, bind proof to customer/order/phone,
// and confirm the order is still a guest order with the same phone through Order API.
type GuestClaimVerifier interface {
	VerifyGuestOrderOwnership(context.Context, uuid.UUID, string, string, string) error
}

func (u *ClaimUsecase) SetVerifier(v GuestClaimVerifier) { u.verifier = v }
func (u *ClaimUsecase) GetOwnedClaim(ctx context.Context, customerID uuid.UUID, orderID string) (*domain.GuestOrderClaim, error) {
	claim, err := u.claimRepo.GetActiveClaim(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if claim.CustomerID != customerID {
		return nil, domain.ErrClaimNotFound
	}
	return claim, nil
}
