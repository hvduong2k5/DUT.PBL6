package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omamx/profile-service/internal/domain"
	"github.com/omamx/profile-service/internal/repository"
)

type ClaimOrderRequest struct {
	OrderID     string `json:"order_id"`
	PhoneNumber string `json:"phone_number"`
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
		return nil, fmt.Errorf("order_id and phone_number are required")
	}

	// 1. Lưu bản ghi claim vào database (bảo vệ bởi Partial Unique Index uq_order_claim_active)
	if err := u.claimRepo.CreateClaim(ctx, customerID, req.OrderID, req.PhoneNumber); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	// 2. Ghi Outbox event vn.omama.profile.guest_order_claimed.v1 để đồng bộ sang MS-04 và MS-07
	payload, _ := json.Marshal(map[string]any{
		"order_id":     req.OrderID,
		"customer_id":  customerID.String(),
		"phone_number": req.PhoneNumber,
		"claimed_at":   now.Format(time.RFC3339),
	})

	outboxQuery := `
		INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, topic)
		VALUES ($1, 'GuestOrderClaim', $2, 'vn.omama.profile.guest_order_claimed.v1', $3, 'profile.events.v1')
	`
	_, _ = u.pool.Exec(ctx, outboxQuery, uuid.New(), req.OrderID, payload)

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
