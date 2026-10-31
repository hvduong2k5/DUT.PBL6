package domain

import (
	"context"

	"github.com/google/uuid"
)

// CustomerRepository định nghĩa các thao tác lưu trữ Aggregate CustomerProfile
type CustomerRepository interface {
	Create(ctx context.Context, c *CustomerProfile) error
	GetByID(ctx context.Context, id uuid.UUID) (*CustomerProfile, error)
	GetByUserID(ctx context.Context, id uuid.UUID) (*CustomerProfile, error)
	UpdateProfileWithOptimisticLock(
		ctx context.Context,
		id uuid.UUID,
		fullName string,
		phone string,
		email string,
		expectedVersion int,
	) (*CustomerProfile, error)
}

// AddressRepository định nghĩa các thao tác lưu trữ ShippingAddress
type AddressRepository interface {
	CreateAddress(ctx context.Context, addr *ShippingAddress) error
	GetAddressByID(ctx context.Context, id uuid.UUID) (*ShippingAddress, error)
	GetOwnedAddress(ctx context.Context, customerID, addressID uuid.UUID) (*ShippingAddress, error)
	UpdateAddress(ctx context.Context, address *ShippingAddress) error
	DeleteAddressWithVersion(ctx context.Context, customerID, addressID uuid.UUID, version int) error
	SwitchDefaultAddressWithVersion(ctx context.Context, customerID, addressID uuid.UUID, version int) error
	GetDefaultAddress(ctx context.Context, customerID uuid.UUID) (*ShippingAddress, error)
	ListAddressesByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*ShippingAddress, error)
	SwitchDefaultAddress(ctx context.Context, customerID, newDefaultAddressID uuid.UUID) error
	CountDefaultAddresses(ctx context.Context, customerID uuid.UUID) (int, error)
	DeleteAddress(ctx context.Context, customerID, addressID uuid.UUID) error
}

// EmployeeRepository định nghĩa các thao tác lưu trữ EmployeeProfile
type EmployeeRepository interface {
	CreateEmployee(ctx context.Context, emp *EmployeeProfile) error
	GetEmployeeByID(ctx context.Context, id uuid.UUID) (*EmployeeProfile, error)
	ListEmployees(ctx context.Context, deptID, status string, limit, offset int) ([]*EmployeeProfile, error)
	FindExpiringCertificates(ctx context.Context, thresholdDays int) ([]*EmployeeProfile, error)
}

// GuestClaimRepository định nghĩa các thao tác lưu trữ GuestOrderClaim
type GuestClaimRepository interface {
	CreateClaim(ctx context.Context, customerID uuid.UUID, orderID, phoneNumber string) error
	RevokeClaim(ctx context.Context, orderID string) error
	GetActiveClaim(ctx context.Context, orderID string) (*GuestOrderClaim, error)
}
