package domain

import (
	"time"

	"github.com/google/uuid"
)

type CustomerProfile struct {
	ID          uuid.UUID         `json:"id"`
	UserID      uuid.UUID         `json:"user_id"`
	FullName    string            `json:"full_name"`
	PhoneNumber string            `json:"phone_number"`
	Email       string            `json:"email"`
	DateOfBirth *time.Time        `json:"date_of_birth,omitempty"`
	Gender      string            `json:"gender"`
	AvatarURL   string            `json:"avatar_url"`
	Preferences map[string]any    `json:"preferences"`
	Status      string            `json:"status"`
	Version     int               `json:"version"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type ShippingAddress struct {
	ID            uuid.UUID `json:"id"`
	CustomerID    uuid.UUID `json:"customer_id"`
	RecipientName string    `json:"recipient_name"`
	PhoneNumber   string    `json:"phone_number"`
	StreetAddress string    `json:"street_address"`
	WardCode      string    `json:"ward_code"`
	WardName      string    `json:"ward_name"`
	ProvinceCode  string    `json:"province_code"`
	ProvinceName  string    `json:"province_name"`
	Latitude      *float64  `json:"latitude,omitempty"`
	Longitude     *float64  `json:"longitude,omitempty"`
	Label         string    `json:"label"`
	IsDefault     bool      `json:"is_default"`
	IsDeleted     bool      `json:"is_deleted"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type GuestOrderClaim struct {
	ID          uuid.UUID `json:"id"`
	CustomerID  uuid.UUID `json:"customer_id"`
	OrderID     string    `json:"order_id"`
	PhoneNumber string    `json:"phone_number"`
	ClaimStatus string    `json:"claim_status"` // 'VERIFIED' | 'REVOKED'
	ClaimedAt   time.Time `json:"claimed_at"`
}

type EmployeeProfile struct {
	ID                   uuid.UUID  `json:"id"`
	UserID               *uuid.UUID `json:"user_id,omitempty"`
	EmployeeCode         string     `json:"employee_code"`
	FullName             string     `json:"full_name"`
	PhoneNumber          string     `json:"phone_number"`
	IDCardEncrypted      []byte     `json:"-"`
	IDCardNonce          []byte     `json:"-"`
	EncryptedDEK         []byte     `json:"-"`
	KEKVersion           int        `json:"kek_version"`
	DepartmentID         string     `json:"department_id"`
	Position             string     `json:"position"`
	ContractType         string     `json:"contract_type"`
	ContractStartDate    time.Time  `json:"contract_start_date"`
	ContractEndDate      *time.Time `json:"contract_end_date,omitempty"`
	FoodSafetyCertNo     string     `json:"food_safety_cert_no"`
	FoodSafetyCertExpiry *time.Time `json:"food_safety_cert_expiry,omitempty"`
	Status               string     `json:"status"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}
